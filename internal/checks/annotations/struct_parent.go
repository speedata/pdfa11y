package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// StructLinked fails for every user-visible annotation that does
// not carry a /StructParent entry tying it back to the structure
// tree. PDF/UA-1 §7.18 requires every real-content annotation to be
// reachable from the structure tree so AT can place it in the reading
// order. Annotations that are purely decorative or page-furniture
// (PrinterMark, TrapNet, Watermark, Popup) are exempt -- they are
// page artifacts and should not appear in the reading order.
//
// Hidden / NoView annotations are also skipped: they are not
// user-visible and do not participate in AT surfaces.
type StructLinked struct{}

func (StructLinked) ID() string                { return "MH-28-004" }
func (StructLinked) Title() string             { return "Annotations are linked from the structure tree" }
func (StructLinked) Category() engine.Category { return engine.CategoryInteractive }
func (StructLinked) Severity() engine.Severity { return engine.SeverityError }
func (StructLinked) Spec() engine.Spec         { return engine.SpecBoth }
func (StructLinked) WCAG() []string            { return []string{"1.3.1"} }
func (StructLinked) Description() string {
	return "PDF/UA-1 §7.18 requires every real-content annotation to be reachable from the structure tree via its /StructParent. Without that link the annotation has no place in the reading order, so AT either announces it out of context or misses it entirely. Decorative annotations (PrinterMark, TrapNet, Watermark, Popup) are exempt."
}

// artifactSubtypes lists annotation subtypes that are inherently
// page furniture and should NOT appear in the structure tree. Popup
// is a companion to other annotations (Text/FreeText comments) and
// is not separately structured.
var artifactSubtypes = map[string]bool{
	"PrinterMark": true,
	"TrapNet":     true,
	"Watermark":   true,
	"Popup":       true,
}

func (c StructLinked) Run(doc model.Document) []engine.Finding {
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
		if artifactSubtypes[a.Subtype] {
			continue
		}
		if a.Hidden || a.NoView {
			continue
		}
		considered++
		if a.StructParent < 0 {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("%s annotation on page %d has no /StructParent", labelFor(a.Subtype), a.Page),
				Hint:     "Add /StructParent (an integer key into the ParentTree) on the annotation and reference the annotation from a structure element via an OBJR child. Decorative annotations should use one of the artifact subtypes (PrinterMark, TrapNet, Watermark) instead.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no structure-tree-eligible annotations -- nothing to inspect",
		}}
	}
	return findings
}

func labelFor(subtype string) string {
	if subtype == "" {
		return "untyped"
	}
	return subtype
}

func init() { engine.Register(StructLinked{}) }
