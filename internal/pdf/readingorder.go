package pdf

import (
	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// StructTreeOrder linearises the structure tree depth-first, emitting
// one ReadingOrderEntry per MCID leaf. The order of the returned slice
// is the structure tree's intended reading order — what a check
// compares against the geometric layout from PageReport.MCIDBoxes.
//
// Page numbers are resolved through the structure-tree /Pg inheritance
// chain; entries whose /Pg never resolves carry Page = 0.
func (d *document) StructTreeOrder() ([]model.ReadingOrderEntry, error) {
	pages, err := d.collectPages()
	if err != nil {
		return nil, err
	}
	refToNum := make(map[pdd.Reference]int, len(pages))
	for _, p := range pages {
		refToNum[p.Ref] = p.Number
	}

	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return nil, err
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return nil, nil
	}

	var out []model.ReadingOrderEntry
	d.walkStructTreeOrder(stree, pdd.Reference{}, "", refToNum, &out)
	return out, nil
}

// walkStructTreeOrder recursively walks one structure element and
// appends its MCID leaves to out in DFS order. inheritedPage carries
// the most specific /Pg seen so far; path is the slash-joined chain of
// /S tags from the root down to elem's parent.
func (d *document) walkStructTreeOrder(
	elem *pdd.Dict,
	inheritedPage pdd.Reference,
	path string,
	refToNum map[pdd.Reference]int,
	out *[]model.ReadingOrderEntry,
) {
	if elem == nil {
		return
	}

	page := inheritedPage
	if pg, ok := elem.Get("Pg"); ok {
		if ref, ok := pg.(pdd.Reference); ok {
			page = ref
		}
	}

	childPath := path
	if tag, ok := elem.Name("S"); ok && len(tag) > 0 {
		if childPath == "" {
			childPath = string(tag)
		} else {
			childPath = childPath + "/" + string(tag)
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
		d.handleStructKEntryOrder(item, page, childPath, refToNum, out)
	}
}

func (d *document) handleStructKEntryOrder(
	item pdd.Object,
	page pdd.Reference,
	path string,
	refToNum map[pdd.Reference]int,
	out *[]model.ReadingOrderEntry,
) {
	if n, ok := asInt(item); ok {
		*out = append(*out, model.ReadingOrderEntry{
			Page:       refToNum[page],
			MCID:       int(n),
			StructPath: path,
		})
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
		*out = append(*out, model.ReadingOrderEntry{
			Page:       refToNum[ref],
			MCID:       int(mcid),
			StructPath: path,
		})
	case "OBJR":
		return
	default:
		d.walkStructTreeOrder(child, page, path, refToNum, out)
	}
}
