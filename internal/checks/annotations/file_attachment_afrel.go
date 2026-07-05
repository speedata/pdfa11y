package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FileAttachmentAFRel fails when a file attachment annotation references a file
// specification dictionary that omits its /AFRelationship entry. ISO 14289-2
// §8.9.2.4.10 (veraPDF UA2:8.9.2.4.10-1, ISO 32000-2 §7.11.3 / §12.5.6.15):
// /AFRelationship declares how the attached file relates to the document
// content (Source, Data, Supplement, …), which assistive technology and
// downstream tools rely on to present the attachment meaningfully.
//
// One finding per offending file attachment annotation. N/A when the document
// has no file attachment annotations referencing a filespec. PDF/UA-2 only.
type FileAttachmentAFRel struct{}

func (FileAttachmentAFRel) ID() string { return "UA-28-024" }
func (FileAttachmentAFRel) Title() string {
	return "File attachment filespecs declare an /AFRelationship"
}
func (FileAttachmentAFRel) Category() engine.Category { return engine.CategoryInteractive }
func (FileAttachmentAFRel) Severity() engine.Severity { return engine.SeverityError }
func (FileAttachmentAFRel) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (FileAttachmentAFRel) WCAG() []string            { return []string{"1.3.1"} }
func (FileAttachmentAFRel) Description() string {
	return "ISO 14289-2 §8.9.2.4.10 requires the file specification dictionary referenced by a file attachment annotation to include an /AFRelationship entry. It declares the relationship of the attached file to the document (Source, Data, Supplement, …); without it, tools cannot present the attachment's role."
}

func (c FileAttachmentAFRel) Run(doc model.Document) []engine.Finding {
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
		if a.Subtype != "FileAttachment" || !a.HasFileSpec {
			continue
		}
		seen++
		if a.FileSpecHasAFRelationship {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("file attachment annotation on page %d references a filespec without an /AFRelationship entry", a.Page),
			Hint:     "Add /AFRelationship to the file specification dictionary (e.g. /Data or /Supplement) describing the attachment's relationship to the document.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no file attachment annotations referencing a filespec -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(FileAttachmentAFRel{}) }
