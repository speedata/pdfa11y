package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MarkupEnclosure fails when a markup annotation that is part of the structure
// tree is enclosed in a structure element other than Annot. ISO 14289-2
// §8.9.2.3: "Markup annotations shall be enclosed within Annot structure
// elements." A markup annotation that is not in the structure tree, or that is
// an artifact, is out of scope here.
//
// The markup annotation subtypes are those of ISO 32000-2 Table 171 (Text,
// FreeText, Line, Square, Circle, Polygon, PolyLine, Highlight, Underline,
// Squiggly, StrikeOut, Caret, Stamp, Ink, FileAttachment, Sound, Redact);
// Link, Widget and Popup are not markup.
//
// One finding per offending annotation. N/A when the document has no
// structure-tree markup annotations. PDF/UA-2 only.
type MarkupEnclosure struct{}

func (MarkupEnclosure) ID() string                { return "UA-28-015" }
func (MarkupEnclosure) Title() string             { return "Markup annotations are enclosed in an Annot element" }
func (MarkupEnclosure) Category() engine.Category { return engine.CategoryInteractive }
func (MarkupEnclosure) Severity() engine.Severity { return engine.SeverityError }
func (MarkupEnclosure) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (MarkupEnclosure) WCAG() []string            { return []string{"1.3.1"} }
func (MarkupEnclosure) Description() string {
	return "ISO 14289-2 §8.9.2.3 requires a markup annotation (ISO 32000-2 Table 171: Text, FreeText, Line, Square, Circle, Polygon, PolyLine, Highlight, Underline, Squiggly, StrikeOut, Caret, Stamp, Ink, FileAttachment, Sound, Redact) to be enclosed within an Annot structure element. A markup annotation whose owning structure element is some other type is not represented correctly; one that is not in the structure tree, or that is an artifact, is out of scope."
}

// markupSubtypes is the set of annotation subtypes classed as markup
// annotations by ISO 32000-2 Table 171.
var markupSubtypes = map[string]bool{
	"Text": true, "FreeText": true, "Line": true, "Square": true,
	"Circle": true, "Polygon": true, "PolyLine": true, "Highlight": true,
	"Underline": true, "Squiggly": true, "StrikeOut": true, "Caret": true,
	"Stamp": true, "Ink": true, "FileAttachment": true, "Sound": true,
	"Redact": true,
}

func (c MarkupEnclosure) Run(doc model.Document) []engine.Finding {
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
		if !markupSubtypes[a.Subtype] {
			continue
		}
		if a.StructType == "" || a.StructType == "Artifact" {
			continue
		}
		seen++
		if a.StructType == "Annot" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("%s markup annotation on page %d is enclosed in a %s element, expected Annot", a.Subtype, a.Page, a.StructType),
			Hint:     "Enclose the markup annotation in an Annot structure element.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure-tree markup annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(MarkupEnclosure{}) }
