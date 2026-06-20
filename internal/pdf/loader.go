// Package pdf is the pdfdisassembler-backed implementation of
// model.Document. It is the only package in the codebase that imports a
// concrete PDF parser; the rest of the project depends on model
// interfaces.
package pdf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// Load parses a PDF from rs.
func Load(rs io.ReadSeeker) (model.Document, error) {
	r, err := pdd.Open(rs)
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}
	d := &document{r: r}
	d.loadRoleMap()
	d.loadPageIndex()
	return d, nil
}

// LoadFile opens path and parses it. The underlying file stays open for
// the lifetime of the returned Document.
func LoadFile(path string) (model.Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r, err := pdd.Open(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	d := &document{r: r, closer: f}
	d.loadRoleMap()
	d.loadPageIndex()
	return d, nil
}

// document is the pdfdisassembler-backed implementation of model.Document.
type document struct {
	r                   *pdd.Reader
	closer              io.Closer
	roleMap             map[string]string     // populated by loadRoleMap; may be empty
	pageIndex           map[pdd.Reference]int // page-ref -> 1-based page number
	pageReports         []model.PageReport    // cached result of Pages(); nil until first call
	annotations         []model.Annotation    // cached result of Annotations()
	annotationsLoaded   bool                  // tracks Annotations() cache (nil-slice is a valid value)
	cachedClassMap      *pdd.Dict             // /StructTreeRoot/ClassMap, lazily resolved by classMap()
	cachedClassMapKnown bool                  // distinguishes "not resolved yet" from "resolved, absent"
}

// loadRoleMap reads StructTreeRoot/RoleMap and caches custom-to-standard
// structure-type name mappings.
func (d *document) loadRoleMap() {
	d.roleMap = map[string]string{}
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return
	}
	rm, ok := stree.Dict("RoleMap")
	if !ok {
		return
	}
	for k, v := range rm.Iter() {
		if n, ok := v.(pdd.Name); ok {
			d.roleMap[k] = string(n)
		}
	}
}

// loadPageIndex walks the page tree once and records each leaf Page
// object's 1-based number. Used by StructElement.Page() to translate
// the /Pg reference on a structure element into a human-readable page
// number. Best-effort: malformed page trees just leave the map empty.
func (d *document) loadPageIndex() {
	d.pageIndex = map[pdd.Reference]int{}
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return
	}
	pagesObj, ok := cat.Get("Pages")
	if !ok {
		return
	}
	ref, ok := pagesObj.(pdd.Reference)
	if !ok {
		return
	}
	counter := 0
	d.walkPageTree(ref, &counter)
}

func (d *document) walkPageTree(ref pdd.Reference, counter *int) {
	obj, err := d.r.Resolve(ref)
	if err != nil {
		return
	}
	dict, ok := obj.(*pdd.Dict)
	if !ok {
		return
	}
	typeName, _ := dict.Name("Type")
	switch typeName {
	case "Page":
		*counter++
		d.pageIndex[ref] = *counter
	case "Pages":
		kids, ok := dict.Array("Kids")
		if !ok {
			return
		}
		for _, kid := range kids {
			kidRef, ok := kid.(pdd.Reference)
			if !ok {
				continue
			}
			d.walkPageTree(kidRef, counter)
		}
	}
}

// resolveRole follows a custom structure type through RoleMap to its
// standard PDF tag. Defensive against pathological cycles.
func (d *document) resolveRole(name string) string {
	for range 8 {
		mapped, ok := d.roleMap[name]
		if !ok || mapped == name {
			return name
		}
		name = mapped
	}
	return name
}

// PageCount returns the number of page leaves discovered during
// loadPageIndex.
func (d *document) PageCount() int { return len(d.pageIndex) }

// Catalog returns the catalog wrapped in a model.Dict.
func (d *document) Catalog() (model.Dict, error) {
	c, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	return dict{c}, nil
}

// DocInfo returns the DocumentInfo /Title (other fields are added on
// demand). pdfdisassembler reads /Info only -- no XMP fallback, which is
// the stricter (and spec-correct) behaviour.
func (d *document) DocInfo() model.DocInfo {
	return model.DocInfo{Title: d.r.DocumentInfo().Title}
}

