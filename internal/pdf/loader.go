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
	r                 *pdd.Reader
	closer            io.Closer
	roleMap           map[string]string     // populated by loadRoleMap; may be empty
	pageIndex         map[pdd.Reference]int // page-ref -> 1-based page number
	pageReports       []model.PageReport    // cached result of Pages(); nil until first call
	annotations       []model.Annotation    // cached result of Annotations()
	annotationsLoaded bool                  // tracks Annotations() cache (nil-slice is a valid value)
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
	return e.doc.resolveRole(string(name))
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

func (e structElement) Attr(name string) string {
	// Dict.String already handles indirect-ref resolution and decodes
	// the PDF text-string encoding (PDFDocEncoded / UTF-16BE / UTF-8).
	s, _ := e.dict.String(name)
	return s
}

// Attribute pulls a Name value out of the structure element's /A
// attribute dictionary (or array of dictionaries). Returns "" when
// /A is missing, when name is not present, or when the value is not
// a Name.
func (e structElement) Attribute(name string) string {
	aObj, ok := e.dict.Get("A")
	if !ok {
		return ""
	}
	resolved, err := e.doc.r.Resolve(aObj)
	if err != nil {
		return ""
	}
	switch v := resolved.(type) {
	case *pdd.Dict:
		if n, ok := v.Name(name); ok {
			return string(n)
		}
	case pdd.Array:
		for _, item := range v {
			d, err := e.doc.r.ResolveDict(item)
			if err != nil || d == nil {
				continue
			}
			if n, ok := d.Name(name); ok {
				return string(n)
			}
		}
	}
	return ""
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
