// Package tables groups checks that inspect Table structure elements.
package tables

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Structure fails for every Table StructElem that has no TR descendants
// in its direct children, even allowing the optional THead/TBody/TFoot
// wrappers. PDF/UA-1 §7.5 requires tables to be expressed via the
// canonical Table → (THead|TBody|TFoot)? → TR → (TH|TD) pattern; a
// Table whose children are paragraphs or other non-row elements
// signals that the author tagged the visual layout rather than the
// semantic structure.
//
// Cell-level validation (TR must contain only TH/TD; TH must declare
// /Scope on header cells) is deferred to a separate check so each
// rule keeps a single responsibility.
type Structure struct{}

func (Structure) ID() string    { return "MH-15-003" }
func (Structure) Title() string { return "Table contains rows (TR)" }
func (Structure) Description() string {
	return "Tables must use the canonical Table -> (THead|TBody|TFoot)? -> TR -> (TH|TD) tagging pattern. This check fires when a Table structure element has no TR among its children -- usually a sign that the author tagged visual layout (paragraphs in a grid) rather than semantic table structure."
}
func (Structure) Category() engine.Category { return engine.CategoryTables }
func (Structure) Severity() engine.Severity { return engine.SeverityError }
func (Structure) Spec() engine.Spec         { return engine.SpecBoth }
func (Structure) WCAG() []string            { return []string{"1.3.1"} }

func (c Structure) Run(doc model.Document) []engine.Finding {
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
	c.walk(root, "/"+root.Type(), &findings)
	return findings
}

func (c Structure) walk(elem model.StructElement, path string, out *[]engine.Finding) {
	if elem.Type() == "Table" {
		if !hasRow(elem) {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "Table has no TR children (directly or via THead/TBody/TFoot)",
				Hint:     "Use the canonical Table → (THead|TBody|TFoot)? → TR → (TH|TD) tagging pattern.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out)
	}
}

// hasRow returns true if the table contains at least one TR child --
// either as a direct child of Table or one level deeper via THead,
// TBody or TFoot. Deeper nesting is not accepted (a TR wrapped in some
// other element would be non-canonical anyway).
func hasRow(table model.StructElement) bool {
	for _, child := range table.Children() {
		switch child.Type() {
		case "TR":
			return true
		case "THead", "TBody", "TFoot":
			for _, grand := range child.Children() {
				if grand.Type() == "TR" {
					return true
				}
			}
		}
	}
	return false
}

func init() { engine.Register(Structure{}) }
