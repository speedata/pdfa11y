package pdf

import (
	"fmt"
	"strconv"
	"strings"

	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// IntraDocDestinations enumerates every intra-document destination in the
// document and classifies its target as a page (a §8.8 violation) or not.
// It walks three carrier families:
//
//   - outline items (catalog /Outlines, recursed via /First and /Next):
//     each item's /Dest, plus a /GoTo action reached through /A;
//   - the catalog /OpenAction, which may be a destination array or an
//     action dictionary;
//   - Link annotations on every page: their /Dest and any /GoTo /A.
//
// Remote destinations (/GoToR) and non-destination actions (/URI, ...)
// carry no intra-document target and are skipped.
func (d *document) IntraDocDestinations() []model.Destination {
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return nil
	}
	var out []model.Destination

	// Outline items.
	if outlinesObj, ok := cat.Get("Outlines"); ok {
		if outlines, err := d.r.ResolveDict(outlinesObj); err == nil && outlines != nil {
			if firstObj, ok := outlines.Get("First"); ok {
				d.walkOutlineDests(firstObj, &out, 0)
			}
		}
	}

	// Catalog /OpenAction: a destination array or an action dictionary.
	if oaObj, ok := cat.Get("OpenAction"); ok {
		if resolved, err := d.r.Resolve(oaObj); err == nil {
			if _, isArray := resolved.(pdd.Array); isArray {
				out = append(out, model.Destination{
					Source:      "catalog /OpenAction",
					TargetsPage: d.destTargetsPage(oaObj),
				})
			} else {
				d.collectGoToAction(oaObj, "catalog /OpenAction", &out)
			}
		}
	}

	// Link annotations.
	if pages, err := d.collectPages(); err == nil {
		for _, p := range pages {
			annotsObj, ok := p.PageDict.Get("Annots")
			if !ok {
				continue
			}
			arr, err := d.r.ResolveArray(annotsObj)
			if err != nil {
				continue
			}
			for _, item := range arr {
				ad, err := d.r.ResolveDict(item)
				if err != nil || ad == nil {
					continue
				}
				if name, _ := ad.Name("Subtype"); name != "Link" {
					continue
				}
				d.collectCarrierDests(ad, fmt.Sprintf("Link annotation on page %d", p.Number), &out)
			}
		}
	}

	return out
}

// walkOutlineDests visits an outline item and, iteratively, its /Next
// siblings, recursing into each item's /First subtree. depth bounds the
// subtree recursion; a large sibling counter bounds pathological /Next
// cycles. Each item's /Dest and /GoTo /A destinations are collected.
func (d *document) walkOutlineDests(itemObj pdd.Object, out *[]model.Destination, depth int) {
	if depth > 200 {
		return
	}
	seen := 0
	for cur := itemObj; cur != nil; {
		item, err := d.r.ResolveDict(cur)
		if err != nil || item == nil {
			return
		}
		title, _ := item.String("Title")
		d.collectCarrierDests(item, fmt.Sprintf("outline item %q", title), out)
		if firstObj, ok := item.Get("First"); ok {
			d.walkOutlineDests(firstObj, out, depth+1)
		}
		nextObj, ok := item.Get("Next")
		if !ok {
			return
		}
		cur = nextObj
		if seen++; seen > 100000 {
			return
		}
	}
}

// collectCarrierDests handles a carrier that may hold a destination two
// ways: a direct /Dest (a plain destination -> veraPDF UA2:8.8-1) and/or
// a /GoTo action under /A (-> UA2:8.8-2). A /SD structure-destination
// entry on the carrier means it already offers a structure destination,
// so the direct /Dest is treated as conformant.
func (d *document) collectCarrierDests(carrier *pdd.Dict, source string, out *[]model.Destination) {
	if destObj, ok := carrier.Get("Dest"); ok {
		targetsPage := false
		if !carrier.Has("SD") {
			targetsPage = d.destTargetsPage(destObj)
		}
		*out = append(*out, model.Destination{Source: source, TargetsPage: targetsPage})
	}
	if aObj, ok := carrier.Get("A"); ok {
		d.collectGoToAction(aObj, source, out)
	}
}

