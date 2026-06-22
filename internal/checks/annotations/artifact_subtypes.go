package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ArtifactSubtypes fails when an annotation whose /Subtype is
// inherently page furniture (PrinterMark, TrapNet) nonetheless
// declares a /StructParent and is therefore referenced by the
// structure tree as real content. Both specs keep these subtypes
// outside the reading order: PrinterMark "shall be an artifact"
// (UA-1 §7.18.8 / UA-2 §8.9.2.4.14) and TrapNet is forbidden
// (UA-1 §7.18.2 / UA-2 §8.9.2.4.15) -- a screen reader should narrate
// neither.
//
// Watermark is deliberately NOT included: ISO 14289-2 §8.9.2.4.16
// allows a Watermark "used as real content", which then must be in
// the structure tree -- so a Watermark with /StructParent is
// conformant, not a defect.
//
// Popup is also page-furniture (it is paired with a parent
// annotation that owns the real content) but is intentionally
// omitted from this check: producers regularly chain Popup back to
// the structure tree via the same /StructParent as the parent text
// annotation, and flagging that as a violation would create false
// positives. The UA-28-004 check already exempts Popup from the
// "must be linked" rule.
type ArtifactSubtypes struct{}

func (ArtifactSubtypes) ID() string                { return "UA-28-006" }
func (ArtifactSubtypes) Title() string             { return "Artifact-only annotation subtypes are not in the structure tree" }
func (ArtifactSubtypes) Category() engine.Category { return engine.CategoryInteractive }
func (ArtifactSubtypes) Severity() engine.Severity { return engine.SeverityError }
func (ArtifactSubtypes) Spec() engine.Spec         { return engine.SpecBoth }
func (ArtifactSubtypes) WCAG() []string            { return []string{"1.3.1"} }
func (ArtifactSubtypes) Description() string {
	return "PrinterMark (UA-1 §7.18.8 / UA-2 §8.9.2.4.14) and TrapNet (UA-1 §7.18.2 / UA-2 §8.9.2.4.15) annotations are page furniture: they are not real content and must not be reachable from the structure tree. A /StructParent on one of these subtypes pulls printer marks or trapping data into the reading order, where screen readers narrate them out of context. (Watermark is excluded: PDF/UA-2 allows it as real content.)"
}

// pureArtifactSubtypes enumerates the annotation subtypes that must
// stay outside the structure tree. Popup is excluded by design (see
// the type-level comment).
var pureArtifactSubtypes = map[string]bool{
	"PrinterMark": true,
	"TrapNet":     true,
}

func (c ArtifactSubtypes) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	considered := 0
	for _, a := range annots {
		if !pureArtifactSubtypes[a.Subtype] {
			continue
		}
		considered++
		if a.StructParent >= 0 {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("%s annotation on page %d declares /StructParent %d -- artifact subtypes must not be in the structure tree", a.Subtype, a.Page, a.StructParent),
				Hint:     "Remove /StructParent from the annotation, and remove any OBJR pointing at it from the structure tree. Printer marks, trapping data and watermarks belong to the page furniture, not the document content.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no PrinterMark / TrapNet annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(ArtifactSubtypes{}) }
