package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MarkupRCContents fails when a markup annotation carries both an /RC rich-text
// contents entry and a /Contents entry whose plain text differs. ISO 14289-2
// §8.9.2.3 (veraPDF UA2:8.9.2.3-2): when both /RC and /Contents are present for
// a markup annotation, they shall be textually equivalent, so assistive
// technology (reading /Contents) presents the same text a sighted user sees in
// the rich-text appearance.
//
// One finding per offending annotation. N/A when no markup annotation carries
// both entries. PDF/UA-2 only.
type MarkupRCContents struct{}

func (MarkupRCContents) ID() string                { return "UA-28-034" }
func (MarkupRCContents) Title() string             { return "Markup annotation /RC matches its /Contents" }
func (MarkupRCContents) Category() engine.Category { return engine.CategoryInteractive }
func (MarkupRCContents) Severity() engine.Severity { return engine.SeverityError }
func (MarkupRCContents) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (MarkupRCContents) WCAG() []string            { return []string{"1.3.1"} }
func (MarkupRCContents) Description() string {
	return "ISO 14289-2 §8.9.2.3 requires that when a markup annotation carries both an /RC rich-text contents entry and a /Contents entry, the two are textually equivalent. Assistive technology reads /Contents; if it disagrees with the displayed rich text, AT presents different text. The comparison strips the /RC markup and compares the plain text."
}

func (c MarkupRCContents) Run(doc model.Document) []engine.Finding {
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
		// The rule applies only when both entries are present on a markup
		// annotation (Contents == null passes).
		if !markupSubtypes[a.Subtype] || a.RichContents == "" || a.Contents == "" {
			continue
		}
		seen++
		if normalizeRichText(a.RichContents) != normalizeRichText(a.Contents) {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("%s annotation /RC and /Contents are not textually equivalent (/RC plain text %q vs /Contents %q)", a.Subtype, normalizeRichText(a.RichContents), normalizeRichText(a.Contents)),
				Hint:     "Make the /Contents text equal to the text content of the rich-text /RC entry.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no markup annotation carries both /RC and /Contents -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(MarkupRCContents{}) }