func (d *document) DereferenceDict(obj model.Object) (model.Dict, error) {
	pobj, err := asObject(obj)
	if err != nil {
		return nil, err
	}
	pd, err := d.r.ResolveDict(pobj)
	if err != nil {
		return nil, err
	}
	if pd == nil {
		return nil, errors.New("object is not a dictionary")
	}
	return dict{pd}, nil
}

func (d *document) DereferenceBoolean(obj model.Object) (bool, error) {
	pobj, err := asObject(obj)
	if err != nil {
		return false, err
	}
	return d.r.ResolveBool(pobj)
}

func (d *document) DecodeStream(obj model.Object) ([]byte, error) {
	pobj, err := asObject(obj)
	if err != nil {
		return nil, err
	}
	return d.r.DecodeStream(pobj)
}

// dict wraps a pdfdisassembler.Dict to satisfy model.Dict. The wrapper
// exists only because Dict.Get returns the package's own Object
// interface, which is narrower than model.Object (any).
type dict struct {
	inner *pdd.Dict
}

func (d dict) Find(key string) (model.Object, bool) {
	obj, ok := d.inner.Get(key)
	if !ok {
		return nil, false
	}
	return obj, true
}

// asObject converts a model.Object back to the pdfdisassembler Object
// type the backend expects.
func asObject(obj model.Object) (pdd.Object, error) {
	if obj == nil {
		return nil, errors.New("nil PDF object")
	}
	pobj, ok := obj.(pdd.Object)
	if !ok {
		return nil, fmt.Errorf("not a pdfdisassembler object: %T", obj)
	}
	return pobj, nil
}

func (d *document) StructTreeRoot() (model.StructElement, error) {
	cat, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return nil, nil
	}
	kObj, ok := stree.Get("K")
	if !ok {
		return nil, nil
	}
	resolved, err := d.r.Resolve(kObj)
	if err != nil {
		return nil, err
	}
	switch v := resolved.(type) {
	case *pdd.Dict:
		return structElement{doc: d, dict: v}, nil
	case pdd.Array:
		// StructTreeRoot.K may be an array of top-level elements; use
		// the first that looks like a structure element. PDF authors
		// rarely emit this shape.
		for _, item := range v {
			cd, err := d.r.ResolveDict(item)
			if err != nil || cd == nil {
				continue
			}
			if _, hasS := cd.Get("S"); hasS {
				return structElement{doc: d, dict: cd}, nil
			}
		}
	}
	return nil, nil
}

// structElement implements model.StructElement on top of a pdfdisassembler
// Dict, with role-map resolution applied at Type().
type structElement struct {
	doc  *document
	dict *pdd.Dict
}

func (e structElement) Type() string {
	name, ok := e.dict.Name("S")
	if !ok {
		return ""
	}
	s := string(name)
	mapped, hasMapping := e.doc.roleMap[s]
	if !hasMapping || mapped == s {
		return s
	}
	// The element has a global /RoleMap entry. /RoleMap is the PDF 1.7
	// mechanism for mapping custom (sui-generis) types onto standard
	// ones; it must not override an element that already declares a
	// standard structure type in the PDF standard namespace. PDF 2.0
	// files routinely role-map new 2.0 types (Aside, FENote, ...) down
	// to 1.7 fallbacks (Note, ...) purely so legacy 1.7 viewers can
	// render them -- the declared 2.0 type is authoritative
	// (ISO 14289-2 §8.2.5.14 Note 1). Custom types, or standard names
	// used outside the standard namespace, still resolve through the
	// map.
	if model.IsStandardStructureType(s) && e.inStandardNamespace() {
		return s
	}
	return e.doc.resolveRole(s)
}

// inStandardNamespace reports whether this element lives in the
// default PDF structure namespace: no /NS at all, or one of the
// registered PDF 1.7 / PDF 2.0 structure namespace URIs
// (ISO 32000-2 §14.8.6.3). A standard structure type is only
// authoritative -- and thus exempt from global role mapping --
// within that namespace.
func (e structElement) inStandardNamespace() bool {
	switch e.Namespace() {
	case "", "http://iso.org/pdf/ssn", "http://iso.org/pdf2/ssn":
		return true
	}
	return false
}

