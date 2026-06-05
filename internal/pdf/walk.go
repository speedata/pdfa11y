package pdf

import (
	"errors"
	"io"

	pdd "github.com/speedata/pdfdisassembler"
	"github.com/speedata/pdfdisassembler/contentstream"

	"github.com/speedata/pdfa11y/internal/model"
)

// maxUntaggedOpsPerPage caps how many untagged-content findings we
// record per page. One broken page can emit hundreds; the check only
// needs enough samples to point the user at the problem.
const maxUntaggedOpsPerPage = 32

// Pages returns one PageReport per page in document order. The result
// is cached; subsequent calls return the same slice.
func (d *document) Pages() ([]model.PageReport, error) {
	if d.pageReports != nil {
		return d.pageReports, nil
	}
	pages, err := d.collectPages()
	if err != nil {
		return nil, err
	}
	treeMCIDs := d.collectStructTreeMCIDs()
	reports := make([]model.PageReport, 0, len(pages))
	for _, p := range pages {
		rep := model.PageReport{
			Number:          p.Number,
			UsedFonts:       map[string]model.Font{},
			ContentMCIDs:    map[int]bool{},
			StructTreeMCIDs: treeMCIDs[p.Ref],
			Tabs:            p.Tabs,
			FontCodes:       map[string]map[uint32]bool{},
		}
		if rep.StructTreeMCIDs == nil {
			rep.StructTreeMCIDs = map[int]bool{}
		}
		if err := d.scanPageContent(p, &rep); err != nil {
			// One broken page should not abort the rest of the report;
			// record what we managed and move on.
			rep.UntaggedOps = append(rep.UntaggedOps, model.UntaggedOp{Operator: "<scan-error>"})
		}
		reports = append(reports, rep)
	}
	d.pageReports = reports
	return reports, nil
}

// pageInfo bundles the per-page data we resolve up-front so the
// content-stream scanner does not need to re-walk inheritance.
type pageInfo struct {
	Ref        pdd.Reference
	Number     int
	PageDict   *pdd.Dict
	Fonts      *pdd.Dict // /Resources/Font, possibly inherited; may be nil
	Properties *pdd.Dict // /Resources/Properties, possibly inherited; may be nil
	Tabs       string    // /Tabs value, possibly inherited; "" if absent everywhere
}

// collectPages walks the page tree once and returns one pageInfo per
// leaf Page node, in document order. /Resources is inherited from
// ancestors per PDF 32000-1:2008 §7.7.3.4 (Inheritance of Attributes).
func (d *document) collectPages() ([]pageInfo, error) {
	cat, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, errors.New("no document catalog")
	}
	pagesObj, ok := cat.Get("Pages")
	if !ok {
		return nil, errors.New("catalog has no /Pages")
	}
	ref, ok := pagesObj.(pdd.Reference)
	if !ok {
		return nil, errors.New("/Pages is not an indirect reference")
	}
	var pages []pageInfo
	counter := 0
	d.collectPagesWalk(ref, nil, nil, "", &pages, &counter)
	return pages, nil
}

func (d *document) collectPagesWalk(ref pdd.Reference, inheritedFonts, inheritedProps *pdd.Dict, inheritedTabs string, out *[]pageInfo, counter *int) {
	pageDict, err := d.r.ResolveDict(ref)
	if err != nil || pageDict == nil {
		return
	}
	fonts := inheritedFonts
	props := inheritedProps
	tabs := inheritedTabs
	if res, ok := pageDict.Dict("Resources"); ok && res != nil {
		if f, ok := res.Dict("Font"); ok {
			fonts = f
		}
		if p, ok := res.Dict("Properties"); ok {
			props = p
		}
	}
	if t, ok := pageDict.Name("Tabs"); ok {
		tabs = string(t)
	}
	typeName, _ := pageDict.Name("Type")
	switch typeName {
	case "Page":
		*counter++
		*out = append(*out, pageInfo{
			Ref:        ref,
			Number:     *counter,
			PageDict:   pageDict,
			Fonts:      fonts,
			Properties: props,
			Tabs:       tabs,
		})
	case "Pages":
		kids, ok := pageDict.Array("Kids")
		if !ok {
			return
		}
		for _, kid := range kids {
			kidRef, ok := kid.(pdd.Reference)
			if !ok {
				continue
			}
			d.collectPagesWalk(kidRef, fonts, props, tabs, out, counter)
		}
	}
}

