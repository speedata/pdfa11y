package fonts

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// NotdefGlyph fails when a text-showing operator references the .notdef glyph.
// ISO 14289-1 §7.21.8 (and ISO 14289-2 §8.4.5.9, veraPDF UA1:7.21.8-1 /
// UA2:8.4.5.9-1): a conforming file shall not reference the .notdef glyph from
// any text-showing operator, in any content stream, regardless of text
// rendering mode. The .notdef glyph carries no character, so text that renders
// through it is invisible to text extraction and assistive technology.
//
// Detection scope: a composite (Type0) font with Identity-H/Identity-V
// encoding shows code 0, which is CID 0 = glyph index 0 = .notdef, when the
// descendant is a CFF CIDFont or a CIDFontType2 with an Identity /CIDToGIDMap.
// A CIDFontType2 whose /CIDToGIDMap is a remapping stream is skipped -- glyph 0
// cannot be confirmed without parsing the map. Simple-font .notdef references
// (an unmapped code or a glyph explicitly named .notdef via /Differences) are
// not covered here; they need font-program glyph analysis.
//
// One finding per offending font-on-page. N/A when no font renders any text.
type NotdefGlyph struct{}

func (NotdefGlyph) ID() string                { return "UA-10-004" }
func (NotdefGlyph) Title() string             { return "No text-showing operator references the .notdef glyph" }
func (NotdefGlyph) Category() engine.Category { return engine.CategoryFonts }
func (NotdefGlyph) Severity() engine.Severity { return engine.SeverityError }
func (NotdefGlyph) Spec() engine.Spec         { return engine.SpecBoth }
func (NotdefGlyph) WCAG() []string            { return []string{"1.3.1"} }
func (NotdefGlyph) Description() string {
	return "ISO 14289-1 §7.21.8 / ISO 14289-2 §8.4.5.9 forbid referencing the .notdef glyph from any text-showing operator. The .notdef glyph carries no character, so text drawn through it cannot be extracted or voiced. This check flags a composite (Type0) Identity-encoded font that shows code 0 -- CID 0, glyph index 0, the .notdef glyph."
}

func (c NotdefGlyph) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	sawFont := false
	var findings []engine.Finding
	for _, p := range pages {
		// Deterministic order for stable reporting.
		keys := make([]string, 0, len(p.FontCodes))
		for k := range p.FontCodes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			f, ok := p.UsedFonts[key]
			if !ok {
				continue
			}
			sawFont = true
			if referencesNotdef(f, p.FontCodes[key]) {
				name := f.BaseFont
				if name == "" {
					name = key
				}
				findings = append(findings, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("page %d: font %s shows code 0, referencing the .notdef glyph (CID 0 / glyph index 0)", p.Number, name),
					Hint:     "Remove the text-showing operation that emits code 0, or map that character to a real glyph. The .notdef glyph is not readable content.",
					Location: &engine.Location{Page: p.Number},
				})
			}
		}
	}
	if !sawFont {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no font renders any text -- nothing to inspect",
		}}
	}
	return findings
}

// referencesNotdef reports whether font f, having shown the given set of glyph
// codes, references the .notdef glyph via a code-0 show under Identity
// encoding. Restricted to the cases where code 0 provably maps to glyph 0.
func referencesNotdef(f model.Font, codes map[uint32]bool) bool {
	if f.Subtype != "Type0" {
		return false
	}
	if f.Encoding != "Identity-H" && f.Encoding != "Identity-V" {
		return false
	}
	switch f.CIDSubtype {
	case "CIDFontType0":
		// CFF CIDFont: CID 0 is .notdef by convention (glyph index 0).
	case "CIDFontType2":
		// TrueType CIDFont: code 0 = CID 0 = glyph 0 only under an Identity
		// CIDToGIDMap. A remapping stream might send CID 0 elsewhere.
		if f.CIDToGIDMap != "Identity" {
			return false
		}
	default:
		return false
	}
	return codes[0]
}

func init() { engine.Register(NotdefGlyph{}) }
