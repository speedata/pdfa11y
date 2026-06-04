// Package lists groups checks that inspect list structure elements
// (L, LI, Lbl, LBody).
package lists

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Structure fails for every L StructElem whose direct children are
// not all LI. PDF/UA-1 §7.6 requires lists to use the L → LI →
// (Lbl|LBody) tagging pattern; a list whose children are paragraphs
// signals that the author tagged visual indentation rather than the
// semantic structure.
//
// Item-level rules (LI should contain Lbl and/or LBody) are deferred
// to a separate check so each rule keeps a single responsibility.
type Structure struct{}

func (Structure) ID() string    { return "MH-16-001" }
func (Structure) Title() string { return "List contains list items (LI)" }
func (Structure) Description() string {
	return "Lists must use the canonical L -> LI -> (Lbl|LBody) tagging pattern. A list whose children are paragraphs (rather than LI) is unannouncable as a list to assistive technology -- the user hears flat content instead of an enumeration."
}
func (Structure) Category() engine.Category { return engine.CategoryLists }
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
	if elem.Type() == "L" {
		if !hasItem(elem) {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "L has no LI children",
				Hint:     "Use the canonical L → LI → (Lbl|LBody) tagging pattern.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out)
	}
}

func hasItem(list model.StructElement) bool {
	for _, child := range list.Children() {
		if child.Type() == "LI" {
			return true
		}
	}
	return false
}

func init() { engine.Register(Structure{}) }
