// Package annotations groups checks that inspect page annotations
// (Link, Widget / form fields, Stamp, ...) and their accessibility
// surfaces.
package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// LinkContents fails for every Link annotation that has no /Contents
// describing where it goes. PDF/UA-1 §7.18 requires interactive
// elements to expose an accessible label so assistive technology can
// announce them. /Contents is the canonical place for a link's
// purpose ("Project documentation", "Email the author"); without it
// AT either reads the raw URI (often opaque) or skips the link.
//
// Hidden (/F bit 2) and NoView annotations are not user-visible and
// therefore not required to carry /Contents.
type LinkContents struct{}

func (LinkContents) ID() string                { return "MH-28-001" }
func (LinkContents) Title() string             { return "Link annotations have /Contents" }
func (LinkContents) Category() engine.Category { return engine.CategoryInteractive }
func (LinkContents) Severity() engine.Severity { return engine.SeverityError }
func (LinkContents) Spec() engine.Spec         { return engine.SpecBoth }
func (LinkContents) WCAG() []string            { return []string{"2.4.4"} }
func (LinkContents) Description() string {
	return "PDF/UA-1 §7.18 requires every interactive Link annotation to expose a /Contents value describing where the link goes. Without /Contents, assistive technology either reads the raw URI or skips the link entirely -- both leave the user without context."
}

func (c LinkContents) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	linkCount := 0
	for _, a := range annots {
		if a.Subtype != "Link" {
			continue
		}
		linkCount++
		if a.Hidden || a.NoView {
			continue
		}
		if a.Contents == "" {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("Link annotation on page %d has no /Contents", a.Page),
				Hint:     "Add /Contents with a short description of the link target (e.g. 'Project documentation') so AT can announce it.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if linkCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Link annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(LinkContents{}) }
