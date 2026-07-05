package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// PopupNotTagged fails when a Popup annotation is present in the structure
// tree. ISO 14289-2 §8.9.2.4.9 (veraPDF UA2:8.9.2.4.9-1, ISO 32000-2 §14.11.3):
// a Popup annotation is subordinate display furniture for its parent markup
// annotation -- it carries no independent content and shall not appear in
// logical structure. A Popup that is enclosed in a structure element gives
// assistive technology a spurious, contentless node to announce.
//
// One finding per offending Popup. The annotation's owning structure type is
// resolved via the /StructParent -> ParentTree lookup (Annotation.StructType);
// an empty type (not in the tree) or the Artifact type is compliant. N/A when
// the document has no Popup annotations. PDF/UA-2 only.
type PopupNotTagged struct{}

func (PopupNotTagged) ID() string { return "UA-28-017" }
func (PopupNotTagged) Title() string {
	return "Popup annotations are not present in the structure tree"
}
func (PopupNotTagged) Category() engine.Category { return engine.CategoryInteractive }
func (PopupNotTagged) Severity() engine.Severity { return engine.SeverityError }
func (PopupNotTagged) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (PopupNotTagged) WCAG() []string            { return []string{"1.3.1"} }
func (PopupNotTagged) Description() string {
	return "ISO 14289-2 §8.9.2.4.9 requires that Popup annotations not be present in the structure tree. A Popup is display furniture for its parent markup annotation and carries no independent content; enclosing it in a structure element adds a spurious, contentless node for assistive technology to announce."
}

func (c PopupNotTagged) Run(doc model.Document) []engine.Finding {
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
		if a.Subtype != "Popup" {
			continue
		}
		seen++
		// Not in the structure tree (or marked as an artifact) is compliant.
		if a.StructType == "" || a.StructType == "Artifact" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("Popup annotation on page %d is enclosed in a %s structure element", a.Page, a.StructType),
			Hint:     "Remove the Popup annotation from logical structure: it must not be tagged (it is display furniture for its parent markup annotation).",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no Popup annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(PopupNotTagged{}) }