func (e structElement) Children() []model.StructElement {
	kObj, ok := e.dict.Get("K")
	if !ok {
		return nil
	}
	resolved, err := e.doc.r.Resolve(kObj)
	if err != nil {
		return nil
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case *pdd.Dict:
		items = []pdd.Object{v}
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		// Integer MCID or other non-structure children -- skip.
		return nil
	}

	var children []model.StructElement
	for _, item := range items {
		childDict, err := e.doc.r.ResolveDict(item)
		if err != nil || childDict == nil {
			continue
		}
		// Structure elements always carry /S. Without it the dict is an
		// MCR, OBJR or some other non-structure child and we skip it.
		if _, hasS := childDict.Get("S"); !hasS {
			continue
		}
		children = append(children, structElement{doc: e.doc, dict: childDict})
	}
	return children
}

// Content walks /K in order and returns the element's content as an
// interleaved sequence of text runs and child structure elements:
//   - bare integer MCID -> text on the element's own /Pg
//   - {Type:MCR, Pg, MCID} dict -> text on the MCR's /Pg (or the
//     element's own /Pg when the MCR omits /Pg)
//   - dict with /S -> a child structure element
//   - OBJR / other -> skipped
//
// Text runs that decode to empty are dropped so callers do not have to
// filter them. Children() and Text() are the element- and text-only
// projections of this sequence.
func (e structElement) Content() []model.ContentItem {
	kObj, ok := e.dict.Get("K")
	if !ok {
		return nil
	}
	resolved, err := e.doc.r.Resolve(kObj)
	if err != nil {
		return nil
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		items = []pdd.Object{v}
	}

	// Page for bare-integer MCID children: the element's own /Pg. MCR
	// children may override it with their own /Pg.
	var ownPg pdd.Reference
	if pg, ok := e.dict.Get("Pg"); ok {
		if ref, ok := pg.(pdd.Reference); ok {
			ownPg = ref
		}
	}

	var out []model.ContentItem
	addText := func(pg pdd.Reference, mcid int) {
		if t := e.doc.mcidText(pg, mcid); t != "" {
			out = append(out, model.ContentItem{Text: t})
		}
	}
	for _, item := range items {
		if n, ok := asInt(item); ok {
			addText(ownPg, int(n))
			continue
		}
		childDict, err := e.doc.r.ResolveDict(item)
		if err != nil || childDict == nil {
			continue
		}
		// Nested structure elements carry their own content.
		if _, hasS := childDict.Get("S"); hasS {
			out = append(out, model.ContentItem{Element: structElement{doc: e.doc, dict: childDict}})
			continue
		}
		if name, _ := childDict.Name("Type"); string(name) != "MCR" {
			continue // OBJR or other non-content child
		}
		mcid, ok := childDict.Int("MCID")
		if !ok {
			continue
		}
		pg := ownPg
		if p, ok := childDict.Get("Pg"); ok {
			if ref, ok := p.(pdd.Reference); ok {
				pg = ref
			}
		}
		addText(pg, int(mcid))
	}
	return out
}

// Text concatenates the text runs of Content() in /K order, skipping the
// child structure elements (which carry their own Text()). The result is
// the page text drawn directly under this element's own marked content.
func (e structElement) Text() string {
	var b strings.Builder
	for _, it := range e.Content() {
		if it.Element == nil {
			b.WriteString(it.Text)
		}
	}
	return b.String()
}

// mcidText returns the decoded content-stream text recorded for one MCID
// on the page identified by ref. Returns "" when ref is the zero
// reference, the page is unknown, or the page carries no such MCID.
func (d *document) mcidText(ref pdd.Reference, mcid int) string {
	if ref == (pdd.Reference{}) {
		return ""
	}
	num := d.pageIndex[ref]
	if num == 0 {
		return ""
	}
	reports, err := d.Pages()
	if err != nil || num > len(reports) {
		return ""
	}
	return reports[num-1].MCIDText[mcid]
}

func (e structElement) Attr(name string) string {
	// Dict.String already handles indirect-ref resolution and decodes
	// the PDF text-string encoding (PDFDocEncoded / UTF-16BE / UTF-8).
	s, _ := e.dict.String(name)
	return s
}

