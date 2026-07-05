package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// InvisibleArtifact fails when an annotation that is not user-visible is
// present in the structure tree without being an artifact. ISO 14289-2
// §8.9.2.2 (ISO 32000-2 Table 167) covers two flag conditions:
//
//   - the Invisible flag (/F bit 1) is set (veraPDF UA2:8.9.2.2-1), or
//   - the NoView flag (/F bit 5) is set while ToggleNoView (/F bit 9) is not
//     (veraPDF UA2:8.9.2.2-2).
//
// An annotation the reader cannot see conveys no perceivable content, so
// including it in logical structure gives assistive technology a phantom node.
// Such an annotation must be an artifact instead. An annotation that is not in
// the structure tree, or already enclosed as an Artifact, is compliant.
//
// One finding per offending annotation. N/A when the document has no invisible
// or no-view annotations. PDF/UA-2 only.
type InvisibleArtifact struct{}

func (InvisibleArtifact) ID() string { return "UA-28-020" }
func (InvisibleArtifact) Title() string {
	return "Invisible and no-view annotations are artifacts, not tagged content"
}
func (InvisibleArtifact) Category() engine.Category { return engine.CategoryInteractive }
func (InvisibleArtifact) Severity() engine.Severity { return engine.SeverityError }
func (InvisibleArtifact) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (InvisibleArtifact) WCAG() []string            { return []string{"1.3.1"} }
func (InvisibleArtifact) Description() string {
	return "ISO 14289-2 §8.9.2.2 requires an annotation to be an artifact when its Invisible flag is set, or when its NoView flag is set without ToggleNoView. Such an annotation is not visible to the reader and conveys no content; enclosing it in logical structure gives assistive technology a phantom node to announce."
}

func (c InvisibleArtifact) Run(doc model.Document) []engine.Finding {
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
		invisible := a.Invisible
		noView := a.NoView && !a.ToggleNoView
		if !invisible && !noView {
			continue
		}
		seen++
		// Not in the structure tree, or already an artifact, is compliant.
		if a.StructType == "" || a.StructType == "Artifact" {
			continue
		}
		reason := "invisible"
		if !invisible {
			reason = "no-view"
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("%s annotation on page %d is enclosed in a %s structure element instead of being an artifact", reason, a.Page, a.StructType),
			Hint:     "Mark the annotation as an artifact (keep it out of logical structure); it is not visible to the reader.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no invisible or no-view annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(InvisibleArtifact{}) }
