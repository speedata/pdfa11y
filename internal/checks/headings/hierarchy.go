// Package headings groups checks that inspect the heading structure
// (H, H1, H2, …) of a tagged document.
package headings

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Hierarchy enforces two PDF/UA-1 §7.4.2 rules about numbered headings:
//
//  1. The first heading in the document must be H1 -- otherwise the
//     outline implicitly starts inside an absent parent section.
//  2. From there, levels must not jump downward by more than one
//     (H1 -> H3 is bad; H3 -> H1 is fine, that starts a new top-level
//     section).
//
// The unnumbered /S = "H" type is intentionally ignored: it conveys
// "structured heading without level", which has no meaningful position
// in a numeric hierarchy. PDF 2.0 documents using only /H benefit from
// a different check that is not yet implemented.
type Hierarchy struct{}

func (Hierarchy) ID() string    { return "UA-14-003" }
func (Hierarchy) Title() string { return "Headings start at H1 and are properly nested" }
func (Hierarchy) Description() string {
	return "PDF/UA-1 §7.4.2 requires the first heading to be H1 and subsequent levels to descend by at most one. Starting with H3, or jumping H1 -> H3, leaves the document outline broken in ways assistive technology cannot reconstruct."
}
func (Hierarchy) Category() engine.Category { return engine.CategoryHeadings }
func (Hierarchy) Severity() engine.Severity { return engine.SeverityError }
func (Hierarchy) Spec() engine.Spec         { return engine.SpecBoth }
func (Hierarchy) WCAG() []string            { return []string{"1.3.1", "2.4.6"} }

type occurrence struct {
	level int
	path  string
	page  int
}

func (c Hierarchy) Run(doc model.Document) []engine.Finding {
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}

	var headings []occurrence
	collect(root, "/"+root.Type(), &headings)
	if len(headings) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no numbered heading elements -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	prev := 0
	for i, h := range headings {
		switch {
		case i == 0 && h.level != 1:
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("first heading is H%d, expected H1", h.level),
				Hint:     "Begin the document outline with an H1; deeper levels follow only after their parent heading exists.",
				Location: &engine.Location{Page: h.page, StructPath: h.path},
			})
		case prev > 0 && h.level-prev > 1:
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("heading H%d follows H%d -- level skipped", h.level, prev),
				Hint:     fmt.Sprintf("Use H%d (or shallower) instead of H%d, or insert intermediate heading(s).", prev+1, h.level),
				Location: &engine.Location{Page: h.page, StructPath: h.path},
			})
		}
		prev = h.level
	}
	return findings
}

func collect(elem model.StructElement, path string, out *[]occurrence) {
	if level := headingLevel(elem.Type()); level > 0 {
		*out = append(*out, occurrence{level: level, path: path, page: elem.Page()})
	}
	for _, child := range elem.Children() {
		collect(child, path+"/"+child.Type(), out)
	}
}

// headingLevel parses the trailing digits of an "H<n>" structure type.
// Returns 0 for non-heading types or the bare "H" (which has no level).
func headingLevel(typ string) int {
	if !strings.HasPrefix(typ, "H") || len(typ) < 2 {
		return 0
	}
	n, err := strconv.Atoi(typ[1:])
	if err != nil || n < 1 {
		return 0
	}
	return n
}

func init() { engine.Register(Hierarchy{}) }
