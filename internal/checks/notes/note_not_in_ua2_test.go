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
		{
			// An Aside (valid PDF 2.0 type in the standard namespace)
			// that is role-mapped to Note for legacy 1.7 viewers must
			// not be flagged: the declared 2.0 type is authoritative,
			// the global /RoleMap is only a compatibility hint.
			"PDF/UA-2 Aside role-mapped to Note passes",
			"testdata/aside-rolemapped-to-note-ua2.pdf",
			engine.VerdictPass,
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