// collectGoToAction inspects an action dictionary and, when it is a /GoTo
// action (an intra-document jump), records its /D destination. /GoToR
// (remote) and other action types carry no intra-document target and are
// ignored. A /SD entry on the action is a structure destination and
// conforms.
func (d *document) collectGoToAction(actionObj pdd.Object, source string, out *[]model.Destination) {
	action, err := d.r.ResolveDict(actionObj)
	if err != nil || action == nil {
		return
	}
	if s, _ := action.Name("S"); s != "GoTo" {
		return
	}
	targetsPage := false
	if !action.Has("SD") {
		if dObj, ok := action.Get("D"); ok {
			targetsPage = d.destTargetsPage(dObj)
		}
	}
	*out = append(*out, model.Destination{Source: source, ViaGoToAction: true, TargetsPage: targetsPage})
}

// destTargetsPage reports whether destObj -- an explicit destination
// array, or a name/string naming one in the document's destination tables
// -- resolves to a page target rather than a structure element. Only a
// positively identified page returns true; a structure-element target or
// an unclassifiable one returns false so callers never raise a false
// positive.
func (d *document) destTargetsPage(destObj pdd.Object) bool {
	arr := d.explicitDest(destObj, 0)
	if len(arr) == 0 {
		return false
	}
	return d.targetIsPage(arr[0])
}

// explicitDest resolves destObj to the explicit destination array
// [target /Fit ...]. A name or string is looked up in the /Dests name
// dictionary or the /Names /Dests name tree respectively; a destination
// dictionary is unwrapped through its /D entry. depth bounds the
// indirection chain.
func (d *document) explicitDest(destObj pdd.Object, depth int) pdd.Array {
	if depth > 10 {
		return nil
	}
	resolved, err := d.r.Resolve(destObj)
	if err != nil {
		return nil
	}
	switch v := resolved.(type) {
	case pdd.Array:
		return v
	case pdd.Name:
		return d.explicitDest(d.namedDestByName(string(v)), depth+1)
	case pdd.String:
		return d.explicitDest(d.namedDestByString(string(v)), depth+1)
	case *pdd.Dict:
		if dObj, ok := v.Get("D"); ok {
			return d.explicitDest(dObj, depth+1)
		}
	}
	return nil
}

// targetIsPage reports whether the first element of an explicit
// destination array denotes a page: an inline integer page index, or a
// reference to a /Type /Page object. A structure-element target (or one
// with no /Type) is not a page.
func (d *document) targetIsPage(first pdd.Object) bool {
	if _, ok := first.(pdd.Integer); ok {
		return true
	}
	resolved, err := d.r.Resolve(first)
	if err != nil {
		return false
	}
	if _, ok := resolved.(pdd.Integer); ok {
		return true
	}
	dict, ok := resolved.(*pdd.Dict)
	if !ok {
		return false
	}
	t, _ := dict.Name("Type")
	return t == "Page"
}

// namedDestByName looks up a name-form named destination in the catalog's
// /Dests dictionary (ISO 32000-1 §12.3.2.3). Returns nil when absent.
func (d *document) namedDestByName(name string) pdd.Object {
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return nil
	}
	dests, ok := cat.Dict("Dests")
	if !ok {
		return nil
	}
	obj, _ := dests.Get(name)
	return obj
}

// namedDestByString looks up a string-form named destination in the
// catalog's /Names /Dests name tree (ISO 32000-1 §12.3.2.3). Returns nil
// when absent.
func (d *document) namedDestByString(key string) pdd.Object {
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return nil
	}
	names, ok := cat.Dict("Names")
	if !ok {
		return nil
	}
	destsObj, ok := names.Get("Dests")
	if !ok {
		return nil
	}
	return d.lookupNameTree(destsObj, key, 0)
}

