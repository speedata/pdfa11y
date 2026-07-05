package tables

import (
	"fmt"
	"slices"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Nesting enforces the parent/child type constraints of the table model
// (ISO 14289-1 §7.2, ISO 32000-1 §14.8.4.3.4; the same shape is required by
// ISO 14289-2 §8.2.5.26). It reports, per structure element:
//
//   - a TR whose parent is not Table/THead/TBody/TFoot   (veraPDF UA1:7.2-4)
//   - a THead/TBody/TFoot whose parent is not Table       (UA1:7.2-5/-6/-7)
//   - a TH/TD whose parent is not TR                       (UA1:7.2-8/-9)
//   - a Table with a child other than TR/THead/TBody/TFoot/Caption (UA1:7.2-3)
//   - a THead/TBody/TFoot with a non-TR child           (UA1:7.2-36/-37/-38)
//
// Cell-level content (TR may contain only TH/TD) is UA-15-004; header /Scope
// is UA-15-005; grid geometry is UA-15-006. Type() resolves role mapping, so a
// custom type mapped to a standard table type is judged by its standard type.
//
// One finding per violation. N/A when the document has no table structure
// elements at all.
type Nesting struct{}

func (Nesting) ID() string                { return "UA-15-008" }
func (Nesting) Title() string             { return "Table sub-elements are correctly nested" }
func (Nesting) Category() engine.Category { return engine.CategoryTables }
func (Nesting) Severity() engine.Severity { return engine.SeverityError }
func (Nesting) Spec() engine.Spec         { return engine.SpecBoth }
func (Nesting) WCAG() []string            { return []string{"1.3.1"} }
func (Nesting) Description() string {
	return "The table structure types must be nested per ISO 14289-1 §7.2 (ISO 32000-1 §14.8.4.3.4): TR only inside Table/THead/TBody/TFoot; THead/TBody/TFoot only inside Table; TH/TD only inside TR; Table may contain only TR/THead/TBody/TFoot/Caption; and THead/TBody/TFoot may contain only TR. A Table may carry at most one Caption, placed as its first or last child. Misplaced table elements break the row/column model assistive technology reconstructs."
}

// tableElements is the set of standard table structure types; the check is
// N/A for documents that contain none of them.
var tableElements = map[string]bool{
	"Table": true, "THead": true, "TBody": true, "TFoot": true,
	"TR": true, "TH": true, "TD": true,
}

func (c Nesting) Run(doc model.Document) []engine.Finding {
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
	seen := 0
	c.walk(root, "", "/"+root.Type(), &findings, &seen)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no table structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c Nesting) walk(elem model.StructElement, parentType, path string, out *[]engine.Finding, seen *int) {
	t := elem.Type()
	if tableElements[t] {
		*seen++
	}

	// Child-perspective: does this element sit under a valid parent?
	switch t {
	case "TR":
		if !in(parentType, "Table", "THead", "TBody", "TFoot") {
			c.report(out, path, elem, fmt.Sprintf("TR is contained in %s, expected Table, THead, TBody or TFoot", named(parentType)),
				"Move the TR under a Table (or a THead/TBody/TFoot inside a Table).")
		}
	case "THead", "TBody", "TFoot":
		if parentType != "Table" {
			c.report(out, path, elem, fmt.Sprintf("%s is contained in %s, expected Table", t, named(parentType)),
				"Place the "+t+" group directly inside its Table element.")
		}
	case "TH", "TD":
		if parentType != "TR" {
			c.report(out, path, elem, fmt.Sprintf("%s is contained in %s, expected TR", t, named(parentType)),
				"Wrap the cell in a TR (table row).")
		}
	}

	children := elem.Children()

	// Parent-perspective: are this element's children of the allowed types?
	switch t {
	case "Table":
		for _, ct := range badChildren(children, "TR", "THead", "TBody", "TFoot", "Caption") {
			c.report(out, path, elem, fmt.Sprintf("Table has a %s child; only TR, THead, TBody, TFoot and Caption are allowed", ct),
				"Group rows under THead/TBody/TFoot (or place TR directly); move other content out of the Table element.")
		}
		c.checkCaption(children, out, path, elem)
	case "THead", "TBody", "TFoot":
		for _, ct := range badChildren(children, "TR") {
			c.report(out, path, elem, fmt.Sprintf("%s has a %s child; only TR is allowed", t, ct),
				"A "+t+" may contain table rows (TR) only.")
		}
	}

	for _, child := range children {
		c.walk(child, t, path+"/"+child.Type(), out, seen)
	}
}

// checkCaption enforces the two Table/Caption cardinality and placement rules
// (veraPDF UA1:7.2-39 and UA1:7.2-16): a Table may contain at most one Caption,
// and that Caption must be the first or the last child.
func (c Nesting) checkCaption(children []model.StructElement, out *[]engine.Finding, path string, elem model.StructElement) {
	captions := 0
	for _, ch := range children {
		if ch.Type() == "Caption" {
			captions++
		}
	}
	if captions == 0 {
		return
	}
	if captions > 1 {
		c.report(out, path, elem, fmt.Sprintf("Table has %d Caption children; only one is allowed", captions),
			"Keep a single Caption element on the Table.")
	}
	// Placement: with more than one Caption the "first or last" test is
	// ambiguous, so only evaluate it for the single-Caption case.
	if captions == 1 && len(children) > 1 {
		first := children[0].Type() == "Caption"
		last := children[len(children)-1].Type() == "Caption"
		if !first && !last {
			c.report(out, path, elem, "Table Caption is neither the first nor the last child",
				"Move the Caption to the start or the end of the Table's children.")
		}
	}
}

func (c Nesting) report(out *[]engine.Finding, path string, elem model.StructElement, msg, hint string) {
	*out = append(*out, engine.Finding{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  msg,
		Hint:     hint,
		Location: &engine.Location{Page: elem.Page(), StructPath: path},
	})
}

// badChildren returns the deduplicated standard types among children that are
// not in allowed. Empty when every child is allowed.
func badChildren(children []model.StructElement, allowed ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, child := range children {
		t := child.Type()
		if in(t, allowed...) || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func in(s string, set ...string) bool {
	return slices.Contains(set, s)
}

// named renders a parent type for messages, mapping the empty string (a
// top-level element with no structural parent) to a readable placeholder.
func named(t string) string {
	if t == "" {
		return "no structural parent"
	}
	return t
}

func init() { engine.Register(Nesting{}) }
