package fonts

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ToUnicode fails for every font that has no /ToUnicode CMap. PDF/UA-1
// §7.21.3 requires a ToUnicode mapping so that text extraction, copy-
// paste and screen-reader output can reconstruct the actual characters
// rather than rendering glyph indices.
type ToUnicode struct{}

func (ToUnicode) ID() string    { return "MH-10-001" }
func (ToUnicode) Title() string { return "All fonts have a /ToUnicode map" }
func (ToUnicode) Description() string {
	return "Without a /ToUnicode CMap, text extraction returns glyph codes instead of characters -- breaking copy-paste, search, and screen-reader output. PDF/UA-1 §7.21.3 requires every font to ship this mapping."
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
	var findings []engine.Finding
	for _, f := range fonts {
		if f.HasToUnicode {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "font " + describe(f) + " has no /ToUnicode map",
			Hint:     "Embed a /ToUnicode CMap that maps glyph codes back to Unicode characters so text extraction works.",
		})
	}
	return findings
}

func init() { engine.Register(ToUnicode{}) }
