package lists

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ItemBody fails for every LI whose direct children do not include an
// LBody. PDF/UA-1 §7.6 spells out the canonical L → LI → (Lbl, LBody)
// pattern: Lbl carries the bullet or number, LBody carries the item
// text. Without LBody, screen readers announce only the marker glyph
// and the body content is loose -- either misattributed to the
// preceding LI or skipped entirely.
//
// Lbl is allowed but optional (unnumbered lists carry no marker);
// LBody is the mandatory container.
type ItemBody struct{}

func (ItemBody) ID() string                { return "UA-16-002" }
func (ItemBody) Title() string             { return "List items contain LBody" }
func (ItemBody) Category() engine.Category { return engine.CategoryLists }
func (ItemBody) Severity() engine.Severity { return engine.SeverityError }
func (ItemBody) Spec() engine.Spec         { return engine.SpecBoth }
func (ItemBody) WCAG() []string            { return []string{"1.3.1"} }
func (ItemBody) Description() string {
	return "PDF/UA-1 §7.6 lists must follow the L → LI → (Lbl, LBody) pattern. The Lbl child is optional (unnumbered lists have no marker), but LBody is mandatory: it holds the item's actual content. Without LBody, assistive technology reads only the bullet or number and the body text drifts out of the item's scope."
}

func (c ItemBody) Run(doc model.Document) []engine.Finding {
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
	itemCount := 0
	c.walk(root, "/"+root.Type(), &findings, &itemCount)
	if itemCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no LI (list item) structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c ItemBody) walk(elem model.StructElement, path string, out *[]engine.Finding, count *int) {
	if elem.Type() == "LI" {
		*count++
		if !hasLBody(elem) {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("LI at %s has no LBody child", path),
				Hint:     "Wrap the item's content in an LBody element; keep any bullet or number in a sibling Lbl.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

func hasLBody(li model.StructElement) bool {
	for _, child := range li.Children() {
		if child.Type() == "LBody" {
			return true
		}
	}
	return false
}

func init() { engine.Register(ItemBody{}) }
