package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// WidgetEnclosure fails when a visible Widget annotation is not nested within a
// Form structure element. ISO 14289-1 §7.18.4 (ISO 32000-1 §14.8.4.4.2)
// requires an interactive form field's widget to be tagged with a Form
// structure element, so assistive technology presents it as a form control.
// veraPDF UA1:7.18.4-1 exempts a widget that is not presented -- hidden via /F
// or placed outside the crop box.
//
// One finding per offending Widget annotation. N/A when the document has no
// visible Widget annotations. PDF/UA-1 only (the PDF/UA-2 form, 8.10.1-1, adds
// field-widget and artifact escapes and is tracked separately).
type WidgetEnclosure struct{}

func (WidgetEnclosure) ID() string { return "UA-28-014" }
func (WidgetEnclosure) Title() string {
	return "Widget annotations are nested within a Form structure element"
}
func (WidgetEnclosure) Category() engine.Category { return engine.CategoryInteractive }
func (WidgetEnclosure) Severity() engine.Severity { return engine.SeverityError }
func (WidgetEnclosure) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (WidgetEnclosure) WCAG() []string            { return []string{"1.3.1"} }
func (WidgetEnclosure) Description() string {
	return "ISO 14289-1 §7.18.4 requires a Widget annotation (an interactive form field) to be nested within a Form structure element (ISO 32000-1 §14.8.4.4.2). A widget whose owning structure element is not a Form -- or that is not in the structure tree at all -- is not exposed as a form control. Hidden or off-page widgets are exempt because they are not presented."
}

func (c WidgetEnclosure) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	seen := 0
	for _, a := range annots {
		if a.Subtype != "Widget" {
			continue
		}
		if a.Hidden || a.OffPage {
			continue
		}
		seen++
		if a.StructType == "Form" {
			continue
		}
		where := "is not in the structure tree"
		if a.StructType != "" {
			where = "is nested in a " + a.StructType + " element"
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("Widget annotation on page %d %s, not a Form structure element", a.Page, where),
			Hint:     "Tag the form field's widget with a Form structure element that references the annotation via an OBJR.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no visible Widget annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(WidgetEnclosure{}) }
