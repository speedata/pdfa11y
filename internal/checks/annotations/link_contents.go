// Package annotations groups checks that inspect page annotations
// (Link, Widget / form fields, Stamp, ...) and their accessibility
// surfaces.
package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// LinkContents flags every Link annotation that has no /Contents
// describing where it goes. /Contents is the canonical place for a
// link's purpose ("Project documentation", "Email the author");
// without it AT either reads the raw URI (often opaque) or skips the
// link.
//
// The severity depends on the conformance level:
//   - PDF/UA-1 (ISO 14289-1 §7.18) requires it ("shall") -- a missing
//     /Contents is an error.
//   - PDF/UA-2 (ISO 14289-2) relaxes it to a recommendation; producers
//     legitimately omit it when the surrounding text already conveys
//     the link's purpose, and a redundant /Contents often degrades the
//     reading experience. There it is surfaced as an advisory (Info),
//     which never affects the verdict, not even under --strict.
//
// Hidden (/F bit 2) and NoView annotations are not user-visible and
// therefore not required to carry /Contents.
type LinkContents struct{}

func (LinkContents) ID() string                { return "UA-28-001" }
func (LinkContents) Title() string             { return "Link annotations have /Contents" }
func (LinkContents) Category() engine.Category { return engine.CategoryInteractive }
func (LinkContents) Severity() engine.Severity { return engine.SeverityError }
func (LinkContents) Spec() engine.Spec         { return engine.SpecBoth }
func (LinkContents) WCAG() []string            { return []string{"2.4.4"} }
func (LinkContents) Description() string {
	return "Every interactive Link annotation should expose a /Contents value describing where the link goes; without it assistive technology either reads the raw URI or skips the link entirely. PDF/UA-1 §7.18 requires it (reported as an error). PDF/UA-2 relaxes this to a recommendation, so the absence is reported as an advisory there."
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
	// In PDF/UA-2 a missing /Contents is only a recommendation, so it
	// is reported as an advisory rather than an error. DetectPart reads
	// the XMP pdfuaid:part; on any error we fall back to the stricter
	// UA-1 treatment.
	severity := engine.SeverityError
	if part, found, err := pdfua.DetectPart(doc); err == nil && found && part == 2 {
		severity = engine.SeverityInfo
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
				Severity: severity,
				Message:  fmt.Sprintf("Link annotation on page %d has no /Contents", a.Page),
				Hint:     "Add /Contents with a short description of the link target (e.g. 'Project documentation') so AT can announce it. In PDF/UA-2 this is a recommendation, not a requirement.",
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
