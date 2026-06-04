// Package fonts groups checks that inspect font resources -- embedding,
// encoding, ToUnicode mapping.
package fonts

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Embedded reports any font that lacks an embedded font program.
// PDF/UA-1 §7.21.4 requires every font *used* in a conforming
// document to be embedded -- without the program, rendering depends
// on a system font that may differ or be missing.
//
// Two severities:
//
//   - Error for normal fonts not embedded.
//   - Warning for the standard-14 PostScript fonts (Helvetica,
//     Times, Courier, Symbol, ZapfDingbats) when not embedded.
//     The strict spec requires their embedding too, but historic
//     PDF and PAC accept them because every viewer is expected to
//     supply them. The check still surfaces the situation so the
//     reader knows about it, but it does not break conformance.
//
// Type3 fonts are inherently embedded (their glyphs live in the PDF
// itself). Composite Type0 fonts are checked through their descendant
// CID font's /FontDescriptor.
type Embedded struct{}

func (Embedded) ID() string    { return "MH-09-001" }
func (Embedded) Title() string { return "All fonts are embedded" }
func (Embedded) Description() string {
	return "PDF/UA-1 §7.21.4 requires every font used in the document to be embedded so rendering does not depend on a system font that may differ or be missing. The standard-14 PostScript fonts (Helvetica, Times, Courier, Symbol, ZapfDingbats) are reported as a warning rather than an error when not embedded -- the strict spec requires their embedding, but viewers and validators including PAC accept them as implicitly available."
}
func (Embedded) Category() engine.Category { return engine.CategoryFonts }
func (Embedded) Severity() engine.Severity { return engine.SeverityError }
func (Embedded) Spec() engine.Spec         { return engine.SpecBoth }
func (Embedded) WCAG() []string            { return nil }

func (c Embedded) Run(doc model.Document) []engine.Finding {
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
		if f.Embedded {
			continue
		}
		if hasUsage && !used[fontKey(f)] {
			continue
		}
		if standard14[f.BaseFont] {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityWarning,
				Message:  "standard-14 font " + describe(f) + " is not embedded",
				Hint:     "PAC and most viewers accept the standard-14 fonts as implicitly available, but PDF/UA-1 strictly requires their embedding. Embed for strict conformance.",
			})
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "font " + describe(f) + " is not embedded",
			Hint:     "Re-export the PDF with all fonts embedded; non-standard fonts that fall back to system substitutes break visual fidelity and text extraction.",
		})
	}
	return findings
}

func init() { engine.Register(Embedded{}) }

// standard14 enumerates the 14 PostScript fonts every conforming PDF
// viewer is historically required to supply. ISO 32000-1 §9.6.2.2 and
// PDF/UA-1 §7.21.4 both reference them. Subset prefixes never apply
// to these (they are not subsettable when used as system fonts), so a
// plain string comparison is sufficient.
var standard14 = map[string]bool{
	"Times-Roman":           true,
	"Times-Bold":            true,
	"Times-Italic":          true,
	"Times-BoldItalic":      true,
	"Helvetica":             true,
	"Helvetica-Bold":        true,
	"Helvetica-Oblique":     true,
	"Helvetica-BoldOblique": true,
	"Courier":               true,
	"Courier-Bold":          true,
	"Courier-Oblique":       true,
	"Courier-BoldOblique":   true,
	"Symbol":                true,
	"ZapfDingbats":          true,
}

// describe builds a short, human-readable identifier from BaseFont and
// Subtype so findings point at a specific font without dumping the
// whole dict. Used by both font checks.
func describe(f model.Font) string {
	name := f.BaseFont
	if name == "" {
		name = "(no BaseFont)"
	}
	if f.Subtype != "" {
		return "'" + name + "' (" + f.Subtype + ")"
	}
	return "'" + name + "'"
}
