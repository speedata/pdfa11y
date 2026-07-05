package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// WidgetDescribed fails when a Widget annotation (form field) exposes neither a
// /TU tooltip nor an /Alt on its enclosing structure element. ISO 14289-1
// §7.18.1 (veraPDF UA1:7.18.1-3): a form field shall have a /TU key present, or
// all of its Widget annotations shall carry an alternative description as an
// /Alt entry on the enclosing structure element. Either path satisfies the
// requirement that assistive technology can announce the field's purpose.
//
// This is the spec's exact, more permissive requirement. The stricter house
// rule UA-28-003 (FormTooltip) mandates /TU unconditionally; this check accepts
// the /Alt fallback as well, so it does not flag a UA-1 form that describes its
// widgets purely through the enclosing structure element.
//
// Exemptions mirror the spec: hidden widgets (/F bit 2) and widgets whose
// rectangle lies outside the crop box (not perceivable). One finding per
// offending widget. N/A when the document has no applicable Widget annotations.
// PDF/UA-1 only.
type WidgetDescribed struct{}

func (WidgetDescribed) ID() string { return "UA-28-026" }
func (WidgetDescribed) Title() string {
	return "Widget annotations expose a description (/TU or enclosing /Alt)"
}
func (WidgetDescribed) Category() engine.Category { return engine.CategoryInteractive }
func (WidgetDescribed) Severity() engine.Severity { return engine.SeverityError }
func (WidgetDescribed) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (WidgetDescribed) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (WidgetDescribed) Description() string {
	return "ISO 14289-1 §7.18.1 requires a form field to expose a description so assistive technology can announce it -- either a /TU tooltip on the field (inherited by its widgets) or an /Alt entry on the structure element enclosing each Widget annotation. Hidden widgets and widgets outside the crop box are exempt."
}

func (c WidgetDescribed) Run(doc model.Document) []engine.Finding {
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
		// Exemptions: hidden, outside crop box.
		if a.Hidden || a.OffPage {
			continue
		}
		seen++
		// The field's /TU (FieldTooltip -- read at the field level, not a
		// spurious /TU on a non-field kid widget) or an /Alt on the enclosing
		// structure element (StructAlt) satisfies the rule.
		if a.FieldTooltip != "" || a.StructAlt != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("Widget (form field) on page %d has neither a /TU tooltip nor an /Alt on its enclosing structure element", a.Page),
			Hint:     "Set /TU on the form field, or add an /Alt entry to the structure element that encloses the widget.",
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

func init() { engine.Register(WidgetDescribed{}) }
