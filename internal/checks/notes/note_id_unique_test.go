package notes_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/notes"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestNoteIDUnique covers UA-19-003: Note structure elements must have unique
// /ID values (ISO 14289-1 §7.9; veraPDF UA1:7.9-2).
func TestNoteIDUnique(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/note-dup-id.pdf", engine.VerdictFail},         // two Notes share /ID
		{"testdata/note-unique-id.pdf", engine.VerdictPass},      // two Notes, distinct /ID
		{"testdata/note-with-id.pdf", engine.VerdictPass},        // single Note, no collision
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},   // no Note elements
		{"../structure/testdata/untagged.pdf", engine.VerdictNA}, // no structure tree
	}
	check := notes.NoteIDUnique{}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.want {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.want, result.Findings)
			}
		})
	}
}
