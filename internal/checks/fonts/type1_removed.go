package fonts

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// Type1Removed enforces the PDF 2.0 removal of Type 1 (PostScript)
// fonts under PDF/UA-2. ISO 32000-2 dropped the Type 1 / MMType 1
// font formats; PDF/UA-2 inherits that removal, so a conforming
// PDF/UA-2 document must not reference them. The 14 Standard PDF
// fonts (Helvetica, Times, Courier, Symbol, ZapfDingbats) are Type 1
// and must be replaced with TrueType or OpenType equivalents.
//
// The check is gated twice: declared Spec is PDF/UA-2 (so the CLI
// filters it out for --spec=pdfua1 runs), and the body re-checks
// pdfuaid:part so engine.All()-based callers (tests, the realworld
// driver) also see N/A on PDF/UA-1 documents instead of false
// positives.
type Type1Removed struct{}

func (Type1Removed) ID() string                { return "MH-09-003" }
func (Type1Removed) Title() string             { return "PDF/UA-2 documents do not use Type 1 fonts" }
func (Type1Removed) Category() engine.Category { return engine.CategoryFonts }
func (Type1Removed) Severity() engine.Severity { return engine.SeverityError }
func (Type1Removed) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (Type1Removed) WCAG() []string            { return nil }
func (Type1Removed) Description() string {
	return "PDF 2.0 (ISO 32000-2) removed the Type 1 and MMType 1 font formats. PDF/UA-2 inherits that removal: a conforming PDF/UA-2 document must reference neither. The 14 Standard PDF fonts (Helvetica, Times, Courier, Symbol, ZapfDingbats) are Type 1 and must be replaced with TrueType or OpenType (Type 0 / CIDFontType 2) equivalents."
}

func (c Type1Removed) Run(doc model.Document) []engine.Finding {
	part, found, err := pdfua.DetectPart(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata: " + err.Error(),
		}}
	}
	if !found || part != 2 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "check applies to PDF/UA-2 only; document does not declare pdfuaid:part = 2",
		}}
	}
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
		if f.Subtype != "Type1" && f.Subtype != "MMType1" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("font %q is %s -- the format is removed by PDF 2.0 / PDF/UA-2", f.BaseFont, f.Subtype),
			Hint:     "Replace the font with a TrueType or OpenType (Type 0 / CIDFontType 2) equivalent. The 14 Standard PDF fonts are Type 1 and need substitution.",
		})
	}
	return findings
}

func init() { engine.Register(Type1Removed{}) }
