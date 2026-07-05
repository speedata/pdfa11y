package notes

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// NoteIDUnique fails when two or more Note structure elements share the same
// /ID. ISO 14289-1 §7.9 (veraPDF UA1:7.9-2) requires each Note tag to have a
// unique /ID: the /ID is the anchor a Reference targets, so a value reused
// across notes makes the cross-reference ambiguous -- assistive technology
// cannot tell which note a jump should land on.
//
// Only Notes that carry a non-empty /ID participate; a Note without any /ID is
// the concern of UA-19-001 (§7.9 test 1), not this uniqueness rule. One finding
// per colliding /ID value. N/A when the document has no Note elements.
//
// PDF/UA-1 only: PDF/UA-2 replaces Note with FENote and drops the /ID anchoring
// model (see UA-19-001 / UA-14-009).
type NoteIDUnique struct{}

func (NoteIDUnique) ID() string                { return "UA-19-003" }
func (NoteIDUnique) Title() string             { return "Note structure elements have unique /ID values" }
func (NoteIDUnique) Category() engine.Category { return engine.CategoryNotes }
func (NoteIDUnique) Severity() engine.Severity { return engine.SeverityError }
func (NoteIDUnique) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (NoteIDUnique) WCAG() []string            { return []string{"1.3.1", "2.4.4"} }
func (NoteIDUnique) Description() string {
	return "PDF/UA-1 §7.9 requires each Note structure element to have a unique /ID. The /ID is the anchor a Reference element targets; reusing the same value across notes makes the cross-reference ambiguous, so assistive technology cannot resolve which note a jump refers to."
}

func (c NoteIDUnique) Run(doc model.Document) []engine.Finding {
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

	// Map each non-empty Note /ID to the number of Notes that carry it.
	counts := map[string]int{}
	notes := 0
	collectNoteIDs(root, counts, &notes)
	if notes == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Note structure elements -- nothing to inspect",
		}}
	}

	var dup []string
	for id, n := range counts {
		if n > 1 {
			dup = append(dup, id)
		}
	}
	if len(dup) == 0 {
		return nil
	}
	sort.Strings(dup)
	var findings []engine.Finding
	for _, id := range dup {
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("/ID %q is shared by %d Note elements -- Note IDs must be unique", id, counts[id]),
			Hint:     "Give each Note structure element a distinct /ID; update the matching /Ref targets accordingly.",
		})
	}
	return findings
}

// collectNoteIDs walks the tree and, for every Note element with a non-empty
// /ID, increments that ID's count. notes tracks how many Note elements exist
// at all so the caller can distinguish "no Notes" (N/A) from "Notes present,
// none colliding" (pass).
func collectNoteIDs(elem model.StructElement, counts map[string]int, notes *int) {
	if elem.Type() == "Note" {
		*notes++
		if id := elem.Attr("ID"); id != "" {
			counts[id]++
		}
	}
	for _, child := range elem.Children() {
		collectNoteIDs(child, counts, notes)
	}
}

func init() { engine.Register(NoteIDUnique{}) }
