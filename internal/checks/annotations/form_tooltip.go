package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FormTooltip fails for every Widget annotation (form field) that
// has no /TU value, either on the widget itself or inherited through
// its /Parent chain. PDF/UA-1 §7.18.2 requires form fields to carry
// a tooltip so screen readers can announce their purpose ("First
// name", "Date of birth"); the visible label next to the field is
// not enough -- it lives in the page content, not on the field, and
// AT cannot reliably associate them by geometry.
//
// Hidden / NoView widgets are skipped (not user-interactive).
type FormTooltip struct{}

func (FormTooltip) ID() string                { return "UA-28-003" }
func (FormTooltip) Title() string             { return "Form fields have /TU tooltip" }
func (FormTooltip) Category() engine.Category { return engine.CategoryInteractive }
func (FormTooltip) Severity() engine.Severity { return engine.SeverityError }
func (FormTooltip) Spec() engine.Spec         { return engine.SpecBoth }
func (FormTooltip) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (FormTooltip) Description() string {
	return "PDF/UA-1 §7.18.2 requires every form field to expose a /TU tooltip so AT can announce its purpose. The visible label rendered on the page is not enough: AT cannot reliably associate label glyphs with field widgets by geometry. /TU may live on the widget itself or be inherited from a Parent field dictionary."
}

func (c FormTooltip) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	widgetCount := 0
	for _, a := range annots {
		if a.Subtype != "Widget" {
			continue
		}
		widgetCount++
		if a.Hidden || a.NoView {
			continue
		}
		if a.Tooltip == "" {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("Widget (form field) on page %d has no /TU tooltip", a.Page),
				Hint:     "Set /TU on the form field (or the widget annotation) to the spoken label users should hear, e.g. 'First name'.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if widgetCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Widget (form field) annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(FormTooltip{}) }
