package pdf

import (
	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// Annotations returns one model.Annotation per /Annots entry across
// every page, with /Parent-inherited tooltips already resolved. Cached
// after the first call.
func (d *document) Annotations() ([]model.Annotation, error) {
	if d.annotationsLoaded {
		return d.annotations, nil
	}
	pages, err := d.collectPages()
	if err != nil {
		return nil, err
	}
	var out []model.Annotation
	for _, p := range pages {
		annotsObj, ok := p.PageDict.Get("Annots")
		if !ok {
			continue
		}
		arr, err := d.r.ResolveArray(annotsObj)
		if err != nil {
			continue
		}
		pageBox := d.pageBox(p.PageDict)
		for _, item := range arr {
			ad, err := d.r.ResolveDict(item)
			if err != nil || ad == nil {
				continue
			}
			out = append(out, d.annotationFromDict(ad, p.Number, pageBox))
		}
	}
	d.annotations = out
	d.annotationsLoaded = true
	return d.annotations, nil
}

// annotationFromDict snapshots the fields the UA-28 checks need.
func (d *document) annotationFromDict(ad *pdd.Dict, page int, pageBox [4]float64) model.Annotation {
	subtypeName, _ := ad.Name("Subtype")
	contents, _ := ad.String("Contents")
	nameVal, _ := ad.Name("Name")
	tooltip := d.inheritedTooltip(ad)
	fieldTooltip := d.fieldTooltip(ad)
	lang := d.inheritedLang(ad)
	structParent := -1
	if sp, ok := ad.Int("StructParent"); ok {
		structParent = int(sp)
	}
	flags, _ := ad.Int("F")
	rect, rectOK := readBoxArray(ad, "Rect")
	hasFS, fsHasAFRel := d.fileSpecAFRelationship(ad)
	_, hasAA := ad.Get("AA")
	isTextField, fieldValue, richTextValue := d.textFieldValues(ad)
	return model.Annotation{
		Page:         page,
		Subtype:      string(subtypeName),
		Contents:     contents,
		Name:         string(nameVal),
		Tooltip:      tooltip,
		FieldTooltip: fieldTooltip,
		Lang:         lang,
		StructLang:   d.structLangForParent(structParent),
		StructType:   d.structTypeForParent(structParent),
		StructAlt:    d.structAltForParent(structParent),
		StructParent: structParent,
		Hidden:       flags&2 != 0,   // bit 2: Hidden
		NoView:       flags&32 != 0,  // bit 5 (PDF flags are 1-indexed): NoView
		Invisible:    flags&1 != 0,   // bit 1: Invisible
		ToggleNoView: flags&256 != 0, // bit 9: ToggleNoView
		OffPage:      rectIsOffPage(rect, pageBox),
		ZeroSize:     rectOK && rect[2]-rect[0] == 0 && rect[3]-rect[1] == 0,

		HasFileSpec:               hasFS,
		FileSpecHasAFRelationship: fsHasAFRel,

		HasAA:         hasAA,
		IsFieldWidget: string(subtypeName) == "Widget" && d.hasFieldAncestry(ad),

		IsTextField:   isTextField,
		FieldValue:    fieldValue,
		RichTextValue: richTextValue,
		RichContents:  d.richTextEntry(ad, "RC"),
		StructHasLbl:  d.structHasLblForParent(structParent),
		StructOwnLang: d.structOwnLangForParent(structParent),
	}
}

// textFieldValues resolves the owning form field (the nearest dict carrying
// /FT, up the /Parent chain) and reports whether it is a text field (/FT /Tx)
// together with its /V (plain value) and /RV (rich-text value). /RV may be a
// text string or a text stream; a stream is decoded. Empty strings when the
// entries are absent. Eight levels bound the walk against cycles.
func (d *document) textFieldValues(annot *pdd.Dict) (isTextField bool, value, richText string) {
	cur := annot
	for range 8 {
		if ft, ok := cur.Name("FT"); ok {
			isTextField = ft == "Tx"
			value, _ = cur.String("V")
			richText = d.richTextValue(cur)
			return isTextField, value, richText
		}
		parentObj, ok := cur.Get("Parent")
		if !ok {
			return false, "", ""
		}
		parent, err := d.r.ResolveDict(parentObj)
		if err != nil || parent == nil {
			return false, "", ""
		}
		cur = parent
	}
	return false, "", ""
}

// richTextValue reads a field's /RV entry, which ISO 32000-1 §12.7.3.4 allows
// to be either a text string or a text stream. A stream is decoded to its raw
// bytes (the XHTML fragment); a string is returned as-is. Empty when absent.
func (d *document) richTextValue(field *pdd.Dict) string {
	return d.richTextEntry(field, "RV")
}

// richTextEntry reads a rich-text entry (/RV or /RC) that may be either a text
// string or a text stream (ISO 32000-1 §12.7.3.4 / §12.5.6.2). A stream is
// decoded to its raw XHTML bytes; a string is returned as-is. Empty when
// absent.
func (d *document) richTextEntry(dict *pdd.Dict, key string) string {
	if s, ok := dict.String(key); ok {
		return s
	}
	obj, ok := dict.Get(key)
	if !ok {
		return ""
	}
	if body, err := d.r.DecodeStream(obj); err == nil {
		return string(body)
	}
	return ""
}

// hasFieldAncestry reports whether the annotation carries /FT itself or
// inherits it from a /Parent up the chain -- i.e. it participates in the
// AcroForm field tree. Eight levels bound the walk against cycles.
func (d *document) hasFieldAncestry(annot *pdd.Dict) bool {
	cur := annot
	for range 8 {
		if _, ok := cur.Get("FT"); ok {
			return true
		}
		parentObj, ok := cur.Get("Parent")
		if !ok {
			return false
		}
		parent, err := d.r.ResolveDict(parentObj)
		if err != nil || parent == nil {
			return false
		}
		cur = parent
	}
	return false
}

// fileSpecAFRelationship resolves an annotation's /FS filespec (if any) and
// reports whether it is present and whether it carries an /AFRelationship
// entry. Used by UA-28-024 (ISO 14289-2 §8.9.2.4.10).
func (d *document) fileSpecAFRelationship(ad *pdd.Dict) (hasFS, hasAFRel bool) {
	fsObj, ok := ad.Get("FS")
	if !ok {
		return false, false
	}
	fs, err := d.r.ResolveDict(fsObj)
	if err != nil || fs == nil {
		return false, false
	}
	if _, ok := fs.Get("AFRelationship"); ok {
		return true, true
	}
	return true, false
}

// pageBox returns the page's visible box: CropBox if present, else
// MediaBox, else a zero rectangle (which causes rectIsOffPage to
// return false and the OffPage check to decline).
func (d *document) pageBox(pageDict *pdd.Dict) [4]float64 {
	if r, ok := readBoxArray(pageDict, "CropBox"); ok {
		return r
	}
	if r, ok := readBoxArray(pageDict, "MediaBox"); ok {
		return r
	}
	return [4]float64{}
}

// readBoxArray pulls a four-number array (typically MediaBox or
// CropBox) out of pageDict and normalises it to [llx, lly, urx, ury].
// Returns ok=false when the key is absent, the array has the wrong
// shape, or any element is not numeric.
func readBoxArray(pageDict *pdd.Dict, key string) ([4]float64, bool) {
	arr, ok := pageDict.Array(key)
	if !ok || len(arr) != 4 {
		return [4]float64{}, false
	}
	var out [4]float64
	for i := range 4 {
		v, ok := numberOf(arr[i])
		if !ok {
			return [4]float64{}, false
		}
		out[i] = v
	}
	// Normalise so [0..1] are min and [2..3] are max. Some authors
	// emit boxes with swapped corners.
	if out[0] > out[2] {
		out[0], out[2] = out[2], out[0]
	}
	if out[1] > out[3] {
		out[1], out[3] = out[3], out[1]
	}
	return out, true
}

// numberOf accepts pdd.Integer or pdd.Real and returns the value as
// a float64. Reference values are not resolved here; rect entries
// are expected to be inline numerics.
func numberOf(o pdd.Object) (float64, bool) {
	switch v := o.(type) {
	case pdd.Integer:
		return float64(v), true
	case pdd.Real:
		return float64(v), true
	}
	return 0, false
}

// rectIsOffPage reports whether annot's rectangle lies entirely
// outside pageBox. When either box is zero (failed to read), the
// function returns false -- "we don't know" is reported as "on page"
// so UA-28-008 does not produce false positives on documents we
// cannot measure.
func rectIsOffPage(annot, page [4]float64) bool {
	zero := [4]float64{}
	if annot == zero || page == zero {
		return false
	}
	// Disjoint axis-aligned rectangles: annot is entirely to the
	// left, right, below or above the page box.
	return annot[2] <= page[0] ||
		annot[0] >= page[2] ||
		annot[3] <= page[1] ||
		annot[1] >= page[3]
}

// inheritedTooltip resolves /TU starting from the annotation itself
// and walking up the /Parent chain. Form fields express their tooltip
// on the field dict rather than the per-widget annotation, and a
// field can be nested below a non-terminal field that owns the /TU.
// Eight levels is more than enough for the deepest forms ever seen
// in the wild and bounds the loop against pathological cycles.
func (d *document) inheritedTooltip(annot *pdd.Dict) string {
	if tu, ok := annot.String("TU"); ok && tu != "" {
		return tu
	}
	cur := annot
	for range 8 {
		parentObj, ok := cur.Get("Parent")
		if !ok {
			return ""
		}
		parent, err := d.r.ResolveDict(parentObj)
		if err != nil || parent == nil {
			return ""
		}
		if tu, ok := parent.String("TU"); ok && tu != "" {
			return tu
		}
		cur = parent
	}
	return ""
}

// fieldTooltip returns the /TU of the form field that owns this annotation.
// /TU is a field-level attribute (ISO 32000-1 Table 220): for a widget merged
// with its field (own /FT) it is read from the widget dict; for a widget that
// is a kid of a separate field dictionary it is read from that field (the
// nearest ancestor carrying /FT). A /TU sitting on a kid widget that is not
// itself a field is NOT a field description and is ignored -- this matches
// veraPDF's PDWidgetAnnot.TU semantics for UA1:7.18.1-3, where a radio field
// whose kid widgets carry per-widget /TU but whose field lacks /TU still fails.
// Eight levels bounds the walk against pathological cycles. Empty when no field
// is found or the field has no /TU.
func (d *document) fieldTooltip(annot *pdd.Dict) string {
	cur := annot
	for range 8 {
		if _, ok := cur.Get("FT"); ok {
			tu, _ := cur.String("TU")
			return tu
		}
		parentObj, ok := cur.Get("Parent")
		if !ok {
			return ""
		}
		parent, err := d.r.ResolveDict(parentObj)
		if err != nil || parent == nil {
			return ""
		}
		cur = parent
	}
	return ""
}

// inheritedLang resolves /Lang starting from the annotation itself and
// walking up the /Parent chain, mirroring inheritedTooltip: a widget's
// language may be declared on the owning form field rather than on the
// per-widget annotation. Returns "" when no /Lang is found within the
// bounded walk. This is the annotation's OWN language; the language it
// inherits from the structure tree is resolved separately (StructLang).
func (d *document) inheritedLang(annot *pdd.Dict) string {
	if l, ok := annot.String("Lang"); ok && l != "" {
		return l
	}
	cur := annot
	for range 8 {
		parentObj, ok := cur.Get("Parent")
		if !ok {
			return ""
		}
		parent, err := d.r.ResolveDict(parentObj)
		if err != nil || parent == nil {
			return ""
		}
		if l, ok := parent.String("Lang"); ok && l != "" {
			return l
		}
		cur = parent
	}
	return ""
}
