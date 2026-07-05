package pdf

import (
	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// MediaClips enumerates the media clip data dictionaries reachable from the
// document's Screen annotations. The path is Screen annotation -> /A Rendition
// action -> /R rendition -> /C media clip (ISO 32000-1 §13.2). Selector
// renditions (/S /SR) fan out to a /R array of sub-renditions, which are
// followed recursively. Only media clip *data* dictionaries (/S /MCD, the ones
// that carry /CT and /Alt) are reported.
func (d *document) MediaClips() []model.MediaClip {
	pages, err := d.collectPages()
	if err != nil {
		return nil
	}
	var out []model.MediaClip
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
			if name, _ := ad.Name("Subtype"); string(name) != "Screen" {
				continue
			}
			for _, action := range d.resolveDictOrArray(ad, "A") {
				d.collectMediaClipsFromAction(action, p.Number, &out)
			}
		}
	}
	return out
}

// collectMediaClipsFromAction follows a Rendition action's /R rendition to the
// media clip(s) it references.
func (d *document) collectMediaClipsFromAction(action *pdd.Dict, page int, out *[]model.MediaClip) {
	if s, _ := action.Name("S"); string(s) != "Rendition" {
		return
	}
	rendObj, ok := action.Get("R")
	if !ok {
		return
	}
	rend, err := d.r.ResolveDict(rendObj)
	if err != nil || rend == nil {
		return
	}
	d.collectMediaClipsFromRendition(rend, page, out, 0)
}

// collectMediaClipsFromRendition resolves a rendition to its media clip. A
// media rendition (/S /MR) points at a single clip through /C; a selector
// rendition (/S /SR) fans out to a /R array of sub-renditions. depth bounds the
// recursion against pathological cycles.
func (d *document) collectMediaClipsFromRendition(rend *pdd.Dict, page int, out *[]model.MediaClip, depth int) {
	if depth > 20 {
		return
	}
	switch s, _ := rend.Name("S"); string(s) {
	case "MR":
		clipObj, ok := rend.Get("C")
		if !ok {
			return
		}
		clip, err := d.r.ResolveDict(clipObj)
		if err != nil || clip == nil {
			return
		}
		if mc, ok := d.readMediaClip(clip, page); ok {
			*out = append(*out, mc)
		}
	case "SR":
		subs, err := d.r.ResolveArray(mustGet(rend, "R"))
		if err != nil {
			return
		}
		for _, sub := range subs {
			subRend, err := d.r.ResolveDict(sub)
			if err != nil || subRend == nil {
				continue
			}
			d.collectMediaClipsFromRendition(subRend, page, out, depth+1)
		}
	}
}

// readMediaClip snapshots the /CT and /Alt entries of a media clip data
// dictionary (/S /MCD). Returns ok=false when the dictionary is not a media
// clip data dictionary.
func (d *document) readMediaClip(clip *pdd.Dict, page int) (model.MediaClip, bool) {
	if s, _ := clip.Name("S"); string(s) != "MCD" {
		return model.MediaClip{}, false
	}
	mc := model.MediaClip{Page: page}
	if _, ok := clip.Get("CT"); ok {
		mc.HasCT = true
	}
	mc.HasCorrectAlt = d.altHasDefaultText(clip)
	return mc, true
}

// altHasDefaultText reports whether the media clip's /Alt array contains a
// default entry -- a (culture, text) pair whose culture string is empty --
// carrying a non-empty text string. The array alternates culture and text
// strings (ISO 32000-1 §13.2.4.2).
func (d *document) altHasDefaultText(clip *pdd.Dict) bool {
	altObj, ok := clip.Get("Alt")
	if !ok {
		return false
	}
	arr, err := d.r.ResolveArray(altObj)
	if err != nil {
		return false
	}
	for i := 0; i+1 < len(arr); i += 2 {
		culture, cok := d.r.Resolve(arr[i])
		text, tok := d.r.Resolve(arr[i+1])
		if cok != nil || tok != nil {
			continue
		}
		cs, cIsStr := culture.(pdd.String)
		ts, tIsStr := text.(pdd.String)
		if cIsStr && tIsStr && len(cs) == 0 && len(ts) > 0 {
			return true
		}
	}
	return false
}

// resolveDictOrArray resolves container[key], which may be either a single
// dictionary or an array of dictionaries, into a slice of dictionaries.
func (d *document) resolveDictOrArray(container *pdd.Dict, key string) []*pdd.Dict {
	obj, ok := container.Get(key)
	if !ok {
		return nil
	}
	resolved, err := d.r.Resolve(obj)
	if err != nil {
		return nil
	}
	switch v := resolved.(type) {
	case *pdd.Dict:
		return []*pdd.Dict{v}
	case pdd.Array:
		var out []*pdd.Dict
		for _, item := range v {
			if dict, err := d.r.ResolveDict(item); err == nil && dict != nil {
				out = append(out, dict)
			}
		}
		return out
	}
	return nil
}

// mustGet returns dict[key] or nil; a nil object handed to ResolveArray simply
// yields an error, which callers already tolerate.
func mustGet(dict *pdd.Dict, key string) pdd.Object {
	obj, _ := dict.Get(key)
	return obj
}