// scanPageContent walks the content stream of a single page and
// populates the report fields. Errors are returned for malformed
// streams; the caller decides whether to bail.
func (d *document) scanPageContent(p pageInfo, rep *model.PageReport) error {
	body, err := d.pageContentBytes(p.PageDict)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	mcDepth := 0
	currentFontKey := ""
	currentFontCodeBytes := 1 // 1-byte default for simple fonts
	sc := contentstream.New(body)
	for {
		op, err := sc.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch op.Operator {
		case "BMC", "BDC":
			mcDepth++
			if op.Operator == "BDC" {
				if mcid, ok := extractMCID(op, p.Properties); ok {
					rep.ContentMCIDs[mcid] = true
				}
			}
		case "EMC":
			if mcDepth > 0 {
				mcDepth--
			}
		case "Tf":
			if len(op.Operands) >= 1 && op.Operands[0].Kind == contentstream.KindName {
				key := op.Operands[0].Name
				currentFontKey = key
				var f model.Font
				var ok bool
				if f, ok = rep.UsedFonts[key]; !ok {
					f, ok = d.resolveResourceFont(p.Fonts, key)
					if ok {
						rep.UsedFonts[key] = f
					}
				}
				currentFontCodeBytes = codeBytesFor(f, ok)
			}
		case "Tj", "'", `"`:
			recordTextCodes(rep, currentFontKey, currentFontCodeBytes, textArgBytes(op))
			recordIfUntagged(rep, op, mcDepth)
		case "TJ":
			recordTextCodes(rep, currentFontKey, currentFontCodeBytes, tjArrayBytes(op))
			recordIfUntagged(rep, op, mcDepth)
		default:
			recordIfUntagged(rep, op, mcDepth)
		}
	}
}

// recordIfUntagged appends the op to rep.UntaggedOps when it is a
// real-content operator running outside any marked-content sequence.
// Capped at maxUntaggedOpsPerPage so a single broken page does not
// flood the report.
func recordIfUntagged(rep *model.PageReport, op contentstream.Op, mcDepth int) {
	if mcDepth != 0 || !isRealContentOp(op.Operator) {
		return
	}
	if len(rep.UntaggedOps) >= maxUntaggedOpsPerPage {
		return
	}
	rep.UntaggedOps = append(rep.UntaggedOps, model.UntaggedOp{
		Operator: op.Operator,
		Offset:   op.Offset,
	})
}

// recordTextCodes splits raw text-show bytes into per-font codes and
// records them in rep.FontCodes. codeBytes is the active font's
// code width (1 for simple fonts and Type0 with single-byte CMaps;
// 2 for Type0 with Identity-H or other two-byte CMaps).
func recordTextCodes(rep *model.PageReport, fontKey string, codeBytes int, raw []byte) {
	if fontKey == "" || len(raw) == 0 {
		return
	}
	if codeBytes < 1 {
		codeBytes = 1
	}
	set, ok := rep.FontCodes[fontKey]
	if !ok {
		set = map[uint32]bool{}
		rep.FontCodes[fontKey] = set
	}
	switch codeBytes {
	case 1:
		for _, b := range raw {
			set[uint32(b)] = true
		}
	case 2:
		for i := 0; i+1 < len(raw); i += 2 {
			set[uint32(raw[i])<<8|uint32(raw[i+1])] = true
		}
	default:
		// >2-byte codes are rare in practice; pack big-endian.
		for i := 0; i+codeBytes <= len(raw); i += codeBytes {
			var v uint32
			for j := 0; j < codeBytes && j < 4; j++ {
				v = v<<8 | uint32(raw[i+j])
			}
			set[v] = true
		}
	}
}

// codeBytesFor picks the right code width for a Tf-referenced font.
//
// Simple fonts (Type1, TrueType, MMType1, Type3) always emit one
// byte per glyph in the Tj/TJ stream regardless of what their
// /ToUnicode codespace declares -- producers routinely declare a
// permissive <0000>-<FFFF> codespace on a 1-byte CMap, which means
// the codespace alone is not a reliable signal.
//
// For Type0 composite fonts the codespace IS authoritative
// (Identity-H is two-byte; custom CMaps may declare one-byte).
// Default to 2 when Type0 has no /ToUnicode at all.
//
// resolved=false (unknown font dict) collapses to the 1-byte
// fallback so we never accidentally pack ASCII text as 2-byte CIDs.
func codeBytesFor(f model.Font, resolved bool) int {
	if !resolved {
		return 1
	}
	if f.Subtype != "Type0" {
		return 1
	}
	if f.ToUnicodeCodeBytes > 0 {
		return f.ToUnicodeCodeBytes
	}
	return 2
}

