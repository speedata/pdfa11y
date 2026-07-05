package pdf

import (
	pdd "github.com/speedata/pdfdisassembler"
)

// parentTree returns (and lazily caches) a map from a /StructParent key to the
// raw PDF object registered under that key in the StructTreeRoot /ParentTree
// number tree. For an annotation the value is a single structure-element
// reference; for a page it is an array of MCID owners. Callers resolve the
// value themselves. The map is empty when the document has no /ParentTree.
func (d *document) parentTree() map[int]pdd.Object {
	if d.parentTreeLoaded {
		return d.parentTreeMap
	}
	d.parentTreeMap = map[int]pdd.Object{}
	d.parentTreeLoaded = true

	stree, ok := d.structTreeRootDict()
	if !ok {
		return d.parentTreeMap
	}
	ptObj, ok := stree.Get("ParentTree")
	if !ok {
		return d.parentTreeMap
	}
	d.walkNumberTree(ptObj, 0)
	return d.parentTreeMap
}

// walkNumberTree recurses a PDF number tree (ISO 32000-1 §7.9.7), collecting
// its /Nums key/value pairs into d.parentTreeMap. Intermediate nodes carry a
// /Kids array; leaves carry /Nums [key1 val1 key2 val2 ...]. depth guards
// against pathological cycles.
func (d *document) walkNumberTree(nodeObj pdd.Object, depth int) {
	if depth > 50 {
		return
	}
	node, err := d.r.ResolveDict(nodeObj)
	if err != nil || node == nil {
		return
	}
	if numsObj, ok := node.Get("Nums"); ok {
		if arr, err := d.r.ResolveArray(numsObj); err == nil {
			for i := 0; i+1 < len(arr); i += 2 {
				key, ok := arr[i].(pdd.Integer)
				if !ok {
					continue
				}
				d.parentTreeMap[int(key)] = arr[i+1]
			}
		}
	}
	if kidsObj, ok := node.Get("Kids"); ok {
		if arr, err := d.r.ResolveArray(kidsObj); err == nil {
			for _, kid := range arr {
				d.walkNumberTree(kid, depth+1)
			}
		}
	}
}

// structLangForParent resolves the effective /Lang of the structure element
// registered under the given /StructParent key: the element's own /Lang, or
// the nearest ancestor's /Lang walking up the /P chain. Returns "" when the
// key is absent, does not resolve to a structure element, or no ancestor
// declares /Lang.
func (d *document) structLangForParent(structParent int) string {
	if structParent < 0 {
		return ""
	}
	val, ok := d.parentTree()[structParent]
	if !ok {
		return ""
	}
	elem, err := d.r.ResolveDict(val)
	if err != nil || elem == nil {
		return ""
	}
	return d.effectiveElementLang(elem)
}

// structTypeForParent returns the standard structure type (role-map resolved)
// of the element registered under the given /StructParent key. Returns "" when
// the key is absent or does not resolve to a structure element.
func (d *document) structTypeForParent(structParent int) string {
	if structParent < 0 {
		return ""
	}
	val, ok := d.parentTree()[structParent]
	if !ok {
		return ""
	}
	elem, err := d.r.ResolveDict(val)
	if err != nil || elem == nil {
		return ""
	}
	if _, hasS := elem.Get("S"); !hasS {
		return ""
	}
	return structElement{doc: d, dict: elem}.Type()
}

// structOwnLangForParent returns the /Lang declared directly on the structure
// element registered under the given /StructParent key -- WITHOUT walking up
// the /P ancestor chain. ISO 14289-1 §7.2 (veraPDF UA1:7.2-25, UA-11-009)
// treats a form field's /TU language as determinable only when the field's own
// structure element carries /Lang (or the catalog does); an ancestor's /Lang
// does not qualify. Returns "" when the key is absent or the element has no own
// /Lang.
func (d *document) structOwnLangForParent(structParent int) string {
	if structParent < 0 {
		return ""
	}
	val, ok := d.parentTree()[structParent]
	if !ok {
		return ""
	}
	elem, err := d.r.ResolveDict(val)
	if err != nil || elem == nil {
		return ""
	}
	if l, ok := elem.String("Lang"); ok {
		return l
	}
	return ""
}

// structAltForParent returns the /Alt entry of the structure element
// registered under the given /StructParent key -- the immediate enclosing
// element of the annotation. Returns "" when the key is absent, does not
// resolve to a structure element, or the element has no non-empty /Alt.
func (d *document) structAltForParent(structParent int) string {
	if structParent < 0 {
		return ""
	}
	val, ok := d.parentTree()[structParent]
	if !ok {
		return ""
	}
	elem, err := d.r.ResolveDict(val)
	if err != nil || elem == nil {
		return ""
	}
	if alt, ok := elem.String("Alt"); ok {
		return alt
	}
	return ""
}

// structHasLblForParent reports whether the structure element registered under
// the given /StructParent key -- the element enclosing a widget's OBJR -- has a
// Lbl structure element among its direct /K children. ISO 14289-2 §8.10.2.3
// treats such a Lbl as the label of the widget; UA-28-035 requires a widget
// without one (and without /Contents) to be flagged. Returns false when the key
// is absent or the element has no Lbl child.
func (d *document) structHasLblForParent(structParent int) bool {
	if structParent < 0 {
		return false
	}
	val, ok := d.parentTree()[structParent]
	if !ok {
		return false
	}
	elem, err := d.r.ResolveDict(val)
	if err != nil || elem == nil {
		return false
	}
	kObj, ok := elem.Get("K")
	if !ok {
		return false
	}
	resolved, err := d.r.Resolve(kObj)
	if err != nil {
		return false
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		items = []pdd.Object{v}
	}
	for _, item := range items {
		child, err := d.r.ResolveDict(item)
		if err != nil || child == nil {
			continue
		}
		if _, hasS := child.Get("S"); !hasS {
			continue
		}
		if (structElement{doc: d, dict: child}).Type() == "Lbl" {
			return true
		}
	}
	return false
}

// effectiveElementLang returns elem's /Lang, or the nearest ancestor's /Lang
// found by walking up the /P chain. Bounded against cycles.
func (d *document) effectiveElementLang(elem *pdd.Dict) string {
	cur := elem
	for range 50 {
		if l, ok := cur.String("Lang"); ok && l != "" {
			return l
		}
		pObj, ok := cur.Get("P")
		if !ok {
			return ""
		}
		parent, err := d.r.ResolveDict(pObj)
		if err != nil || parent == nil {
			return ""
		}
		cur = parent
	}
	return ""
}
