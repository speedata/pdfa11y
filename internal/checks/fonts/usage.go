package fonts

import "github.com/speedata/pdfa11y/internal/model"

// usedFontKeys builds the set of fonts that are actually referenced by
// a Tf operator on at least one page. The set is keyed by a synthetic
// identifier combining Subtype and BaseFont so that the same logical
// font referenced from multiple resource dictionaries collapses to one
// entry, matching the granularity at which doc.Fonts() reports.
//
// The second return value reports whether content-stream walking was
// available on this backend. When false, callers should fall back to
// inspecting every font in doc.Fonts() -- the alternative (silently
// suppressing all findings) would mask real problems on backends that
// have not implemented Pages() yet.
//
// When walking succeeded but observed zero Tf operators, the function
// returns (empty-map, true). That case correctly silences font checks
// on documents that declare fonts in /Resources without ever
// rendering with them (PDF/A.org's UA1_Tpdf-G2_F03 is the canonical
// example: SegoeUISymbol declared, never used).
func usedFontKeys(doc model.Document) (map[string]bool, bool) {
	pages, err := doc.Pages()
	if err != nil || len(pages) == 0 {
		return nil, false
	}
	out := map[string]bool{}
	for _, p := range pages {
		for _, f := range p.UsedFonts {
			out[fontKey(f)] = true
		}
	}
	return out, true
}

// fontKey is the identifying key used to match a model.Font from
// doc.Fonts() against an entry in PageReport.UsedFonts. Subtype +
// BaseFont is unique enough in practice; the rare collision (same
// BaseFont with different embedding state across two font dicts) is
// harmless because the result is the same finding for both.
func fontKey(f model.Font) string {
	return f.Subtype + "\x00" + f.BaseFont
}
