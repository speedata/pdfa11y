package lists

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Numbering warns for every L StructElem that does not declare a
// /ListNumbering attribute. Warning rather than Error: ISO 32000-1
// Table 348 defines a /ListNumbering default of "None", so an unset
// attribute is technically valid for unordered lists with neutral
// bullets. For *ordered* lists (Decimal, LowerRoman, …) the attribute
// is required so AT announces the list as ordered -- but we cannot
// tell ordered from unordered without inspecting the Lbl glyph
// content, which lives in the page content stream and would be a
// brittle string-pattern check ("1.", "i.", "a)" vs "•").
//
// Severity Warning keeps the diagnostic visible (so authors who
// forget /ListNumbering on an actually-ordered list see the hint)
// without producing false-positive FAIL verdicts on legitimately
// unordered lists where pdfa.org accepts the omission.
type Numbering struct{}

func (Numbering) ID() string                { return "MH-16-003" }
func (Numbering) Title() string             { return "Lists declare /ListNumbering" }
func (Numbering) Category() engine.Category { return engine.CategoryLists }
func (Numbering) Severity() engine.Severity { return engine.SeverityWarning }
func (Numbering) Spec() engine.Spec         { return engine.SpecBoth }
func (Numbering) WCAG() []string            { return []string{"1.3.1"} }
func (Numbering) Description() string {
	return "ISO 32000-1 §14.8.5.4 defines /ListNumbering on L structure elements with a default of None. Ordered lists (Decimal, LowerRoman, …) require the attribute so assistive technology announces the ordering; unordered lists with neutral bullets may legitimately omit it. pdfa11y cannot distinguish ordered from unordered without inspecting the Lbl glyph -- so a missing /ListNumbering is reported as a Warning rather than a hard Error."
}

func (c Numbering) Run(doc model.Document) []engine.Finding {
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
	var findings []engine.Finding
	listCount := 0
	c.walk(root, "/"+root.Type(), &findings, &listCount)
	if listCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no L (list) structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c Numbering) walk(elem model.StructElement, path string, out *[]engine.Finding, count *int) {
	if elem.Type() == "L" {
		*count++
		if elem.Attribute("ListNumbering") == "" {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityWarning,
				Message:  fmt.Sprintf("L at %s has no /ListNumbering attribute (default is None; ordered lists must declare Decimal / Roman / Alpha)", path),
				Hint:     "If this list is ordered, add /A << /O /List /ListNumbering /Decimal >> (or LowerRoman, …) to the L structure element. Unordered lists with neutral bullets may omit the attribute.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

func init() { engine.Register(Numbering{}) }
