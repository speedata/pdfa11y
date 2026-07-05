package tables

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Groups enforces the cardinality of a Table's row-group children
// (ISO 14289-1 §7.2, ISO 32000-1 §14.8.4.3.4). Per Table:
//
//   - at most one THead                                (veraPDF UA1:7.2-11)
//   - at most one TFoot                                 (UA1:7.2-12)
//   - a TFoot only alongside at least one TBody          (UA1:7.2-13)
//   - a THead only alongside at least one TBody          (UA1:7.2-14)
//
// A simple table that lists TR directly under Table (no THead/TBody/TFoot
// grouping) trivially satisfies all four rules. The type-nesting rules are
// UA-15-008; this check is only about how many of each group are present.
//
// One finding per violation. N/A when the document has no Table elements.
type Groups struct{}

func (Groups) ID() string                { return "UA-15-009" }
func (Groups) Title() string             { return "Table has at most one THead/TFoot, with a TBody when grouped" }
func (Groups) Category() engine.Category { return engine.CategoryTables }
func (Groups) Severity() engine.Severity { return engine.SeverityError }
func (Groups) Spec() engine.Spec         { return engine.SpecBoth }
func (Groups) WCAG() []string            { return []string{"1.3.1"} }
func (Groups) Description() string {
	return "A Table may contain at most one THead and at most one TFoot row group (ISO 14289-1 §7.2, ISO 32000-1 §14.8.4.3.4). Whenever a THead or a TFoot is present, the Table must also carry at least one TBody, so the header/footer rows are anchored to a body. Simple tables that place TR directly under Table are unaffected."
}

func (c Groups) Run(doc model.Document) []engine.Finding {
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
	tables := 0
	c.walk(root, "/"+root.Type(), &findings, &tables)
	if tables == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Table structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c Groups) walk(elem model.StructElement, path string, out *[]engine.Finding, tables *int) {
	children := elem.Children()
	if elem.Type() == "Table" {
		*tables++
		var head, body, foot int
		for _, ch := range children {
			switch ch.Type() {
			case "THead":
				head++
			case "TBody":
				body++
			case "TFoot":
				foot++
			}
		}
		if head > 1 {
			c.report(out, path, elem, "Table has more than one THead; only one is allowed",
				"Merge the header rows into a single THead group.")
		}
		if foot > 1 {
			c.report(out, path, elem, "Table has more than one TFoot; only one is allowed",
				"Merge the footer rows into a single TFoot group.")
		}
		if foot > 0 && body == 0 {
			c.report(out, path, elem, "Table has a TFoot but no TBody",
				"Group the table's body rows in a TBody alongside the TFoot.")
		}
		if head > 0 && body == 0 {
			c.report(out, path, elem, "Table has a THead but no TBody",
				"Group the table's body rows in a TBody alongside the THead.")
		}
	}
	for _, child := range children {
		c.walk(child, path+"/"+child.Type(), out, tables)
	}
}

func (c Groups) report(out *[]engine.Finding, path string, elem model.StructElement, msg, hint string) {
	*out = append(*out, engine.Finding{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  msg,
		Hint:     hint,
		Location: &engine.Location{Page: elem.Page(), StructPath: path},
	})
}

func init() { engine.Register(Groups{}) }
