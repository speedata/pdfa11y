package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestCaptionPosition covers UA-01-011: a Caption must be the first or last
// child of its parent.
func TestCaptionPosition(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/caption-first.pdf", engine.VerdictPass},  // [Caption, TR, TR]
		{"testdata/caption-last.pdf", engine.VerdictPass},   // [TR, TR, Caption]
		{"testdata/caption-middle.pdf", engine.VerdictFail}, // [TR, Caption, TR]
		{"testdata/tagged.pdf", engine.VerdictNA},           // no Caption element
	}
	check := structure.CaptionPosition{}
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
