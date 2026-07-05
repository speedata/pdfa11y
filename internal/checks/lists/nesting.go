package lists

import (
	"fmt"
	"slices"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Nesting enforces the parent/child type constraints of the list model
// (ISO 14289-1 §7.2, ISO 32000-1 §14.8.4.3.3; the same shape is required by
// ISO 14289-2 §8.2.5.25). It reports, per structure element:
//
//   - an LI whose parent is not L                    (veraPDF UA1:7.2-17)
//   - an LBody whose parent is not LI                 (UA1:7.2-18)
//   - an L with a child other than L/LI/Caption       (UA1:7.2-19)
//   - an LI with a child other than Lbl/LBody          (UA1:7.2-20)
//   - a Caption that is not the L's first child        (UA1:7.2-40)
//
// The "L has at least one LI" requirement is UA-16-001; per-item body
// placement is UA-16-002. Type() resolves role mapping, so a custom type
// mapped to a standard list type is judged by its standard type.
//
// One finding per violation. N/A when the document has no list structure
// elements at all.
type Nesting struct{}

func (Nesting) ID() string                { return "UA-16-004" }
func (Nesting) Title() string             { return "List sub-elements are correctly nested" }
func (Nesting) Category() engine.Category { return engine.CategoryLists }
func (Nesting) Severity() engine.Severity { return engine.SeverityError }
func (Nesting) Spec() engine.Spec         { return engine.SpecBoth }
func (Nesting) WCAG() []string            { return []string{"1.3.1"} }
func (Nesting) Description() string {
	return "The list structure types must be nested per ISO 14289-1 §7.2 (ISO 32000-1 §14.8.4.3.3): LI only inside L; LBody only inside LI; L may contain only L/LI/Caption; and LI may contain only Lbl/LBody. A Caption on an L, when present, must be its first child. Misplaced list elements stop assistive technology from announcing the enumeration structure."
}

// listElements is the set of standard list structure types; the check is N/A
// for documents that contain none of them.
var listElements = map[string]bool{
	"L": true, "LI": true, "Lbl": true, "LBody": true,
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
			Message:  "document contains no list structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c Nesting) walk(elem model.StructElement, parentType, path string, out *[]engine.Finding, seen *int) {
	t := elem.Type()
	if listElements[t] {
		*seen++
	}

	// Child-perspective: does this element sit under a valid parent?
	switch t {
	case "LI":
		if parentType != "L" {
			c.report(out, path, elem, fmt.Sprintf("LI is contained in %s, expected L", named(parentType)),
				"Place the LI directly inside an L (list) element.")
		}
	case "LBody":
		if parentType != "LI" {
			c.report(out, path, elem, fmt.Sprintf("LBody is contained in %s, expected LI", named(parentType)),
				"Wrap the list-item body in an LI element.")
		}
	}

	children := elem.Children()

	// Parent-perspective: are this element's children of the allowed types?
	switch t {
	case "L":
		for _, ct := range badChildren(children, "L", "LI", "Caption") {
			c.report(out, path, elem, fmt.Sprintf("L has a %s child; only L, LI and Caption are allowed", ct),
				"Wrap each list entry in an LI (a nested list may appear directly).")
		}
		c.checkCaption(children, out, path, elem)
	case "LI":
		for _, ct := range badChildren(children, "Lbl", "LBody") {
			c.report(out, path, elem, fmt.Sprintf("LI has a %s child; only Lbl and LBody are allowed", ct),
				"Put the item label in Lbl and the item content in LBody.")
		}
	}

	for _, child := range children {
		c.walk(child, t, path+"/"+child.Type(), out, seen)
	}
}

// checkCaption enforces UA1:7.2-40: a Caption on an L may appear only as the
// list's first child. Any Caption at a later position -- including a second
// Caption -- is a violation.
func (c Nesting) checkCaption(children []model.StructElement, out *[]engine.Finding, path string, elem model.StructElement) {
	for i, ch := range children {
		if ch.Type() == "Caption" && i != 0 {
			c.report(out, path, elem, "L Caption is not the first child",
				"Move the Caption to the start of the list, and keep only one Caption.")
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
		if slices.Contains(allowed, t) || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
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
