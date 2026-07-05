package lists

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Numbering inspects every L (list) structure element for a
// /ListNumbering attribute and reports a missing one as a Warning.
//
// ISO 14289-2 §8.2.5.25 reads "If Lbl structure elements are present,
// the ListNumbering attribute shall be present on the respective L
// structure element" -- a literal "shall". We nonetheless keep this a
// Warning rather than an Error, because the antecedent that makes it a
// violation ("is this an *ordered* list?") is not machine-decidable
// from the structure tree. The pdfa.org reference corpus proves the
// point: L_07 ("unordered lists with the ListNumbering correctly not
// set") carries Lbl children whose text is "✓ " and glossary terms
// like "ACCC: ", and L_09 carries "• " bullets -- all legitimately
// unordered, all correctly omitting /ListNumbering. The Lbl *text*
// cannot disambiguate them (a glossary "ACCC:" has letters yet is
// unordered; an ordered list's "1." may even decode to "" as in
// L_F03). PAC and veraPDF's PDF/UA-1 profile do not enforce this
// clause as a hard machine check for the same reason. Reporting it as
// a Warning keeps the diagnostic visible, escalates under --strict,
// and never produces a false-positive FAIL on a conforming unordered
// list. Genuine ordered-list violations (L_F03) still fail the verdict
// via the missing PDF/UA identifier (UA-06-003).
//
// Severity() returns SeverityError so the engine's registry presents
// the rule at its strictest level; the per-finding severity below is
// what reporters actually surface.
type Numbering struct{}

func (Numbering) ID() string                { return "UA-16-003" }
func (Numbering) Title() string             { return "Lists declare /ListNumbering" }
func (Numbering) Category() engine.Category { return engine.CategoryLists }
func (Numbering) Severity() engine.Severity { return engine.SeverityError }
func (Numbering) Spec() engine.Spec         { return engine.SpecBoth }
func (Numbering) WCAG() []string            { return []string{"1.3.1"} }
func (Numbering) Description() string {
	return "ISO 14289-2 §8.2.5.25 (and ISO 32000-1 §14.8.5.4) governs /ListNumbering on L structure elements. The default is None. Ordered lists (Decimal, LowerRoman, ...) must declare the attribute so AT announces the ordering; unordered lists with neutral bullets may legitimately omit it. ISO 14289-2 strengthens this when the list carries Lbl children: \"If Lbl structure elements are present, the ListNumbering attribute shall be present on the respective L structure element\" -- in that case the missing attribute is a hard Error; otherwise it remains a Warning."
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
			msg := fmt.Sprintf("L at %s has no /ListNumbering attribute (default is None; ordered lists must declare Decimal / Roman / Alpha)", path)
			hint := "If this list is ordered, add /A << /O /List /ListNumbering /Decimal >> (or LowerRoman, …) to the L structure element. Unordered lists with neutral bullets may omit the attribute."
			if hasLblChild(elem) {
				msg = fmt.Sprintf("L at %s carries Lbl children but has no /ListNumbering attribute (ISO 14289-2 §8.2.5.25 requires it when the list is ordered; unordered lists with bullet labels may omit it)", path)
				hint = "If this list is ordered, add /A << /O /List /ListNumbering /Decimal >> (or LowerRoman, LowerAlpha, ...) to the L structure element so AT can announce the ordering scheme. Unordered lists whose Lbl glyphs are neutral bullets (•, ✓, –) may leave /ListNumbering unset."
			}
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityWarning,
				Message:  msg,
				Hint:     hint,
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

// hasLblChild reports whether the L element has any LI descendant
// that itself carries an Lbl child. The Lbl can live directly under
// LI (the typical shape) or nested deeper inside the LI's subtree;
// either way the §8.2.5.25 requirement applies.
func hasLblChild(l model.StructElement) bool {
	for _, child := range l.Children() {
		if child.Type() != "LI" {
			continue
		}
		if subtreeHasType(child, "Lbl") {
			return true
		}
	}
	return false
}

func subtreeHasType(elem model.StructElement, want string) bool {
	if elem.Type() == want {
		return true
	}
	for _, child := range elem.Children() {
		if subtreeHasType(child, want) {
			return true
		}
	}
	return false
}

func init() { engine.Register(Numbering{}) }
