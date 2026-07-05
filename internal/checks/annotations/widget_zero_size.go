package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// WidgetZeroSize fails when a Widget annotation of zero height and width is
// present in the structure tree without being an artifact. ISO 14289-2
// §8.9.2.4.13 (veraPDF UA2:8.9.2.4.13-1, ISO 32000-2 §12.7.5.5): a zero-size
// widget has no visible appearance, so it carries no perceivable content and
// shall be treated as an artifact rather than a real, announced structure node.
//
// The rule is scoped to zero-size widgets: a widget whose /Rect has both zero
// width and zero height. A widget that is not in the structure tree (empty
// owning type) or is enclosed as an Artifact is compliant. One finding per
// offending widget. N/A when the document has no zero-size Widget annotations.
// PDF/UA-2 only.
type WidgetZeroSize struct{}

func (WidgetZeroSize) ID() string { return "UA-28-018" }
func (WidgetZeroSize) Title() string {
	return "Zero-size Widget annotations are artifacts, not tagged content"
}
func (WidgetZeroSize) Category() engine.Category { return engine.CategoryInteractive }
func (WidgetZeroSize) Severity() engine.Severity { return engine.SeverityError }
func (WidgetZeroSize) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (WidgetZeroSize) WCAG() []string            { return []string{"1.3.1"} }
func (WidgetZeroSize) Description() string {
	return "ISO 14289-2 §8.9.2.4.13 requires a Widget annotation of zero height and width to be an artifact. A zero-size widget has no visible appearance and thus no perceivable content; enclosing it as a real structure element gives assistive technology a phantom node to announce. Mark such widgets as artifacts (keep them out of logical structure)."
}

func (c WidgetZeroSize) Run(doc model.Document) []engine.Finding {
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
		if a.Subtype != "Widget" || !a.ZeroSize {
			continue
		}
		seen++
		// Not in the structure tree, or enclosed as an artifact, is compliant.
		if a.StructType == "" || a.StructType == "Artifact" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("zero-size Widget annotation on page %d is enclosed in a %s structure element instead of being an artifact", a.Page, a.StructType),
			Hint:     "Mark the zero-size widget as an artifact (remove it from logical structure); it has no perceivable content.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no zero-size Widget annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(WidgetZeroSize{}) }
