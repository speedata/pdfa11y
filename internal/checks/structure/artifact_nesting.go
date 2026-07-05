package structure

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ArtifactNesting fails when a page content stream nests an Artifact and
// structure-tagged content inside one another. ISO 14289-1 §7.1 (veraPDF
// UA1:7.1-1 and UA1:7.1-2) forbids this in both directions: an Artifact
// marked-content sequence must not sit inside tagged content, and tagged
// content must not sit inside an Artifact. Artifacts are content the structure
// tree deliberately excludes; mixing the two makes it ambiguous whether the
// enclosed content belongs to the reading order, so assistive technology
// cannot decide whether to present or skip it.
//
// One finding per illegal nesting. N/A when the document carries no page
// content to inspect.
type ArtifactNesting struct{}

func (ArtifactNesting) ID() string { return "UA-14-010" }
func (ArtifactNesting) Title() string {
	return "Artifact and tagged content are not nested inside one another"
}
func (ArtifactNesting) Category() engine.Category { return engine.CategoryStructure }
func (ArtifactNesting) Severity() engine.Severity { return engine.SeverityError }
func (ArtifactNesting) Spec() engine.Spec         { return engine.SpecBoth }
func (ArtifactNesting) WCAG() []string            { return []string{"1.3.1"} }
func (ArtifactNesting) Description() string {
	return "ISO 14289-1 §7.1 forbids nesting an Artifact and structure-tagged content inside one another, in either direction. An Artifact inside tagged content, or tagged content inside an Artifact, makes it ambiguous whether the enclosed content is part of the reading order: assistive technology cannot tell whether to present or skip it. Keep Artifact marked-content sequences and structure-tagged sequences as siblings, never nested."
}

func (c ArtifactNesting) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 || !anyContentActivity(pages) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document pages carry no marked content -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, p := range pages {
		for _, v := range p.MCNestingViolations {
			var msg string
			if v.ArtifactInsideTagged {
				msg = fmt.Sprintf("page %d nests an Artifact marked-content sequence inside tagged content", p.Number)
			} else {
				msg = fmt.Sprintf("page %d nests tagged content (%s) inside an Artifact marked-content sequence", p.Number, v.InnerTag)
			}
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  msg,
				Hint:     "Close the enclosing marked-content sequence before opening the other: Artifacts and structure-tagged content must be siblings, never nested inside one another.",
				Location: &engine.Location{Page: p.Number},
			})
		}
	}
	return findings
}

func init() { engine.Register(ArtifactNesting{}) }