func (e structElement) AttrPresent(name string) bool {
	_, ok := e.dict.String(name)
	return ok
}

// Attribute pulls a Name value out of the structure element's
// attribute owner dictionaries. PDF gives two equivalent ways to
// attach attributes to a structure element (ISO 32000-1
// §14.7.5.3/4): the /A entry on the element itself, holding either
// a single attribute-owner dict or an array of them; and the /C
// entry naming one or more class names that resolve against the
// StructTreeRoot's /ClassMap to attribute-owner dicts. /A and /C
// are searched in that order and the first hit wins. Returns ""
// when neither carries the requested attribute, or when the value
// is not a Name.
//
// Numeric and other non-name attribute values are not surfaced
// here; add a typed accessor per data shape if a check needs one.
func (e structElement) Attribute(name string) string {
	if v := nameInAttrOwner(e.doc, e.dict.Get, "A", name); v != "" {
		return v
	}
	return e.doc.attributeFromClass(e.dict, name)
}

// nameInAttrOwner reads key on dict (via getFn), resolves it as an
// attribute-owner dict or array of dicts, and returns the first
// Name value found under attrName. The indirection through getFn
// is so the same helper can read /A from a struct element and
// /<className> from a ClassMap dict.
func nameInAttrOwner(d *document, getFn func(string) (pdd.Object, bool), key, attrName string) string {
	obj, ok := getFn(key)
	if !ok {
		return ""
	}
	resolved, err := d.r.Resolve(obj)
	if err != nil {
		return ""
	}
	switch v := resolved.(type) {
	case *pdd.Dict:
		if n, ok := v.Name(attrName); ok {
			return string(n)
		}
	case pdd.Array:
		for _, item := range v {
			itemDict, err := d.r.ResolveDict(item)
			if err != nil || itemDict == nil {
				continue
			}
			if n, ok := itemDict.Name(attrName); ok {
				return string(n)
			}
		}
	}
	return ""
}

// attributeFromClass walks the element's /C entry (a single Name or
// an array of Names), looks each class up in the document's
// /ClassMap, and returns the first Name value found for attrName in
// the resolved attribute-owner dict.
func (d *document) attributeFromClass(elemDict *pdd.Dict, attrName string) string {
	cObj, ok := elemDict.Get("C")
	if !ok {
		return ""
	}
	classMap, ok := d.classMap()
	if !ok {
		return ""
	}
	classes := classNamesFrom(cObj, d)
	for _, cls := range classes {
		if v := nameInAttrOwner(d, classMap.Get, cls, attrName); v != "" {
			return v
		}
	}
	return ""
}

// classMap returns the StructTreeRoot /ClassMap dict, cached on the
// document. Returns (nil, false) when the catalog has no
// StructTreeRoot or no ClassMap.
func (d *document) classMap() (*pdd.Dict, bool) {
	if d.cachedClassMapKnown {
		return d.cachedClassMap, d.cachedClassMap != nil
	}
	d.cachedClassMapKnown = true
	cat, err := d.r.Catalog()
	if err != nil {
		return nil, false
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return nil, false
	}
	cm, ok := stree.Dict("ClassMap")
	if !ok {
		return nil, false
	}
	d.cachedClassMap = cm
	return cm, true
}

// classNamesFrom decodes /C into the list of class names it carries.
// /C may be a single Name or an array of Names; both forms collapse
// to []string.
func classNamesFrom(cObj pdd.Object, d *document) []string {
	resolved, err := d.r.Resolve(cObj)
	if err != nil {
		return nil
	}
	switch v := resolved.(type) {
	case pdd.Name:
		return []string{string(v)}
	case pdd.Array:
		out := make([]string, 0, len(v))
		for _, item := range v {
			r, err := d.r.Resolve(item)
			if err != nil {
				continue
			}
			if n, ok := r.(pdd.Name); ok {
				out = append(out, string(n))
			}
		}
		return out
	}
	return nil
}

