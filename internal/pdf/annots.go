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
	tooltip := d.inheritedTooltip(ad)
	structParent := -1
	if sp, ok := ad.Int("StructParent"); ok {
		structParent = int(sp)
	}
	flags, _ := ad.Int("F")
	rect := readRect(ad)
	return model.Annotation{
		Page:         page,
		Subtype:      string(subtypeName),
		Contents:     contents,
		Tooltip:      tooltip,
		StructParent: structParent,
		Hidden:       flags&2 != 0,  // bit 2: Hidden
		NoView:       flags&32 != 0, // bit 5 (PDF flags are 1-indexed): NoView
		OffPage:      rectIsOffPage(rect, pageBox),
	}
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

// readRect resolves an annotation's /Rect to a normalised
// [llx, lly, urx, ury]. Returns a zero rect on parse failure; the
// caller treats a zero rect as "no information" (OffPage stays false).
func readRect(annot *pdd.Dict) [4]float64 {
	r, _ := readBoxArray(annot, "Rect")
	return r
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
