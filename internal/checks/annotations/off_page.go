package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// OffPage fails for every annotation whose /Rect lies entirely
// outside the page's visible box (CropBox if present, MediaBox
// otherwise) and that does not have either the Hidden or NoView /F
// flag set. Such annotations are not perceivable to sighted readers
// but assistive technology still sees them in the reading order,
// creating phantom announcements that the user cannot orient
// against any visible content. PDF/UA-1 §7.18 requires off-page
// annotations to be marked Hidden or NoView so AT skips them too.
//
// The walker computes OffPage by comparing the annotation rectangle
// against the page box and ignores annotations where either box
// could not be parsed (those are reported as "on page" to avoid
// false positives on documents we cannot measure).
type OffPage struct{}

func (OffPage) ID() string                { return "MH-28-008" }
func (OffPage) Title() string             { return "Off-page annotations are hidden from AT" }
func (OffPage) Category() engine.Category { return engine.CategoryInteractive }
func (OffPage) Severity() engine.Severity { return engine.SeverityError }
func (OffPage) Spec() engine.Spec         { return engine.SpecBoth }
func (OffPage) WCAG() []string            { return []string{"1.3.1"} }
func (OffPage) Description() string {
	return "PDF/UA-1 §7.18 requires annotations whose rectangle falls outside the page's visible box to set the Hidden or NoView /F flag, so assistive technology also skips them. Off-page annotations that AT can still see appear as phantom entries in the reading order, with no visual content for the user to orient against."
}

func (c OffPage) Run(doc model.Document) []engine.Finding {
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
		if !a.OffPage {
			continue
		}
		considered++
		if a.Hidden || a.NoView {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("%s annotation on page %d sits outside the page box but is neither Hidden nor NoView", labelFor(a.Subtype), a.Page),
			Hint:     "Either move the annotation inside the page's CropBox/MediaBox, or set /F to include the Hidden (bit 2) or NoView (bit 5) flag so AT also skips it.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no off-page annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(OffPage{}) }
