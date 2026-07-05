package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// WidgetAAContents fails when a widget annotation that carries an /AA
// additional-actions dictionary provides no /Contents description. ISO 14289-2
// §8.10.2.3 (veraPDF UA2:8.10.2.3-2): a field widget with additional actions
// shall carry a /Contents entry so assistive technology can announce what the
// action-bearing control does. Without it, a control that triggers behaviour on
// focus, blur or other events is opaque to AT.
//
// Applies to field widgets that sit in the structure tree (structParentType
// present, i.e. not an artifact). One finding per offending widget. N/A when
// the document has no such widgets. PDF/UA-2 only.
type WidgetAAContents struct{}

func (WidgetAAContents) ID() string { return "UA-28-029" }
func (WidgetAAContents) Title() string {
	return "Widget annotations with additional actions provide /Contents"
}
func (WidgetAAContents) Category() engine.Category { return engine.CategoryInteractive }
func (WidgetAAContents) Severity() engine.Severity { return engine.SeverityError }
func (WidgetAAContents) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (WidgetAAContents) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (WidgetAAContents) Description() string {
	return "ISO 14289-2 §8.10.2.3 requires a widget annotation that carries an /AA additional-actions dictionary to also carry a /Contents entry. The /Contents describes what the action-bearing control does so assistive technology can announce it; a widget that triggers behaviour on focus or other events without a description is opaque to AT. Applies to field widgets present in the structure tree."
}

func (c WidgetAAContents) Run(doc model.Document) []engine.Finding {
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
		// Applies only to field widgets that are in the structure tree
		// (not artifacts). An artifact or a non-field widget is exempt.
		if !a.IsFieldWidget || a.StructType == "" {
			continue
		}
		if !a.HasAA {
			continue
		}
		seen++
		if a.Contents != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("field widget on page %d carries an /AA additional-actions dictionary but no /Contents description", a.Page),
			Hint:     "Add a /Contents entry to the widget describing what its actions do, so assistive technology can announce the control.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no field widgets with additional actions in the structure tree -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(WidgetAAContents{}) }
