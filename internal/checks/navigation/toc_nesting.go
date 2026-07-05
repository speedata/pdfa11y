package navigation

import (
	"fmt"
	"slices"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// TOCNesting enforces the parent/child type constraints of the table-of-
// contents model (ISO 14289-1 §7.2, ISO 32000-1 §14.8.4.3.5; the same shape is
// required by ISO 14289-2 §8.2.5.8). It reports, per structure element:
//
//   - a TOCI whose parent is not TOC                 (veraPDF UA1:7.2-26)
//   - a TOC with a child other than TOC/TOCI/Caption   (UA1:7.2-27)
//   - a Caption that is not the TOC's first child       (UA1:7.2-28)
//
// TOCI /Ref targeting is UA-01-012. Type() resolves role mapping, so a custom
// type mapped to a standard TOC type is judged by its standard type.
//
// One finding per violation. N/A when the document has no TOC/TOCI elements.
type TOCNesting struct{}

func (TOCNesting) ID() string                { return "UA-27-002" }
func (TOCNesting) Title() string             { return "Table-of-contents sub-elements are correctly nested" }
func (TOCNesting) Category() engine.Category { return engine.CategoryNavigation }
func (TOCNesting) Severity() engine.Severity { return engine.SeverityError }
func (TOCNesting) Spec() engine.Spec         { return engine.SpecBoth }
func (TOCNesting) WCAG() []string            { return []string{"1.3.1"} }
func (TOCNesting) Description() string {
	return "The table-of-contents structure types must be nested per ISO 14289-1 §7.2 (ISO 32000-1 §14.8.4.3.5): TOCI only inside TOC; TOC may contain only TOC/TOCI/Caption; and a Caption on a TOC, when present, must be its first child. Misplaced TOC elements break the navigable outline assistive technology reconstructs."
}

// tocElements is the set of standard TOC structure types; the check is N/A for
// documents that contain none of them.
var tocElements = map[string]bool{"TOC": true, "TOCI": true}

func (c TOCNesting) Run(doc model.Document) []engine.Finding {
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
			Message:  "document contains no TOC structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c TOCNesting) walk(elem model.StructElement, parentType, path string, out *[]engine.Finding, seen *int) {
	t := elem.Type()
	if tocElements[t] {
		*seen++
	}

	// Child-perspective: does this element sit under a valid parent?
	if t == "TOCI" && parentType != "TOC" {
		c.report(out, path, elem, fmt.Sprintf("TOCI is contained in %s, expected TOC", tocNamed(parentType)),
			"Place the TOCI directly inside a TOC element.")
	}

	children := elem.Children()

	// Parent-perspective: are this element's children of the allowed types?
	if t == "TOC" {
		for _, ct := range tocBadChildren(children, "TOC", "TOCI", "Caption") {
			c.report(out, path, elem, fmt.Sprintf("TOC has a %s child; only TOC, TOCI and Caption are allowed", ct),
				"Wrap each entry in a TOCI (a nested TOC may appear directly).")
		}
		// UA1:7.2-28: a Caption may appear only as the first child. Any
		// Caption at a later position -- including a second Caption -- fails.
		for i, ch := range children {
			if ch.Type() == "Caption" && i != 0 {
				c.report(out, path, elem, "TOC Caption is not the first child",
					"Move the Caption to the start of the TOC, and keep only one Caption.")
			}
		}
	}

	for _, child := range children {
		c.walk(child, t, path+"/"+child.Type(), out, seen)
	}
}

func (c TOCNesting) report(out *[]engine.Finding, path string, elem model.StructElement, msg, hint string) {
	*out = append(*out, engine.Finding{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  msg,
		Hint:     hint,
		Location: &engine.Location{Page: elem.Page(), StructPath: path},
	})
}

// tocBadChildren returns the deduplicated standard types among children that
// are not in allowed. Empty when every child is allowed.
func tocBadChildren(children []model.StructElement, allowed ...string) []string {
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

// tocNamed renders a parent type for messages, mapping the empty string (a
// top-level element with no structural parent) to a readable placeholder.
func tocNamed(t string) string {
	if t == "" {
		return "no structural parent"
	}
	return t
}

func init() { engine.Register(TOCNesting{}) }
