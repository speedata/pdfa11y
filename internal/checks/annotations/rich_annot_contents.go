package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// RichAnnotContents fails when an interactive or rich-media annotation
// whose meaning is not otherwise exposed to assistive technology lacks a
// /Contents entry. PDF/UA-2 requires an alternate description in /Contents
// for these subtypes:
//
//   - Ink        §8.9.2.4.8  ("A Contents entry describing the author's
//                              intent shall be provided")
//   - Screen     §8.9.2.4.12 ("Screen annotations shall include a Contents
//                              entry")
//   - 3D         §8.9.2.4.19 ("3D annotation shall include alternate
//                              description in respective Contents entry")
//   - RichMedia  §8.9.2.4.19 (same, for RichMedia)
//
// Unlike text markup, these subtypes carry no intrinsic textual content a
// screen reader could fall back on -- an ink stroke, an embedded media
// player or a 3D scene is opaque without an author-supplied description.
//
// Annotations that are not presented to the user (Hidden or NoView) are
// skipped: they expose no accessibility surface, mirroring the treatment
// documented on model.Annotation. One finding per offending annotation.
type RichAnnotContents struct{}

func (RichAnnotContents) ID() string                { return "UA-28-011" }
func (RichAnnotContents) Title() string             { return "Ink, Screen, 3D and RichMedia annotations declare /Contents" }
func (RichAnnotContents) Category() engine.Category { return engine.CategoryInteractive }
func (RichAnnotContents) Severity() engine.Severity { return engine.SeverityError }
func (RichAnnotContents) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (RichAnnotContents) WCAG() []string            { return []string{"1.1.1"} }
func (RichAnnotContents) Description() string {
	return "PDF/UA-2 §8.9.2.4 requires Ink (§8.9.2.4.8), Screen (§8.9.2.4.12), 3D and RichMedia (§8.9.2.4.19) annotations to carry a /Contents entry describing the author's intent. These subtypes have no intrinsic text an assistive technology could read, so without /Contents their meaning is lost."
}

// contentsRequiredSubtypes enumerates the annotation subtypes that must
// carry a /Contents alternate description under PDF/UA-2 §8.9.2.4.
var contentsRequiredSubtypes = map[string]bool{
	"Ink":       true,
	"Screen":    true,
	"3D":        true,
	"RichMedia": true,
}

func (c RichAnnotContents) Run(doc model.Document) []engine.Finding {
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
		if !contentsRequiredSubtypes[a.Subtype] {
			continue
		}
		// Not presented to the user -> no accessibility surface to describe.
		if a.Hidden || a.NoView {
			continue
		}
		considered++
		if a.Contents == "" {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("%s annotation on page %d has no /Contents entry -- it carries no text for assistive technology to read", a.Subtype, a.Page),
				Hint:     "Add a /Contents string describing the annotation's purpose (the ink stroke's meaning, the media's content, the 3D scene). Screen readers surface /Contents for annotations that have no tagged text of their own.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Ink / Screen / 3D / RichMedia annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(RichAnnotContents{}) }
