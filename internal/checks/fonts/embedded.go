// Package fonts groups checks that inspect font resources -- embedding,
// encoding, ToUnicode mapping.
package fonts

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Embedded fails for every font dictionary that lacks an embedded font
// program. PDF/UA-1 §7.21.4 requires every font referenced in a
// conforming document to be embedded -- without the program the
// rendering depends on a system font that may render differently or
// not be available, breaking visual fidelity and (worse) text
// extraction for assistive technology.
//
// Type3 fonts are inherently embedded (their glyphs live in the PDF
// itself). Composite Type0 fonts are checked through their descendant
// CID font's /FontDescriptor.
type Embedded struct{}

func (Embedded) ID() string    { return "MH-09-001" }
func (Embedded) Title() string { return "All fonts are embedded" }
func (Embedded) Description() string {
	return "PDF/UA-1 §7.21.4 requires every font used in the document to be embedded so rendering does not depend on a system font that may differ or be missing. Type3 fonts are inherently embedded; Type0 composite fonts are checked through their descendant CID font's FontDescriptor."
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
	var findings []engine.Finding
	for _, f := range fonts {
		if f.Embedded {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "font " + describe(f) + " is not embedded",
			Hint:     "Re-export the PDF with all fonts embedded; for the standard-14 fonts (Helvetica, Times, …) embed a real font file rather than relying on the viewer.",
		})
	}
	return findings
}

func init() { engine.Register(Embedded{}) }

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
