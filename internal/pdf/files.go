package pdf

import (
	"fmt"
	"strings"

	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// AssociatedFiles walks the catalog's /AF array and each page's /AF
// array, returning one AssociatedFile per filespec found. Annotation-
// and XObject-level /AF arrays are not yet walked; documents using
// those surfaces (rare today) will produce false negatives for
// UA-12-001.
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

// EmbeddedFileSpecs walks the catalog's /Names /EmbeddedFiles name tree and
// returns one snapshot per file specification dictionary found. Returns an
// empty slice when the document has no such tree.
func (d *document) EmbeddedFileSpecs() []model.EmbeddedFileSpec {
	cat, err := d.r.Catalog()
	if err != nil {
		return nil
	}
	names, ok := cat.Dict("Names")
	if !ok {
		return nil
	}
	efObj, ok := names.Get("EmbeddedFiles")
	if !ok {
		return nil
	}
	var out []model.EmbeddedFileSpec
	d.walkNameTree(efObj, &out, 0)
	return out
}

// walkNameTree recurses through a PDF name tree (ISO 32000-1 §7.9.6) rooted at
// nodeObj, appending one EmbeddedFileSpec per value entry. A node carries
// either a /Names leaf array ([key1 val1 key2 val2 ...]) or a /Kids array of
// child nodes; both are handled. depth guards against pathological cycles.
func (d *document) walkNameTree(nodeObj pdd.Object, out *[]model.EmbeddedFileSpec, depth int) {
	if depth > 50 {
		return
	}
	node, err := d.r.ResolveDict(nodeObj)
	if err != nil || node == nil {
		return
	}
	if namesObj, ok := node.Get("Names"); ok {
		if arr, err := d.r.ResolveArray(namesObj); err == nil {
			// Entries alternate key, value; the values are filespec dicts.
			for i := 1; i < len(arr); i += 2 {
				fs, err := d.r.ResolveDict(arr[i])
				if err != nil || fs == nil {
					continue
				}
				*out = append(*out, d.readEmbeddedFileSpec(fs))
			}
		}
	}
	if kidsObj, ok := node.Get("Kids"); ok {
		if arr, err := d.r.ResolveArray(kidsObj); err == nil {
			for _, kid := range arr {
				d.walkNameTree(kid, out, depth+1)
			}
		}
	}
}

// readEmbeddedFileSpec snapshots the filespec fields UA-12-002 needs.
func (d *document) readEmbeddedFileSpec(fs *pdd.Dict) model.EmbeddedFileSpec {
	var e model.EmbeddedFileSpec
	f, fok := fs.String("F")
	uf, ufok := fs.String("UF")
	e.HasNonEmptyF = fok && f != ""
	e.HasNonEmptyUF = ufok && uf != ""
	if e.HasNonEmptyUF {
		e.Filename = uf
	} else if e.HasNonEmptyF {
		e.Filename = f
	}
	if _, ok := fs.Get("Desc"); ok {
		e.HasDesc = true
	}
	if _, ok := fs.Get("EF"); ok {
		e.HasEF = true
	}
	return e
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
		af.Subtype = d.embeddedFileSubtype(fs)
		af.Content = d.embeddedFileContent(fs)
		*out = append(*out, af)
	}
}

// embeddedFileSubtype resolves /EF/F (or /EF/UF) on the filespec to
// the EmbeddedFile stream dict and returns its /Subtype as a MIME
// string. PDF encodes the MIME type as a Name with '/' written
// '#2F'; we normalise it back to the conventional MIME form so
// callers can compare against literals like "application/mathml+xml".
// Returns "" when /EF is missing, the stream cannot be resolved, or
// /Subtype is absent.
func (d *document) embeddedFileSubtype(fs *pdd.Dict) string {
	ef, ok := fs.Dict("EF")
	if !ok {
		return ""
	}
	streamObj, ok := ef.Get("F")
	if !ok {
		streamObj, ok = ef.Get("UF")
		if !ok {
			return ""
		}
	}
	resolved, err := d.r.Resolve(streamObj)
	if err != nil {
		return ""
	}
	var streamDict *pdd.Dict
	switch v := resolved.(type) {
	case *pdd.Dict:
		streamDict = v
	case *pdd.Stream:
		streamDict = v.Dict
	default:
		return ""
	}
	name, ok := streamDict.Name("Subtype")
	if !ok {
		return ""
	}
	// PDF Names use '#2F' as the escape for '/'. pdfdisassembler hands
	// the Name back already decoded in some cases and raw in others;
	// normalise both forms.
	s := string(name)
	s = strings.ReplaceAll(s, "#2F", "/")
	s = strings.ReplaceAll(s, "#2f", "/")
	return s
}

// embeddedFileContent resolves /EF/F on the filespec to the
// EmbeddedFile stream and returns its decoded bytes. Returns nil
// when /EF is missing, the stream cannot be resolved, or decoding
// fails -- callers (currently UA-17-004) treat nil as "no content
// available" rather than retry.
func (d *document) embeddedFileContent(fs *pdd.Dict) []byte {
	ef, ok := fs.Dict("EF")
	if !ok {
		return nil
	}
	streamObj, ok := ef.Get("F")
	if !ok {
		streamObj, ok = ef.Get("UF")
		if !ok {
			return nil
		}
	}
	content, err := d.DecodeStream(streamObj)
	if err != nil {
		return nil
	}
	return content
}
