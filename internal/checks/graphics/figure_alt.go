// Package graphics groups checks that inspect graphical structure
// elements (Figure, Formula) and their accessible text equivalents.
package graphics

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FigureAlt fails for every Figure structure element that carries
// neither /Alt nor /ActualText. PDF/UA-1 §7.3 requires figures to
// provide an alternate description; without it screen readers have no
// announceable text for the content.
//
// One finding is emitted per offending Figure, so a single document
// with three undecorated figures produces three findings under the
// same check ID.
type FigureAlt struct{}

func (FigureAlt) ID() string    { return "MH-13-004" }
func (FigureAlt) Title() string { return "Figure has Alt or ActualText" }
func (FigureAlt) Description() string {
	return "Every Figure structure element must provide an accessible text equivalent so screen readers have something to announce. PDF/UA-1 §7.3 accepts /Alt (a description of the image) or /ActualText (a textual replacement, used for decorative or rebus-style figures)."
}
func (FigureAlt) Category() engine.Category { return engine.CategoryGraphics }
func (FigureAlt) Severity() engine.Severity { return engine.SeverityError }
func (FigureAlt) Spec() engine.Spec         { return engine.SpecBoth }
func (FigureAlt) WCAG() []string            { return []string{"1.1.1"} }

func (c FigureAlt) Run(doc model.Document) []engine.Finding {
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		// No structure tree means there are no Figure StructElems to
		// inspect. MH-01-005 already flags the missing tree; declare
		// ourselves not-applicable so the verdict honestly says N/A
		// instead of a misleading PASS.
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	figureCount := 0
	c.walk(root, "/"+root.Type(), &findings, &figureCount)
	if figureCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Figure structure elements -- nothing to inspect",
		}}
	}
	return findings
}

// walk descends the structure tree and records one finding per Figure
// without an accessible text equivalent. /Alt and /ActualText are not
// symmetric: /Alt requires a non-empty value (an empty Alt conveys no
// description), while /ActualText is satisfied by mere presence -- a
// present-but-empty /ActualText is the documented "render as silence"
// idiom for decorative figures, mirroring veraPDF UA-1 §7.7 pass-c
// for Formula.
func (c FigureAlt) walk(elem model.StructElement, path string, out *[]engine.Finding, count *int) {
	if elem.Type() == "Figure" {
		*count++
		if elem.Attr("Alt") == "" && !elem.AttrPresent("ActualText") {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "Figure has neither /Alt nor /ActualText",
				Hint:     "Add /Alt with a textual description (or /ActualText for decorative replacements) to the Figure structure element.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

func init() { engine.Register(FigureAlt{}) }