// lookupNameTree finds key in a PDF name tree (ISO 32000-1 §7.9.6),
// returning its value or nil. It follows /Kids into child nodes and scans
// each /Names leaf array (alternating key, value). depth guards cycles.
func (d *document) lookupNameTree(nodeObj pdd.Object, key string, depth int) pdd.Object {
	if depth > 50 {
		return nil
	}
	node, err := d.r.ResolveDict(nodeObj)
	if err != nil || node == nil {
		return nil
	}
	if namesObj, ok := node.Get("Names"); ok {
		if arr, err := d.r.ResolveArray(namesObj); err == nil {
			for i := 0; i+1 < len(arr); i += 2 {
				if s, ok := arr[i].(pdd.String); ok && string(s) == key {
					return arr[i+1]
				}
			}
		}
	}
	if kidsObj, ok := node.Get("Kids"); ok {
		if arr, err := d.r.ResolveArray(kidsObj); err == nil {
			for _, kid := range arr {
				if v := d.lookupNameTree(kid, key, depth+1); v != nil {
					return v
				}
			}
		}
	}
	return nil
}

// linkTargetKey computes a canonical string identifying where a link
// annotation points, so two enclosed links can be compared for
// "same location". A link's target is, in order of precedence: the /A
// action's destination (its /SD structure destination, else /D), an
// explicit /SD or /Dest on the annotation, or a URI. Returns "" when no
// target can be determined. Used by UA-28-030 (ISO 14289-2 §8.2.5.20).
func (d *document) linkTargetKey(annot *pdd.Dict) string {
	if aObj, ok := annot.Get("A"); ok {
		if key := d.actionTargetKey(aObj); key != "" {
			return key
		}
	}
	// A structure destination (/SD) takes precedence over the legacy
	// page destination (/Dest): PDF 2.0 readers follow /SD.
	if sd, ok := annot.Get("SD"); ok {
		if k := d.destKey(sd); k != "" {
			return "dest:" + k
		}
	}
	if dest, ok := annot.Get("Dest"); ok {
		if k := d.destKey(dest); k != "" {
			return "dest:" + k
		}
	}
	return ""
}

// actionTargetKey computes the target key for an /A action: the URI for a
// URI action, or the destination (/SD preferred, else /D) for a GoTo /
// GoToR action. Other action types fall back to the action's object
// identity. Returns "" when the action cannot be resolved.
func (d *document) actionTargetKey(aObj pdd.Object) string {
	action, err := d.r.ResolveDict(aObj)
	if err != nil || action == nil {
		return ""
	}
	s, _ := action.Name("S")
	switch s {
	case "URI":
		uri, _ := action.String("URI")
		return "uri:" + uri
	case "GoTo":
		if sd, ok := action.Get("SD"); ok {
			if k := d.destKey(sd); k != "" {
				return "goto:" + k
			}
		}
		if dObj, ok := action.Get("D"); ok {
			if k := d.destKey(dObj); k != "" {
				return "goto:" + k
			}
		}
		return ""
	case "GoToR":
		file, _ := action.String("F")
		dst := ""
		if dObj, ok := action.Get("D"); ok {
			dst = d.destKey(dObj)
		}
		return "gotor:" + file + ":" + dst
	default:
		if ref, ok := aObj.(pdd.Reference); ok {
			return fmt.Sprintf("act:R%d", ref.Number)
		}
		return "act:" + string(s)
	}
}

// destKey resolves destObj to its explicit destination array and serialises
// it into a canonical string, so two destinations that point at the same
// target and view produce the same key. Returns "" when the destination
// cannot be resolved.
func (d *document) destKey(destObj pdd.Object) string {
	arr := d.explicitDest(destObj, 0)
	if len(arr) == 0 {
		return ""
	}
	parts := make([]string, 0, len(arr))
	for _, e := range arr {
		parts = append(parts, destTokenOf(e))
	}
	return strings.Join(parts, ",")
}

// destTokenOf renders one explicit-destination element as a stable token.
func destTokenOf(o pdd.Object) string {
	switch v := o.(type) {
	case pdd.Reference:
		return "R" + strconv.Itoa(v.Number)
	case pdd.Name:
		return "/" + string(v)
	case pdd.Integer:
		return strconv.FormatInt(int64(v), 10)
	case pdd.Real:
		return strconv.FormatFloat(float64(v), 'g', -1, 64)
	case pdd.String:
		return "(" + string(v) + ")"
	default:
		return "?"
	}
}
