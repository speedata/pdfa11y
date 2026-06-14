package notes_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/notes"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestNoteNotInUA2(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"PDF/UA-2 with Note element fails",
			"testdata/note-in-ua2.pdf",
			engine.VerdictFail,
		},
		{
			// note-with-id.pdf is a PDF/UA-1 fixture; the check is
			// spec-gated to UA-2, so it declines.
			"PDF/UA-1 document is N/A",
			"testdata/note-with-id.pdf",
			engine.VerdictNA,
		},
	}
	check := notes.NoteNotInUA2{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantState {
				t.Errorf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
		})
	}
}
