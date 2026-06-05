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
		for _, item := range arr {
			ad, err := d.r.ResolveDict(item)
			if err != nil || ad == nil {
				continue
			}
			out = append(out, d.annotationFromDict(ad, p.Number))
		}
	}
	d.annotations = out
	d.annotationsLoaded = true
	return d.annotations, nil
}

// annotationFromDict snapshots the fields the MH-28 checks need.
func (d *document) annotationFromDict(ad *pdd.Dict, page int) model.Annotation {
	subtypeName, _ := ad.Name("Subtype")
	contents, _ := ad.String("Contents")
	tooltip := d.inheritedTooltip(ad)
	structParent := -1
	if sp, ok := ad.Int("StructParent"); ok {
		structParent = int(sp)
	}
	flags, _ := ad.Int("F")
	return model.Annotation{
		Page:         page,
		Subtype:      string(subtypeName),
		Contents:     contents,
		Tooltip:      tooltip,
		StructParent: structParent,
		Hidden:       flags&2 != 0,  // bit 2: Hidden
		NoView:       flags&32 != 0, // bit 5 (PDF flags are 1-indexed): NoView
	}
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
