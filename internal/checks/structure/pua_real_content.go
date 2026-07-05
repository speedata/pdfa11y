package structure

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// PUARealContent fails when real page content maps to a Unicode Private Use
// Area value without an /ActualText or /Alt entry to give its readable
// meaning. ISO 14289-2 §8.4.3 (veraPDF UA2:8.4.3-1): in all cases where real
// content maps to Unicode PUA values, an /ActualText or /Alt entry shall be
// present. PUA code points carry no assigned meaning, so text that extracts to
// them is unreadable unless an alternate representation is supplied -- on the
// marked-content sequence itself or on an enclosing structure element.
//
// Detection uses the glyphs' /ToUnicode mapping; content in a font without a
// /ToUnicode CMap cannot be classified here (UA-10-001 requires that CMap
// separately). One finding per offending page. N/A when no real content maps
// to PUA. PDF/UA-2 only.
type PUARealContent struct{}

func (PUARealContent) ID() string { return "UA-01-020" }
func (PUARealContent) Title() string {
	return "Real content mapping to Private Use Area has ActualText or Alt"
}
func (PUARealContent) Category() engine.Category { return engine.CategoryRealContent }
func (PUARealContent) Severity() engine.Severity { return engine.SeverityError }
func (PUARealContent) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (PUARealContent) WCAG() []string            { return []string{"1.1.1"} }
func (PUARealContent) Description() string {
	return "ISO 14289-2 §8.4.3 requires an /ActualText or /Alt entry wherever real content maps to Unicode Private Use Area values. PUA code points have no assigned meaning, so extracted text is unreadable unless an alternate representation is supplied on the marked-content sequence or an enclosing structure element. Detection relies on the /ToUnicode mapping of the shown glyphs."
}

func (c PUARealContent) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no pages -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	sawPUA := false
	for _, p := range pages {
		if p.HadPUARealContent {
			sawPUA = true
		}
		if len(p.PUAContentMCIDs) == 0 {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("page %d has real content mapping to Unicode Private Use Area values with no /ActualText or /Alt (%d marked-content sequence(s))", p.Number, len(p.PUAContentMCIDs)),
			Hint:     "Add an /ActualText (or /Alt) entry -- on the marked-content sequence or its enclosing structure element -- giving the readable text for the Private Use Area content.",
			Location: &engine.Location{Page: p.Number},
		})
	}
	if !sawPUA {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no real content maps to Unicode Private Use Area values -- nothing to inspect",
		}}
	}
	return findings // nil (pass) when PUA content was present but all covered
}

func init() { engine.Register(PUARealContent{}) }