// Namespace resolves the /NS entry on this struct element (or
// inherits it from /P ancestors) and returns the URI string from
// the referenced Namespace dictionary's own /NS entry. Returns ""
// when no /NS is declared anywhere up the chain or the resolution
// fails. The recursion is bounded to a small depth to avoid
// looping on malformed circular /P chains.
func (e structElement) Namespace() string {
	return e.doc.resolveNamespace(e.dict, 0)
}

// resolveNamespace walks /NS on dict, falling back to /P
// inheritance up to maxDepth levels. A namespace dict carries the
// URI under its own /NS string entry (ISO 32000-2 §14.8.6.3).
func (d *document) resolveNamespace(dict *pdd.Dict, depth int) string {
	const maxDepth = 32
	if depth > maxDepth {
		return ""
	}
	if nsObj, ok := dict.Get("NS"); ok {
		nsDict, err := d.r.ResolveDict(nsObj)
		if err == nil && nsDict != nil {
			if uri, ok := nsDict.String("NS"); ok {
				return uri
			}
		}
	}
	pObj, ok := dict.Get("P")
	if !ok {
		return ""
	}
	pDict, err := d.r.ResolveDict(pObj)
	if err != nil || pDict == nil {
		return ""
	}
	// Stop at the StructTreeRoot: it has /Type /StructTreeRoot, not
	// /StructElem, and does not participate in namespace inheritance.
	if t, ok := pDict.Name("Type"); ok && string(t) == "StructTreeRoot" {
		return ""
	}
	return d.resolveNamespace(pDict, depth+1)
}

// AssociatedFiles resolves the element's own /AF array to a slice of
// AssociatedFile snapshots. Used by UA-17-001 to detect MathML/LaTeX
// AFs attached directly to a Formula structure element (the BPG
// "Use of Associated files" pattern). Returns nil when /AF is
// missing or empty; filespec entries that fail to resolve are
// silently dropped.
func (e structElement) AssociatedFiles() []model.AssociatedFile {
	afObj, ok := e.dict.Get("AF")
	if !ok {
		return nil
	}
	arr, err := e.doc.r.ResolveArray(afObj)
	if err != nil {
		return nil
	}
	var out []model.AssociatedFile
	for _, item := range arr {
		fs, err := e.doc.r.ResolveDict(item)
		if err != nil || fs == nil {
			continue
		}
		af := model.AssociatedFile{SourcePath: "StructElement"}
		if uf, ok := fs.String("UF"); ok && uf != "" {
			af.Filename = uf
		} else if f, ok := fs.String("F"); ok {
			af.Filename = f
		}
		if rel, ok := fs.Name("AFRelationship"); ok {
			af.Relationship = string(rel)
		}
		af.Subtype = e.doc.embeddedFileSubtype(fs)
		af.Content = e.doc.embeddedFileContent(fs)
		out = append(out, af)
	}
	return out
}

// Refs resolves the element's /Ref array (indirect references on
// the StructElem dict) to the target StructElements. Targets that
// either fail to resolve or that are not structure elements (no /S)
// are dropped silently.
func (e structElement) Refs() []model.StructElement {
	refObj, ok := e.dict.Get("Ref")
	if !ok {
		return nil
	}
	arr, err := e.doc.r.ResolveArray(refObj)
	if err != nil {
		return nil
	}
	var out []model.StructElement
	for _, item := range arr {
		d, err := e.doc.r.ResolveDict(item)
		if err != nil || d == nil {
			continue
		}
		if _, hasS := d.Get("S"); !hasS {
			continue
		}
		out = append(out, structElement{doc: e.doc, dict: d})
	}
	return out
}

func (e structElement) Page() int {
	pgObj, ok := e.dict.Get("Pg")
	if !ok {
		return 0
	}
	ref, ok := pgObj.(pdd.Reference)
	if !ok {
		return 0
	}
	return e.doc.pageIndex[ref]
}