// textArgBytes returns the raw string operand of Tj, ', " operators.
func textArgBytes(op contentstream.Op) []byte {
	if len(op.Operands) == 0 {
		return nil
	}
	// Tj has one string operand; ' has one string; " has two
	// numbers (Tw, Tc) followed by the string.
	last := op.Operands[len(op.Operands)-1]
	if last.Kind == contentstream.KindString {
		return last.Bytes
	}
	return nil
}

// tjArrayBytes concatenates all string elements of the TJ operand
// array, ignoring the kerning numbers.
func tjArrayBytes(op contentstream.Op) []byte {
	if len(op.Operands) == 0 || op.Operands[0].Kind != contentstream.KindArray {
		return nil
	}
	var out []byte
	for _, el := range op.Operands[0].Array {
		if el.Kind == contentstream.KindString {
			out = append(out, el.Bytes...)
		}
	}
	return out
}

// pageContentBytes returns the decoded /Contents bytes for a page,
// joining stream arrays with a single newline separator (PDF
// 32000-1:2008 §7.8.2 allows any whitespace; a newline is unambiguous).
func (d *document) pageContentBytes(pageDict *pdd.Dict) ([]byte, error) {
	cObj, ok := pageDict.Get("Contents")
	if !ok {
		return nil, nil
	}
	switch v := cObj.(type) {
	case pdd.Reference:
		return d.r.DecodeStream(v)
	case pdd.Array:
		var out []byte
		for i, item := range v {
			b, err := d.r.DecodeStream(item)
			if err != nil {
				return nil, err
			}
			if i > 0 {
				out = append(out, '\n')
			}
			out = append(out, b...)
		}
		return out, nil
	}
	return nil, nil
}

// extractMCID pulls the /MCID integer from a BDC properties operand.
// BDC's second operand is either an inline dict (with /MCID directly)
// or a name pointing into /Resources/Properties (whose entry is the
// dict). Returns false when no MCID is present.
func extractMCID(op contentstream.Op, properties *pdd.Dict) (int, bool) {
	if len(op.Operands) < 2 {
		return 0, false
	}
	props := op.Operands[1]
	switch props.Kind {
	case contentstream.KindDict:
		if v, ok := props.Dict["MCID"]; ok {
			if n, ok := v.Int(); ok {
				return int(n), true
			}
		}
	case contentstream.KindName:
		if properties == nil {
			return 0, false
		}
		entry, ok := properties.Dict(props.Name)
		if !ok || entry == nil {
			return 0, false
		}
		if n, ok := entry.Int("MCID"); ok {
			return int(n), true
		}
	}
	return 0, false
}

// resolveResourceFont resolves a font resource key (e.g. "F1") through
// /Resources/Font to a model.Font snapshot matching what Fonts()
// returns. The lookup is best-effort: missing keys return ok=false.
func (d *document) resolveResourceFont(fontsDict *pdd.Dict, key string) (model.Font, bool) {
	if fontsDict == nil {
		return model.Font{}, false
	}
	obj, ok := fontsDict.Get(key)
	if !ok {
		return model.Font{}, false
	}
	fd, err := d.r.ResolveDict(obj)
	if err != nil || fd == nil {
		return model.Font{}, false
	}
	return d.fontFromDict(fd), true
}

// fontFromDict builds a model.Font snapshot from a resolved font dict.
// Mirrors the logic in Fonts(); kept as a small helper so Pages() does
// not have to re-implement field population.
func (d *document) fontFromDict(fd *pdd.Dict) model.Font {
	subtypeName, _ := fd.Name("Subtype")
	subtype := string(subtypeName)
	baseFont, _ := fd.Name("BaseFont")
	encName, hasDiff := d.fontEncoding(fd)
	hasToU := fd.Has("ToUnicode")
	isSym := d.fontIsSymbolic(fd, subtype)
	mappings, codeBytes := d.parseToUnicodeFromFont(fd)
	return model.Font{
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
	}
}

