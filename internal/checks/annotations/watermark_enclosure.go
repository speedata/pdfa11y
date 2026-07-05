package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// WatermarkEnclosure fails when a Watermark annotation used as real content is
// enclosed in a structure element other than Annot. ISO 14289-2 §8.9.2.4.16
// (veraPDF UA2:8.9.2.4.16-1): a Watermark that participates in logical
// structure (rather than being an artifact) conveys content and shall be
// wrapped in an Annot structure element so assistive technology announces it as
// an annotation.
//
// A Watermark that is not in the structure tree (empty owning type) or is
// enclosed as an Artifact is out of scope -- it is page furniture, governed by
// the artifact rules, not real content. One finding per offending Watermark.
// N/A when the document has no structure-tree Watermark annotations. PDF/UA-2
// only.
type WatermarkEnclosure struct{}

func (WatermarkEnclosure) ID() string { return "UA-28-019" }
func (WatermarkEnclosure) Title() string {
	return "Watermark annotations used as content are enclosed in an Annot element"
}
func (WatermarkEnclosure) Category() engine.Category { return engine.CategoryInteractive }
func (WatermarkEnclosure) Severity() engine.Severity { return engine.SeverityError }
func (WatermarkEnclosure) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (WatermarkEnclosure) WCAG() []string            { return []string{"1.3.1"} }
func (WatermarkEnclosure) Description() string {
	return "ISO 14289-2 §8.9.2.4.16 requires a Watermark annotation used as real content to be enclosed within an Annot structure element. A watermark tagged under some other element type is not announced as an annotation; a watermark that is an artifact (or not in the structure tree at all) is page furniture and out of scope here."
}

func (c WatermarkEnclosure) Run(doc model.Document) []engine.Finding {
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
		if a.Subtype != "Watermark" {
			continue
		}
		// Not in the structure tree, or an artifact -> page furniture, out of scope.
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
			Message:  fmt.Sprintf("Watermark annotation on page %d is enclosed in a %s element, expected Annot", a.Page, a.StructType),
			Hint:     "Enclose the watermark annotation in an Annot structure element, or mark it as an artifact if it is not content.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure-tree Watermark annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(WatermarkEnclosure{}) }
