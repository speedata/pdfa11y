package annotations

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FormRoleChild fails when a Form structure element omits a Role attribute yet
// does not have exactly one child: an object reference identifying the widget
// annotation. ISO 14289-1 §7.18.4 (veraPDF UA1:7.18.4-2): a Form element that
// carries no PrintField /Role attribute (Table 348) represents a single
// interactive control, so its /K shall be a lone object reference to the
// widget annotation (ISO 32000-1 §14.7.4.3 / §14.8.4.5, Table 340). A Form
// carrying a /Role attribute may group several controls and is exempt.
//
// One finding per offending Form element. N/A when the document has no Form
// structure elements. PDF/UA-1 only (the veraPDF rule is scoped to PDFUA_1).
type FormRoleChild struct{}

func (FormRoleChild) ID() string { return "UA-28-032" }
func (FormRoleChild) Title() string {
	return "Form without a Role attribute references exactly one widget"
}
func (FormRoleChild) Category() engine.Category { return engine.CategoryInteractive }
func (FormRoleChild) Severity() engine.Severity { return engine.SeverityError }
func (FormRoleChild) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (FormRoleChild) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (FormRoleChild) Description() string {
	return "ISO 14289-1 §7.18.4 requires a Form structure element that omits a PrintField /Role attribute (Table 348) to have only one child: an object reference identifying the widget annotation (ISO 32000-1 §14.8.4.5, Table 340). Such a Form represents a single interactive control, so it must not enclose additional structure children or reference zero or several widgets. A Form carrying a /Role attribute may group controls and is exempt."
}

func (c FormRoleChild) Run(doc model.Document) []engine.Finding {
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

func (c FormRoleChild) walk(elem model.StructElement, out *[]engine.Finding, seen *int) {
	if elem.Type() == "Form" {
		*seen++
		// A /Role attribute (PrintField owner, Table 348) exempts the Form:
		// it may group several controls.
		if elem.Attribute("Role") == "" {
			// hasOneInteractiveChild: exactly one child, an object reference
			// to a widget annotation. Structure-element children or a widget
			// count other than one both violate "only one child".
			if elem.EnclosedWidgetCount() != 1 || len(elem.Children()) != 0 {
				*out = append(*out, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  "Form structure element omits a Role attribute but does not have exactly one child referencing a widget annotation",
					Hint:     "Give the Role-less Form a single /K object reference to its widget annotation, or add a PrintField /Role attribute if it groups several controls.",
					Location: &engine.Location{Page: elem.Page()},
				})
			}
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, out, seen)
	}
}

func init() { engine.Register(FormRoleChild{}) }
