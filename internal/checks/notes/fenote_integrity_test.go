package notes_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/notes"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestFENoteIntegrity covers UA-19-002: bidirectional FENote /Ref and valid
// /NoteType (PDF/UA-2 §8.2.5.14-2/-3/-4).
func TestFENoteIntegrity(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/fenote-valid.pdf", engine.VerdictPass},        // citation <-> note, NoteType Footnote
		{"testdata/fenote-orphan.pdf", engine.VerdictFail},       // citation -> note, no back reference
		{"testdata/fenote-bad-notetype.pdf", engine.VerdictFail}, // NoteType "Sidebar"
		{"testdata/note-with-id.pdf", engine.VerdictNA},          // no FENote elements
	}
	check := notes.FENoteIntegrity{}
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
