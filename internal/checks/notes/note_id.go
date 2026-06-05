// Package notes groups checks that inspect the Note / Reference
// structure types and the cross-links between them (PDF/UA-1 §7.10).
package notes

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// NoteID enforces both halves of MH-19-001:
//
//   - Every Note structure element must carry an /ID so a reference
//     elsewhere in the document can target it. Without /ID assistive
//     technology has no anchor to surface "footnote / endnote" jumps.
//   - Every /Ref entry the walker can see must resolve to a structure
//     element that itself carries /ID. A /Ref targeting an element
//     without /ID is unreachable as a labelled anchor; a /Ref
//     resolving to nothing at all is dropped before we see it (see
//     model.StructElement.Refs comment).
//
// Both halves share the structure-tree walk: pass 1 collects the set
// of /ID values that exist anywhere in the tree, pass 2 walks the
// tree again and verifies (a) every Note has /ID and (b) every
// /Ref-target has an /ID present in that set.
type NoteID struct{}

func (NoteID) ID() string                { return "MH-19-001" }
func (NoteID) Title() string             { return "Notes carry /ID and references resolve" }
func (NoteID) Category() engine.Category { return engine.CategoryNotes }
func (NoteID) Severity() engine.Severity { return engine.SeverityError }
func (NoteID) Spec() engine.Spec         { return engine.SpecBoth }
func (NoteID) WCAG() []string            { return []string{"1.3.1", "2.4.4"} }
func (NoteID) Description() string {
	return "PDF/UA-1 §7.10 requires Note structure elements to carry an /ID so other elements (typically Reference) can target them, and requires every /Ref to resolve to an existing tagged element. Without /ID a Note cannot be reached as a labelled anchor; a /Ref targeting an untagged or /ID-less element produces a broken cross-reference."
}

func (c NoteID) Run(doc model.Document) []engine.Finding {
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

	// Pass 1: collect every /ID value present in the tree.
	ids := map[string]bool{}
	collectIDs(root, ids)

	// Pass 2: walk again, emitting findings for Notes without /ID
	// and for /Ref-targets whose /ID is missing or not in the set.
	var findings []engine.Finding
	considered := 0
	c.walk(root, "/"+root.Type(), ids, &findings, &considered)
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Note or /Ref-bearing elements -- nothing to inspect",
		}}
	}
	return findings
}

// collectIDs walks the tree once and records every non-empty /ID it
// finds. The map is the "what exists" set the second pass checks
// /Ref targets against.
func collectIDs(elem model.StructElement, out map[string]bool) {
	if id := elem.Attr("ID"); id != "" {
		out[id] = true
	}
	for _, child := range elem.Children() {
		collectIDs(child, out)
	}
}

func (c NoteID) walk(elem model.StructElement, path string, ids map[string]bool, out *[]engine.Finding, count *int) {
	hadInterest := false

	if elem.Type() == "Note" {
		hadInterest = true
		if elem.Attr("ID") == "" {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("Note at %s has no /ID", path),
				Hint:     "Set /ID on the Note structure element to a stable byte-string identifier; references to the note carry the same value in their /Ref array.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}

	refs := elem.Refs()
	if len(refs) > 0 {
		hadInterest = true
		for _, t := range refs {
			tid := t.Attr("ID")
			if tid == "" {
				*out = append(*out, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("%s /Ref target %s has no /ID -- the cross-link cannot be labelled", path, t.Type()),
					Hint:     "Add /ID on the referenced structure element so the /Ref array can label the destination for assistive technology.",
					Location: &engine.Location{Page: elem.Page(), StructPath: path},
				})
				continue
			}
			if !ids[tid] {
				*out = append(*out, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("%s /Ref target /ID %q is not present anywhere in the structure tree", path, tid),
					Hint:     "Update the /Ref to point at an existing tagged element (typically a Note), or remove the dangling reference.",
					Location: &engine.Location{Page: elem.Page(), StructPath: path},
				})
			}
		}
	}

	if hadInterest {
		*count++
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), ids, out, count)
	}
}

func init() { engine.Register(NoteID{}) }
