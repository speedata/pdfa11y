package annotations

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// WidgetLabelContents fails when a field widget in the structure tree has
// neither an associated label (a Lbl structure element) nor a /Contents entry.
// ISO 14289-2 §8.10.2.3 (veraPDF UA2:8.10.2.3-1): if a label for a widget
// annotation is not present, a /Contents entry shall be provided to supply the
// widget's description and context. Without either, assistive technology has no
// name to announce for the form control.
//
// A widget that is not a form field, not in the structure tree, or an artifact
// is out of scope. One finding per offending widget. N/A when no field widget
// in the structure tree needs inspection. PDF/UA-2 only.
type WidgetLabelContents struct{}

func (WidgetLabelContents) ID() string                { return "UA-28-035" }
func (WidgetLabelContents) Title() string             { return "Field widget has a label or a Contents entry" }
func (WidgetLabelContents) Category() engine.Category { return engine.CategoryInteractive }
func (WidgetLabelContents) Severity() engine.Severity { return engine.SeverityError }
func (WidgetLabelContents) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (WidgetLabelContents) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (WidgetLabelContents) Description() string {
	return "ISO 14289-2 §8.10.2.3 requires a field widget that has no associated label (a Lbl structure element enclosed with it) to provide a /Contents entry describing the control. A widget with neither leaves assistive technology without a name to announce for the form field."
}

func (c WidgetLabelContents) Run(doc model.Document) []engine.Finding {
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
		// Applies only to field widgets that are in the structure tree and
		// not artifacts (an artifact or non-field widget is exempt).
		if !a.IsFieldWidget || a.StructType == "" || a.StructType == "Artifact" {
			continue
		}
		seen++
		if a.StructHasLbl || a.Contents != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "field widget has neither an associated Lbl label nor a /Contents entry",
			Hint:     "Add a Lbl structure element labelling the widget, or give the widget annotation a /Contents entry describing the control.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no field widget in the structure tree -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(WidgetLabelContents{}) }