func (d *document) Fonts() ([]model.Font, error) {
	var fonts []model.Font
	for entry := range d.r.Objects() {
		fd, ok := entry.Object.(*pdd.Dict)
		if !ok {
			continue
		}
		typeName, ok := fd.Name("Type")
		if !ok || typeName != "Font" {
			continue
		}
		subtypeName, _ := fd.Name("Subtype")
		subtype := string(subtypeName)
		// Skip CID descendant fonts; their embedding state is folded
		// into the enclosing Type0 font.
		if subtype == "CIDFontType0" || subtype == "CIDFontType2" {
			continue
		}
		baseFont, _ := fd.Name("BaseFont")
		encName, hasDiff := d.fontEncoding(fd)
		hasToU := fd.Has("ToUnicode")
		isSym := d.fontIsSymbolic(fd, subtype)
		mappings, codeBytes := d.parseToUnicodeFromFont(fd)
		cidSubtype, cidToGID := d.cidDescendantInfo(fd, subtype)
		fonts = append(fonts, model.Font{
			Subtype:                subtype,
			BaseFont:               string(baseFont),
			Embedded:               d.fontIsEmbedded(fd, subtype),
			HasToUnicode:           hasToU,
			Encoding:               encName,
			HasEncodingDifferences: hasDiff,
			IsSymbolic:             isSym,
			HasUnicodeMapping:      hasToU || hasDeterministicUnicodeMapping(subtype, encName, hasDiff, isSym),
			ToUnicodeMappings:      mappings,
			ToUnicodeCodeBytes:     codeBytes,
			CIDSubtype:             cidSubtype,
			CIDToGIDMap:            cidToGID,
		})
	}
	return fonts, nil
}

func (d *document) fontIsEmbedded(font *pdd.Dict, subtype string) bool {
	switch subtype {
	case "Type3":
		// Type3 inlines its glyphs in the PDF, always "embedded" in spec terms.
		return true
	case "Type0":
		arr, ok := font.Array("DescendantFonts")
		if !ok || len(arr) == 0 {
			return false
		}
		cid, err := d.r.ResolveDict(arr[0])
		if err != nil || cid == nil {
			return false
		}
		return descriptorHasFontFile(cid)
	default:
		return descriptorHasFontFile(font)
	}
}

// cidDescendantInfo resolves the first DescendantFonts entry for a
// Type0 font and reports its /Subtype plus the shape of its
// /CIDToGIDMap entry. Returns ("", "") for non-Type0 fonts or when
// the descendant cannot be resolved. CIDToGIDMap classification:
//   - "Identity" — explicit /Identity name OR entry absent (ISO
//     32000-1 §9.7.4.2 makes Identity the default)
//   - "Stream"   — a stream is present
//   - <name>     — any other Name value (the UA-31-001 failure path)
// Only meaningful when the descendant is CIDFontType2; CIDFontType0
// (Adobe CFF source) has no CIDToGIDMap at all and the field stays
// "".
func (d *document) cidDescendantInfo(font *pdd.Dict, subtype string) (cidSubtype, cidToGIDMap string) {
	if subtype != "Type0" {
		return "", ""
	}
	arr, ok := font.Array("DescendantFonts")
	if !ok || len(arr) == 0 {
		return "", ""
	}
	cid, err := d.r.ResolveDict(arr[0])
	if err != nil || cid == nil {
		return "", ""
	}
	cn, _ := cid.Name("Subtype")
	cidSubtype = string(cn)
	if cidSubtype != "CIDFontType2" {
		return cidSubtype, ""
	}
	mapObj, present := cid.Get("CIDToGIDMap")
	if !present {
		return cidSubtype, "Identity"
	}
	resolved, err := d.r.Resolve(mapObj)
	if err != nil {
		return cidSubtype, ""
	}
	switch v := resolved.(type) {
	case pdd.Name:
		return cidSubtype, string(v)
	case *pdd.Stream:
		return cidSubtype, "Stream"
	}
	return cidSubtype, ""
}

// fontEncoding reads the /Encoding entry, returning the (possibly
// /BaseEncoding) Name and whether a /Differences array is present.
// /Encoding may be a Name, an inline Dict or an indirect ref to a Dict.
func (d *document) fontEncoding(fd *pdd.Dict) (name string, hasDifferences bool) {
	encObj, ok := fd.Get("Encoding")
	if !ok {
		return "", false
	}
	switch v := encObj.(type) {
	case pdd.Name:
		return string(v), false
	case pdd.Reference:
		ed, err := d.r.ResolveDict(v)
		if err != nil || ed == nil {
			return "", false
		}
		return dictEncoding(ed)
	case *pdd.Dict:
		return dictEncoding(v)
	}
	return "", false
}

