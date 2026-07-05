package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// AnnotationDescribed fails when a visible, non-Widget annotation has neither a
// /Contents entry nor an /Alt entry on its enclosing structure element. ISO
// 14289-1 §7.18.1 (veraPDF UA1:7.18.1-2): an annotation that a sighted user can
// perceive must expose an equivalent textual description to assistive
// technology, supplied either as the annotation's own /Contents or as /Alt on
// the structure element that encloses it.
//
// Exemptions mirror the spec: Widget annotations (their descriptions come from
// the form field, see UA-28-007), hidden annotations (/F bit 2), and
// annotations whose rectangle lies outside the crop box (not perceivable
// anyway). One finding per offending annotation. N/A when the document has no
// applicable annotations. PDF/UA-1 only.
type AnnotationDescribed struct{}

func (AnnotationDescribed) ID() string { return "UA-28-025" }
func (AnnotationDescribed) Title() string {
	return "Visible annotations expose a text description (/Contents or enclosing /Alt)"
}
func (AnnotationDescribed) Category() engine.Category { return engine.CategoryInteractive }
func (AnnotationDescribed) Severity() engine.Severity { return engine.SeverityError }
func (AnnotationDescribed) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (AnnotationDescribed) WCAG() []string            { return []string{"1.1.1"} }
func (AnnotationDescribed) Description() string {
	return "ISO 14289-1 §7.18.1 requires an annotation that is not a Widget, not hidden, and not outside the crop box to expose a textual description -- either its own /Contents entry or an /Alt entry on the enclosing structure element -- so assistive technology can convey it."
}

func (c AnnotationDescribed) Run(doc model.Document) []engine.Finding {
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
		// Exemptions: Widget (form-field rules), hidden, outside crop box.
		if a.Subtype == "Widget" || a.Hidden || a.OffPage {
			continue
		}
		seen++
		if a.Contents != "" || a.StructAlt != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("%s annotation on page %d has neither a /Contents entry nor an /Alt on its enclosing structure element", annotSubtype(a.Subtype), a.Page),
			Hint:     "Add a /Contents entry describing the annotation, or an /Alt entry on the structure element that encloses it.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no visible non-Widget annotations -- nothing to inspect",
		}}
	}
	return findings
}

// annotSubtype returns the annotation subtype for messages, defaulting to
// "annotation" when the subtype is empty.
func annotSubtype(s string) string {
	if s == "" {
		return "annotation"
	}
	return s
}

func init() { engine.Register(AnnotationDescribed{}) }
