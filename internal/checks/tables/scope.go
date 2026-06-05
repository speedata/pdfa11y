package tables

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Scope fails for every TH structure element that does not declare a
// /Scope attribute. PDF/UA-1 §7.5 (and ISO 32000-1 Table 384)
// requires header cells to spell out whether they apply to a Row, a
// Column or Both: assistive technology uses /Scope to associate each
// data cell with the correct header. Without it AT must guess from
// the geometric position, which fails for any layout beyond a
// single-row header.
//
// Valid values: Row, Column, Both.
//
// Severity Error: missing /Scope on a TH is a hard PDF/UA failure,
// distinct from MH-16-003 (ListNumbering) where the spec allows
// omission. Here the spec is unambiguous.
type Scope struct{}

func (Scope) ID() string    { return "MH-15-005" }
func (Scope) Title() string { return "Table header cells declare /Scope" }
func (Scope) Description() string {
	return "Per PDF/UA-1 §7.5 every TH structure element must carry a /Scope attribute (Row, Column or Both) so assistive technology can associate data cells with the correct header. Geometric guessing fails on layouts with column groups, row groups, or multi-level headers."
}
func (Scope) Category() engine.Category { return engine.CategoryTables }
func (Scope) Severity() engine.Severity { return engine.SeverityError }
func (Scope) Spec() engine.Spec         { return engine.SpecBoth }
func (Scope) WCAG() []string            { return []string{"1.3.1"} }

func (c Scope) Run(doc model.Document) []engine.Finding {
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
	thCount := 0
	c.walk(root, "/"+root.Type(), &findings, &thCount)
	if thCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no TH (table header) structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c Scope) walk(elem model.StructElement, path string, out *[]engine.Finding, count *int) {
	if elem.Type() == "TH" {
		*count++
		scope := elem.Attribute("Scope")
		switch scope {
		case "Row", "Column", "Both":
			// OK
		case "":
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "TH has no /Scope attribute",
				Hint:     "Add /A << /O /Table /Scope /Column >> (or /Row, /Both) to the TH structure element so AT knows whether the header applies to a row, a column or both.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		default:
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("TH /Scope is %q, expected Row, Column or Both", scope),
				Hint:     "ISO 32000-1 Table 384 defines /Scope values as Row, Column or Both. Update the attribute to one of those.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

func init() { engine.Register(Scope{}) }
