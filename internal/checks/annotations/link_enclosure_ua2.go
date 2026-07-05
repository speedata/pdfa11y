package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// LinkEnclosureUA2 fails when a Link annotation that is part of the structure
// tree is enclosed in a structure element other than Link or Reference. ISO
// 14289-2 §8.2.5.20: "A link annotation and its associated content shall be
// enclosed in either a Link or a Reference structure element." A link that is
// not in the structure tree at all, or that is an artifact, is out of scope
// here (those cases are governed by the "must be tagged" rules).
//
// One finding per offending Link annotation. N/A when the document has no Link
// annotations in the structure tree. PDF/UA-2 only; the PDF/UA-1 form is
// UA-28-013 (Link element only, no Reference).
type LinkEnclosureUA2 struct{}

func (LinkEnclosureUA2) ID() string { return "UA-28-016" }
func (LinkEnclosureUA2) Title() string {
	return "Link annotations are enclosed in a Link or Reference element (PDF/UA-2)"
}
func (LinkEnclosureUA2) Category() engine.Category { return engine.CategoryInteractive }
func (LinkEnclosureUA2) Severity() engine.Severity { return engine.SeverityError }
func (LinkEnclosureUA2) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (LinkEnclosureUA2) WCAG() []string            { return []string{"1.3.1"} }
func (LinkEnclosureUA2) Description() string {
	return "ISO 14289-2 §8.2.5.20 requires a link annotation and its associated content to be enclosed in a Link or Reference structure element. A link whose owning structure element is some other type is not announced correctly. A link that is not in the structure tree, or that is an artifact, is handled by the tagging rules and not flagged here."
}

func (c LinkEnclosureUA2) Run(doc model.Document) []engine.Finding {
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
		if a.Subtype != "Link" {
			continue
		}
		// Not in the structure tree (or an artifact) -> out of scope.
		if a.StructType == "" || a.StructType == "Artifact" {
			continue
		}
		seen++
		if a.StructType == "Link" || a.StructType == "Reference" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("Link annotation on page %d is enclosed in a %s element, expected Link or Reference", a.Page, a.StructType),
			Hint:     "Enclose the link annotation and its content in a Link (or Reference) structure element.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure-tree Link annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(LinkEnclosureUA2{}) }
