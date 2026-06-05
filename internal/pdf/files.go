package pdf

import (
	"fmt"

	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// AssociatedFiles walks the catalog's /AF array and each page's /AF
// array, returning one AssociatedFile per filespec found. Annotation-
// and XObject-level /AF arrays are not yet walked; documents using
// those surfaces (rare today) will produce false negatives for
// MH-12-001.
func (d *document) AssociatedFiles() ([]model.AssociatedFile, error) {
	var out []model.AssociatedFile

	cat, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	d.collectAF(cat, "Catalog", &out)

	pages, err := d.collectPages()
	if err != nil {
		// Best-effort: return whatever we got from the catalog. The
		// caller (typically a check) can still emit findings on the
		// catalog level even if the page tree was malformed.
		return out, nil
	}
	for _, p := range pages {
		d.collectAF(p.PageDict, fmt.Sprintf("Page %d", p.Number), &out)
	}
	return out, nil
}

// collectAF resolves /AF on container (catalog dict or page dict) and
// appends one AssociatedFile per entry to out. Silently ignores
// entries that cannot be resolved as dicts; a malformed /AF array
// should not break enumeration of the entries that do resolve.
func (d *document) collectAF(container *pdd.Dict, source string, out *[]model.AssociatedFile) {
	afObj, ok := container.Get("AF")
	if !ok {
		return
	}
	arr, err := d.r.ResolveArray(afObj)
	if err != nil {
		return
	}
	for _, item := range arr {
		fs, err := d.r.ResolveDict(item)
		if err != nil || fs == nil {
			continue
		}
		af := model.AssociatedFile{SourcePath: source}
		if uf, ok := fs.String("UF"); ok && uf != "" {
			af.Filename = uf
		} else if f, ok := fs.String("F"); ok {
			af.Filename = f
		}
		if rel, ok := fs.Name("AFRelationship"); ok {
			af.Relationship = string(rel)
		}
		*out = append(*out, af)
	}
}
