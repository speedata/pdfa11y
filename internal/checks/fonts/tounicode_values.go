package fonts

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ToUnicodeValues fails when a font's /ToUnicode CMap maps a glyph code to a
// forbidden Unicode value. ISO 14289-1 §7.21.7 (and ISO 14289-2 §8.4.5.8)
// require every value a ToUnicode CMap produces to be greater than U+0000 and
// different from U+FEFF and U+FFFE:
//
//   - U+0000 (NULL) carries no character, so it defeats text extraction;
//   - U+FEFF (zero-width no-break space / BOM) and U+FFFE (a noncharacter)
//     are not real content and confuse text processing.
//
// One finding per offending font. N/A when no font carries a /ToUnicode CMap.
type ToUnicodeValues struct{}

func (ToUnicodeValues) ID() string                { return "UA-10-003" }
func (ToUnicodeValues) Title() string             { return "/ToUnicode values are valid Unicode" }
func (ToUnicodeValues) Category() engine.Category { return engine.CategoryFonts }
func (ToUnicodeValues) Severity() engine.Severity { return engine.SeverityError }
func (ToUnicodeValues) Spec() engine.Spec         { return engine.SpecBoth }
func (ToUnicodeValues) WCAG() []string            { return []string{"1.3.1"} }
func (ToUnicodeValues) Description() string {
	return "ISO 14289-1 §7.21.7 / ISO 14289-2 §8.4.5.8 require every Unicode value a /ToUnicode CMap maps to be greater than U+0000 and not equal to U+FEFF or U+FFFE. U+0000 carries no character; U+FEFF (BOM/ZWNBSP) and U+FFFE (a noncharacter) are not real content -- any of them breaks the text extraction the CMap exists to enable."
}

// forbiddenToUnicode reports whether r is a Unicode value a /ToUnicode CMap
// must not produce.
func forbiddenToUnicode(r rune) bool {
	return r == 0x0000 || r == 0xFEFF || r == 0xFFFE
}

func (c ToUnicodeValues) Run(doc model.Document) []engine.Finding {
	fonts, err := doc.Fonts()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate fonts: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	withCMap := 0
	for _, f := range fonts {
		if f.ToUnicodeMappings == nil {
			continue
		}
		withCMap++
		if bad, ok := firstForbiddenValue(f.ToUnicodeMappings); ok {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("font %s maps a glyph code to the forbidden Unicode value %s via /ToUnicode", describe(f), bad),
				Hint:     "Remove or correct the /ToUnicode entries that produce U+0000, U+FEFF or U+FFFE; map each glyph to its real character.",
			})
		}
	}
	if withCMap == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no font carries a /ToUnicode CMap -- nothing to inspect",
		}}
	}
	return findings
}

// firstForbiddenValue returns a formatted U+XXXX for the first forbidden
// Unicode value found among the CMap's mapped strings, and whether one exists.
func firstForbiddenValue(mappings map[uint32]string) (string, bool) {
	for _, s := range mappings {
		for _, r := range s {
			if forbiddenToUnicode(r) {
				return fmt.Sprintf("U+%04X", r), true
			}
		}
	}
	return "", false
}

func init() { engine.Register(ToUnicodeValues{}) }
