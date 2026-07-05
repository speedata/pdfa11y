package pdf

import (
	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// HumanReadableStrings enumerates the document's human-readable text strings
// (ISO 14289-2 §8.6): document-information entries, outline titles, form-field
// names, annotation contents, structure-element alternate text and
// optional-content group names. Used by UA-01-019 to flag Unicode PUA in text
// meant to be read.
func (d *document) HumanReadableStrings() []model.TextString {
	var out []model.TextString
	add := func(loc string, dict *pdd.Dict, key string) {
		if s, ok := dict.String(key); ok && s != "" {
			out = append(out, model.TextString{Location: loc + " /" + key, Value: s})
		}
	}

	// Document information dictionary (trailer /Info).
	if tr := d.r.Trailer(); tr != nil {
		if info, ok := tr.Dict("Info"); ok && info != nil {
			for _, k := range []string{"Title", "Author", "Subject", "Keywords"} {
				add("Info", info, k)
			}
		}
	}

	// Object-table scan for structure elements, optional-content groups and
	// form fields / widget annotations. Walking the raw table catches these
	// wherever they live, including inside object streams.
	for entry := range d.r.Objects() {
		var dict *pdd.Dict
		switch v := entry.Object.(type) {
		case *pdd.Dict:
			dict = v
		case *pdd.Stream:
			dict = v.Dict
		}
		if dict == nil {
			continue
		}
		typ, _ := dict.Name("Type")
		sub, _ := dict.Name("Subtype")
		switch {
		case typ == "StructElem":
			for _, k := range []string{"Alt", "ActualText", "E", "T"} {
				add("structure element", dict, k)
			}
		case typ == "OCG":
			add("optional-content group", dict, "Name")
		case sub == "Widget" || dict.Has("FT"):
			add("form field", dict, "T")
			add("form field", dict, "TU")
			add("annotation", dict, "Contents")
		case dict.Has("Rect") && sub != "":
			// Other annotations (markup, link, ...): /Contents is the
			// human-readable description, /T the text label / author.
			add("annotation", dict, "Contents")
			add("annotation", dict, "T")
		}
	}

	// Outline item titles.
	if cat, err := d.r.Catalog(); err == nil && cat != nil {
		if outlines, ok := cat.Dict("Outlines"); ok && outlines != nil {
			if firstObj, ok := outlines.Get("First"); ok {
				d.walkOutlineTitles(firstObj, &out, 0)
			}
		}
	}

	return out
}

// walkOutlineTitles collects the /Title of each outline item, recursing into
// /First subtrees and iterating /Next siblings, mirroring walkOutlineDests.
func (d *document) walkOutlineTitles(itemObj pdd.Object, out *[]model.TextString, depth int) {
	if depth > 200 {
		return
	}
	seen := 0
	for cur := itemObj; cur != nil; {
		item, err := d.r.ResolveDict(cur)
		if err != nil || item == nil {
			return
		}
		if t, ok := item.String("Title"); ok && t != "" {
			*out = append(*out, model.TextString{Location: "outline item /Title", Value: t})
		}
		if firstObj, ok := item.Get("First"); ok {
			d.walkOutlineTitles(firstObj, out, depth+1)
		}
		nextObj, ok := item.Get("Next")
		if !ok {
			return
		}
		cur = nextObj
		seen++
		if seen > 100000 {
			return
		}
	}
}
