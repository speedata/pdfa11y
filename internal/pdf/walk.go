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
	d.collectPagesWalk(ref, nil, nil, &pages, &counter)
	return pages, nil
}

func (d *document) collectPagesWalk(ref pdd.Reference, inheritedFonts, inheritedProps *pdd.Dict, out *[]pageInfo, counter *int) {
	pageDict, err := d.r.ResolveDict(ref)
	if err != nil || pageDict == nil {
		return
	}
	fonts := inheritedFonts
	props := inheritedProps
	if res, ok := pageDict.Dict("Resources"); ok && res != nil {
		if f, ok := res.Dict("Font"); ok {
			fonts = f
		}
		if p, ok := res.Dict("Properties"); ok {
			props = p
		}
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
			d.collectPagesWalk(kidRef, fonts, props, out, counter)
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
				if _, already := rep.UsedFonts[key]; !already {
					if f, ok := d.resolveResourceFont(p.Fonts, key); ok {
						rep.UsedFonts[key] = f
					}
				}
			}
		default:
			if mcDepth == 0 && isRealContentOp(op.Operator) {
				if len(rep.UntaggedOps) < maxUntaggedOpsPerPage {
					rep.UntaggedOps = append(rep.UntaggedOps, model.UntaggedOp{
						Operator: op.Operator,
						Offset:   op.Offset,
					})
				}
			}
		}
	}
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
	return model.Font{
		Subtype:                subtype,
		BaseFont:               string(baseFont),
		Embedded:               d.fontIsEmbedded(fd, subtype),
		HasToUnicode:           hasToU,
		Encoding:               encName,
		HasEncodingDifferences: hasDiff,
		IsSymbolic:             isSym,
		HasUnicodeMapping:      hasToU || hasDeterministicUnicodeMapping(subtype, encName, hasDiff, isSym),
	}
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
