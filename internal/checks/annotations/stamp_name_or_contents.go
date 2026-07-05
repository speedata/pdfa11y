package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// StampNameOrContents fails when a rubber-stamp annotation carries neither a
// /Name nor a /Contents entry. ISO 14289-2 §8.9.2.4.7 (veraPDF UA2:8.9.2.4.7-1,
// ISO 32000-2 §12.5.6.2): the intent of a stamp is conveyed by its /Name (the
// standard stamp icon, e.g. "Approved"); when the name is insufficient a
// /Contents entry describing the author's intent shall be provided. A stamp
// with neither offers assistive technology no way to describe it.
//
// One finding per offending Stamp annotation. N/A when the document has no
// Stamp annotations. PDF/UA-2 only.
type StampNameOrContents struct{}

func (StampNameOrContents) ID() string { return "UA-28-021" }
func (StampNameOrContents) Title() string {
	return "Rubber-stamp annotations carry a /Name or /Contents entry"
}
func (StampNameOrContents) Category() engine.Category { return engine.CategoryInteractive }
func (StampNameOrContents) Severity() engine.Severity { return engine.SeverityError }
func (StampNameOrContents) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (StampNameOrContents) WCAG() []string            { return []string{"1.1.1"} }
func (StampNameOrContents) Description() string {
	return "ISO 14289-2 §8.9.2.4.7 requires a rubber-stamp annotation to describe its intent. The intent is normally the /Name (a standard stamp icon); when the name is insufficient, a /Contents entry describing the author's intent shall be provided. A stamp carrying neither gives assistive technology nothing to announce."
}

func (c StampNameOrContents) Run(doc model.Document) []engine.Finding {
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
		if a.Subtype != "Stamp" {
			continue
		}
		seen++
		if a.Name != "" || a.Contents != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("rubber-stamp annotation on page %d has neither a /Name nor a /Contents entry", a.Page),
			Hint:     "Give the stamp a standard /Name, or add a /Contents entry describing the author's intent.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no rubber-stamp annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(StampNameOrContents{}) }
