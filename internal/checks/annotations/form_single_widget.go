package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FormSingleWidget fails when a Form structure element encloses more than one
// widget annotation. ISO 14289-2 §8.10.1 (veraPDF UA2:8.10.1-2): a Form
// structure element shall contain at most one widget annotation, so assistive
// technology can present a one-to-one mapping between the tagged form control
// and the interactive widget. A Form gathering several widgets under a single
// tag is ambiguous -- AT cannot tell which control the label and role apply to.
//
// One finding per offending Form element. N/A when the document has no Form
// structure elements. PDF/UA-2 only.
type FormSingleWidget struct{}

func (FormSingleWidget) ID() string { return "UA-28-027" }
func (FormSingleWidget) Title() string {
	return "Form structure elements enclose at most one widget annotation"
}
func (FormSingleWidget) Category() engine.Category { return engine.CategoryInteractive }
func (FormSingleWidget) Severity() engine.Severity { return engine.SeverityError }
func (FormSingleWidget) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (FormSingleWidget) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (FormSingleWidget) Description() string {
	return "ISO 14289-2 §8.10.1 requires a Form structure element to enclose at most one widget annotation. A Form that gathers several widgets under one tag is ambiguous: assistive technology cannot map the tagged control's label and role to a single interactive widget. Split each widget into its own Form element."
}

func (c FormSingleWidget) Run(doc model.Document) []engine.Finding {
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	seen := 0
	c.walk(root, &findings, &seen)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no Form structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c FormSingleWidget) walk(elem model.StructElement, out *[]engine.Finding, seen *int) {
	if elem.Type() == "Form" {
		*seen++
		if n := elem.EnclosedWidgetCount(); n > 1 {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("Form structure element encloses %d widget annotations; at most one is allowed", n),
				Hint:     "Give each widget annotation its own Form structure element so assistive technology can map the tagged control to a single widget.",
				Location: &engine.Location{Page: elem.Page()},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, out, seen)
	}
}

func init() { engine.Register(FormSingleWidget{}) }