// parseToUnicodeFromFont decodes the font's /ToUnicode stream (when
// present) and parses its bfchar / bfrange mappings together with
// the codespace byte width. Returns (nil, 0) for fonts without a
// /ToUnicode entry. Returns an empty (non-nil) map for streams that
// parse but contain no mappings.
func (d *document) parseToUnicodeFromFont(fd *pdd.Dict) (map[uint32]string, int) {
	touObj, ok := fd.Get("ToUnicode")
	if !ok {
		return nil, 0
	}
	body, err := d.r.DecodeStream(touObj)
	if err != nil {
		return nil, 0
	}
	cov := parseToUnicode(body)
	mappings := cov.Mappings
	if mappings == nil {
		mappings = map[uint32]string{}
	}
	return mappings, cov.CodeBytes
}

// isRealContentOp reports whether a content-stream operator paints
// something that participates in the document's semantic content:
// text-showing operators, XObject invocations, path-painting
// operators, and inline images. Path-construction operators (m, l,
// re, c, h, y, v) are not painting on their own, so they are not
// counted.
func isRealContentOp(op string) bool {
	switch op {
	case "Tj", "TJ", "'", `"`:
		return true
	case "Do":
		return true
	case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*":
		return true
	case "EI": // inline image
		return true
	}
	return false
}

// collectStructTreeMCIDs walks the structure tree and aggregates, per
// page reference, the set of MCIDs that the tree claims for that page.
//
// MCIDs are surfaced via two child shapes:
//   - integer K-entry under a structure element: MCID = that integer,
//     page = the element's effective /Pg (its own or inherited from
//     an ancestor).
//   - dict child {/Type /MCR /MCID n /Pg ref}: MCID = n, page = /Pg
//     if present, else the effective inherited /Pg.
func (d *document) collectStructTreeMCIDs() map[pdd.Reference]map[int]bool {
	out := map[pdd.Reference]map[int]bool{}
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return out
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return out
	}
	d.walkStructForMCIDs(stree, pdd.Reference{}, out)
	return out
}

func (d *document) walkStructForMCIDs(elem *pdd.Dict, inheritedPage pdd.Reference, out map[pdd.Reference]map[int]bool) {
	if elem == nil {
		return
	}
	page := inheritedPage
	if pg, ok := elem.Get("Pg"); ok {
		if ref, ok := pg.(pdd.Reference); ok {
			page = ref
		}
	}
	kObj, ok := elem.Get("K")
	if !ok {
		return
	}
	resolved, err := d.r.Resolve(kObj)
	if err != nil {
		return
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		items = []pdd.Object{v}
	}
	for _, item := range items {
		d.handleStructKEntry(item, page, out)
	}
}

func (d *document) handleStructKEntry(item pdd.Object, page pdd.Reference, out map[pdd.Reference]map[int]bool) {
	if n, ok := asInt(item); ok {
		// Bare integer: MCID on the current page.
		recordMCID(out, page, int(n))
		return
	}
	child, err := d.r.ResolveDict(item)
	if err != nil || child == nil {
		return
	}
	tName, _ := child.Name("Type")
	switch string(tName) {
	case "MCR":
		mcid, ok := child.Int("MCID")
		if !ok {
			return
		}
		ref := page
		if pg, ok := child.Get("Pg"); ok {
			if r, ok := pg.(pdd.Reference); ok {
				ref = r
			}
		}
		recordMCID(out, ref, int(mcid))
	case "OBJR":
		// Object reference, not an MCID.
		return
	default:
		// Treat as nested structure element.
		d.walkStructForMCIDs(child, page, out)
	}
}

func recordMCID(out map[pdd.Reference]map[int]bool, page pdd.Reference, mcid int) {
	if page == (pdd.Reference{}) {
		return
	}
	m := out[page]
	if m == nil {
		m = map[int]bool{}
		out[page] = m
	}
	m[mcid] = true
}

// asInt accepts the pdd.Integer concrete type and reports its value.
// Defined here so we do not depend on pdd internals beyond the public
// Object types.
func asInt(o pdd.Object) (int64, bool) {
	if n, ok := o.(pdd.Integer); ok {
		return int64(n), true
	}
	return 0, false
}
