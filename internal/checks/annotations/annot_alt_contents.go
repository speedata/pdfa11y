package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// AnnotAltContents fails when an annotation carries a /Contents entry and its
// directly enclosing structure element carries an /Alt entry, but the two
// differ. ISO 14289-2 §8.9.4.2 (veraPDF UA2:8.9.4.2-1): where both are present
// their values shall be identical, so assistive technology does not announce
// two conflicting descriptions for the same object.
//
// Only fires when both texts are present. One finding per offending
// annotation. N/A when no annotation has both a /Contents and an enclosing
// /Alt. PDF/UA-2 only.
type AnnotAltContents struct{}

func (AnnotAltContents) ID() string { return "UA-28-031" }
func (AnnotAltContents) Title() string {
	return "Annotation /Contents matches the enclosing element's /Alt"
}
func (AnnotAltContents) Category() engine.Category { return engine.CategoryInteractive }
func (AnnotAltContents) Severity() engine.Severity { return engine.SeverityError }
func (AnnotAltContents) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (AnnotAltContents) WCAG() []string            { return []string{"1.3.1"} }
func (AnnotAltContents) Description() string {
	return "ISO 14289-2 §8.9.4.2 requires that when an annotation has a /Contents entry and its directly enclosing structure element has an /Alt entry, the two are identical. Divergent values make assistive technology announce two conflicting descriptions for the same object. The check fires only when both texts are present."
}

func (c AnnotAltContents) Run(doc model.Document) []engine.Finding {
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
		if a.Contents == "" || a.StructAlt == "" {
			continue
		}
		seen++
		if a.Contents == a.StructAlt {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("annotation on page %d has /Contents %q but its enclosing element's /Alt is %q; they must be identical", a.Page, a.Contents, a.StructAlt),
			Hint:     "Make the annotation's /Contents and the enclosing structure element's /Alt identical, or drop one of them.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no annotation has both a /Contents entry and an enclosing /Alt -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(AnnotAltContents{}) }