func dictEncoding(d *pdd.Dict) (string, bool) {
	base, _ := d.Name("BaseEncoding")
	_, hasDiff := d.Get("Differences")
	return string(base), hasDiff
}

// hasDeterministicUnicodeMapping reports whether a font without a
// /ToUnicode CMap nonetheless satisfies PDF/UA-1 §7.21.3.1: a
// non-symbolic simple font that declares WinAnsiEncoding,
// MacRomanEncoding or MacExpertEncoding via /Encoding and does not
// introduce a /Differences override has an implicit, deterministic
// Unicode mapping derived from the encoding tables. Symbolic fonts
// (Flags bit 3) carry custom glyph repertoires that the predefined
// encodings cannot describe -- the glyph at byte 0x41 in a Symbol
// or dingbats font is not "A". Type0 composite fonts require their
// own analysis (predefined CMaps, descendant CIDFonts) which is not
// yet implemented; callers fall back to requiring /ToUnicode for them.
func hasDeterministicUnicodeMapping(subtype, encoding string, hasDifferences, isSymbolic bool) bool {
	if hasDifferences || isSymbolic {
		return false
	}
	switch subtype {
	case "Type1", "TrueType", "MMType1":
		switch encoding {
		case "WinAnsiEncoding", "MacRomanEncoding", "MacExpertEncoding":
			return true
		}
	}
	return false
}

// fontIsSymbolic reports whether the font carries a custom glyph
// repertoire that the WinAnsi/MacRoman/MacExpert encoding shortcut
// cannot describe. Two signals are combined because authors
// sometimes mis-declare the FontDescriptor /Flags:
//
//   - /FontDescriptor/Flags bit 3 (Symbolic) set, OR
//   - /BaseFont name contains a well-known symbolic-font marker
//     ("symbol", "dingbats", "wingdings", "webdings"). The subset
//     prefix that precedes embedded names ("ABCDEF+") is stripped
//     before the check.
//
// Either signal is sufficient. A font that lies about its Flags but
// is named "ABCXYZ+SegoeUISymbol" is still recognised.
func (d *document) fontIsSymbolic(font *pdd.Dict, subtype string) bool {
	if flagSaysSymbolic(d, font, subtype) {
		return true
	}
	if name, ok := font.Name("BaseFont"); ok {
		if baseFontLooksSymbolic(string(name)) {
			return true
		}
	}
	return false
}

func flagSaysSymbolic(d *document, font *pdd.Dict, subtype string) bool {
	var fd *pdd.Dict
	if subtype == "Type0" {
		arr, ok := font.Array("DescendantFonts")
		if !ok || len(arr) == 0 {
			return false
		}
		cid, err := d.r.ResolveDict(arr[0])
		if err != nil || cid == nil {
			return false
		}
		fd, _ = cid.Dict("FontDescriptor")
	} else {
		fd, _ = font.Dict("FontDescriptor")
	}
	if fd == nil {
		return false
	}
	flags, ok := fd.Int("Flags")
	if !ok {
		return false
	}
	return flags&4 != 0 // bit 3
}

func baseFontLooksSymbolic(name string) bool {
	if i := strings.IndexByte(name, '+'); i >= 0 && i < 10 {
		name = name[i+1:]
	}
	lower := strings.ToLower(name)
	for _, marker := range []string{"symbol", "dingbats", "wingdings", "webdings"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// descriptorHasFontFile resolves /FontDescriptor and looks for any of
// /FontFile, /FontFile2, /FontFile3 -- the spec signal for "embedded".
func descriptorHasFontFile(font *pdd.Dict) bool {
	fd, ok := font.Dict("FontDescriptor")
	if !ok || fd == nil {
		return false
	}
	for _, k := range []string{"FontFile", "FontFile2", "FontFile3"} {
		if fd.Has(k) {
			return true
		}
	}
	return false
}

// Close releases the underlying file if LoadFile owns one.
func (d *document) Close() error {
	if d.closer != nil {
		return d.closer.Close()
	}
	return nil
}
