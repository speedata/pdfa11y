package fonts

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ToUnicode fails for every font that lacks a way to map glyph codes
// back to Unicode characters. PDF/UA-1 §7.21.3.1 accepts any of:
//
//   a) a /ToUnicode CMap on the font dictionary, OR
//   b) a simple font (Type1, TrueType, MMType1) with /Encoding set
//      to one of the predefined names (WinAnsiEncoding,
//      MacRomanEncoding, MacExpertEncoding) and no /Differences
//      override -- the encoding fixes the Unicode mapping by itself.
//
// Type0 / composite fonts always need /ToUnicode in the current
// implementation; full handling of their predefined CMaps is a
// planned refinement.
type ToUnicode struct{}

func (ToUnicode) ID() string    { return "MH-10-001" }
func (ToUnicode) Title() string { return "All fonts have a /ToUnicode map" }
func (ToUnicode) Description() string {
	return "PDF/UA-1 §7.21.3 requires every font to expose a Unicode mapping so text extraction, copy-paste and screen-reader output can reconstruct the actual characters. A /ToUnicode CMap is the most general solution; simple fonts using the predefined WinAnsiEncoding / MacRomanEncoding / MacExpertEncoding without a /Differences override also qualify, because the encoding table implies the mapping."
}
func (ToUnicode) Category() engine.Category { return engine.CategoryFonts }
func (ToUnicode) Severity() engine.Severity { return engine.SeverityError }
func (ToUnicode) Spec() engine.Spec         { return engine.SpecBoth }
func (ToUnicode) WCAG() []string            { return []string{"1.3.1"} }

func (c ToUnicode) Run(doc model.Document) []engine.Finding {
	fonts, err := doc.Fonts()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate fonts: " + err.Error(),
		}}
	}
	used, hasUsage := usedFontKeys(doc)
	var findings []engine.Finding
	for _, f := range fonts {
		if f.HasUnicodeMapping {
			continue
		}
		if hasUsage && !used[fontKey(f)] {
			continue
		}
		msg := "font " + describe(f) + " has no Unicode mapping (no /ToUnicode CMap and no predefined /Encoding)"
		hint := "Embed a /ToUnicode CMap, or for simple fonts set /Encoding to WinAnsiEncoding (or MacRomanEncoding / MacExpertEncoding) without a /Differences override."
		if f.IsSymbolic {
			msg = "symbolic font " + describe(f) + " has no /ToUnicode CMap; the predefined /Encoding shortcut does not apply because the font carries a custom glyph repertoire"
			hint = "Symbolic fonts (FontDescriptor /Flags bit 3) need an explicit /ToUnicode CMap -- their glyphs are not standard ASCII/Latin characters, so text extraction has no way to recover the underlying Unicode without one."
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  msg,
			Hint:     hint,
		})
	}
	return findings
}

func init() { engine.Register(ToUnicode{}) }
