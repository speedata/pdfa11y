package notes_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/notes"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestNoteID(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
		messagePart  string
	}{
		{"Note with /ID passes", "testdata/note-with-id.pdf", true, 0, ""},
		{"Note without /ID fails (half A)", "testdata/note-no-id.pdf", false, 1, "Note"},
		{"Reference to bare element fails (half B)", "testdata/ref-unresolved.pdf", false, 1, "/Ref target"},
	}

	check := notes.NoteID{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.Passed(); got != tc.wantPass {
				t.Fatalf("Passed() = %v, want %v (findings: %+v)", got, tc.wantPass, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
			if tc.messagePart != "" && !strings.Contains(findings[0].Message, tc.messagePart) {
				t.Errorf("finding message %q does not contain %q", findings[0].Message, tc.messagePart)
			}
		})
	}
}

// TestNoteID_NoNotesOrRefs asserts the check declines on a tagged
// document that contains neither a Note nor any /Ref-bearing element.
func TestNoteID_NoNotesOrRefs(t *testing.T) {
	doc, err := pdf.LoadFile("../structure/testdata/tagged.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := notes.NoteID{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}

// TestNoteID_NoStructTree asserts N/A on an untagged document.
func TestNoteID_NoStructTree(t *testing.T) {
	doc, err := pdf.LoadFile("../structure/testdata/untagged.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := notes.NoteID{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
