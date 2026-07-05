package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// LinkEnclosure fails when a visible Link annotation is not enclosed in a Link
// structure element. ISO 14289-1 §7.18.5 (ISO 32000-1 §14.8.4.4.2) requires a
// link to be tagged with a Link structure element that owns both the link
// annotation (through an OBJR) and its associated content, so assistive
// technology announces the link as a link. veraPDF UA1:7.18.5-1 tolerates a
// link that is not presented -- hidden via /F or placed outside the crop box.
//
// One finding per offending Link annotation. N/A when the document has no
// visible Link annotations. PDF/UA-1 only (the PDF/UA-2 form, which also
// allows a Reference element, is tracked separately).
type LinkEnclosure struct{}

func (LinkEnclosure) ID() string { return "UA-28-013" }
func (LinkEnclosure) Title() string {
	return "Link annotations are enclosed in a Link structure element"
}
func (LinkEnclosure) Category() engine.Category { return engine.CategoryInteractive }
func (LinkEnclosure) Severity() engine.Severity { return engine.SeverityError }
func (LinkEnclosure) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (LinkEnclosure) WCAG() []string            { return []string{"1.3.1"} }
func (LinkEnclosure) Description() string {
	return "ISO 14289-1 §7.18.5 requires a link annotation to be tagged with a Link structure element (ISO 32000-1 §14.8.4.4.2): the Link element owns the annotation via an OBJR and encloses the associated content. A link whose owning structure element is not a Link (or that is not in the structure tree at all) is not announced as a link. Hidden or off-page links are exempt because they are not presented."
}

func (c LinkEnclosure) Run(doc model.Document) []engine.Finding {
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
		// Hidden or off-page links are not presented -> exempt.
		if a.Hidden || a.OffPage {
			continue
		}
		seen++
		if a.StructType == "Link" {
			continue
		}
		where := "is not in the structure tree"
		if a.StructType != "" {
			where = "is enclosed in a " + a.StructType + " element"
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("Link annotation on page %d %s, not a Link structure element", a.Page, where),
			Hint:     "Tag the link with a Link structure element that references the annotation via an OBJR and encloses the link's content.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no visible Link annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(LinkEnclosure{}) }
