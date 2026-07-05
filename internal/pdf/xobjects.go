package pdf

import (
	"errors"
	"io"

	pdd "github.com/speedata/pdfdisassembler"
	cs "github.com/speedata/pdfdisassembler/contentstream"

	"github.com/speedata/pdfa11y/internal/model"
)

// FormXObjects scans every live indirect object for a form XObject (/Type
// /XObject, /Subtype /Form). Each is flagged with whether it carries a /Ref
// entry (a reference XObject importing page content from an external file, ISO
// 32000-1 §8.10.4; used by UA-30-001), whether its content stream carries
// marked content with an /MCID, and how many times it is invoked by a Do
// operator across all page and form-XObject content streams (used by
// UA-30-002). Walking the raw object table rather than the resource graph
// catches XObjects wherever they sit (page resources, nested XObjects,
// annotation appearance streams).
func (d *document) FormXObjects() []model.FormXObject {
	type formEntry struct {
		num     int
		hasRef  bool
		hasMCID bool
		refDict *pdd.Dict // /Resources of this XObject, for resolving nested Do
		refs    []string  // XObject resource names invoked by Do inside this XObject
	}
	var forms []*formEntry
	byNum := map[int]*formEntry{}

	for entry := range d.r.Objects() {
		st, ok := entry.Object.(*pdd.Stream)
		if !ok || st.Dict == nil {
			continue
		}
		dict := st.Dict
		if t, ok := dict.Name("Type"); ok && t != "XObject" {
			continue
		}
		if sub, _ := dict.Name("Subtype"); sub != "Form" {
			continue
		}
		fe := &formEntry{num: entry.Reference.Number, hasRef: dict.Has("Ref")}
		if res, ok := dict.Dict("Resources"); ok {
			fe.refDict = res
		}
		if body, err := d.r.DecodeStream(entry.Reference); err == nil {
			fe.hasMCID, fe.refs = scanMarkedContentAndDo(body)
		}
		forms = append(forms, fe)
		byNum[fe.num] = fe
	}

	// Tally Do references. A reference is any Do operator whose operand
	// name resolves, through the invoking stream's /Resources/XObject, to a
	// form XObject's object number. Both page content streams and nested
	// form-XObject content streams count.
	refCount := map[int]int{}
	countDo := func(names []string, res *pdd.Dict) {
		if res == nil {
			return
		}
		xo, ok := res.Dict("XObject")
		if !ok || xo == nil {
			return
		}
		for _, nm := range names {
			obj, ok := xo.Get(nm)
			if !ok {
				continue
			}
			if ref, ok := obj.(pdd.Reference); ok {
				if _, isForm := byNum[ref.Number]; isForm {
					refCount[ref.Number]++
				}
			}
		}
	}
	for _, fe := range forms {
		countDo(fe.refs, fe.refDict)
	}
	if pages, err := d.collectPages(); err == nil {
		for _, p := range pages {
			body, err := d.pageContentBytes(p.PageDict)
			if err != nil || len(body) == 0 {
				continue
			}
			_, names := scanMarkedContentAndDo(body)
			countDo(names, pageResources(p.PageDict))
		}
	}

	out := make([]model.FormXObject, 0, len(forms))
	for _, fe := range forms {
		out = append(out, model.FormXObject{
			ObjectNumber:     fe.num,
			IsReference:      fe.hasRef,
			HasMarkedContent: fe.hasMCID,
			RefCount:         refCount[fe.num],
		})
	}
	return out
}

// scanMarkedContentAndDo tokenizes a content stream once and reports whether
// it contains a BDC with an /MCID property, plus the XObject resource names
// invoked by Do operators (in order, with duplicates preserved so repeated
// invocations are counted).
func scanMarkedContentAndDo(body []byte) (hasMCID bool, doNames []string) {
	sc := cs.New(body)
	for {
		op, err := sc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			break
		}
		switch op.Operator {
		case "BDC":
			if len(op.Operands) >= 2 && op.Operands[1].Kind == cs.KindDict {
				if _, ok := op.Operands[1].Dict["MCID"]; ok {
					hasMCID = true
				}
			}
		case "Do":
			if len(op.Operands) >= 1 && op.Operands[0].Kind == cs.KindName {
				doNames = append(doNames, op.Operands[0].Name)
			}
		}
	}
	return hasMCID, doNames
}

// pageResources returns a page's effective /Resources dict, or nil. Resource
// inheritance from ancestor page-tree nodes is not followed here: a page that
// draws a form XObject via Do carries an /XObject resource itself in practice.
func pageResources(pageDict *pdd.Dict) *pdd.Dict {
	if pageDict == nil {
		return nil
	}
	if res, ok := pageDict.Dict("Resources"); ok {
		return res
	}
	return nil
}
