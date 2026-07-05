package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestParentEntry covers UA-01-014: every structure element must carry a /P
// (parent) entry (ISO 14289-2 §8.2.1, veraPDF UA2:8.2.1-2).
func TestParentEntry(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/tagged.pdf", engine.VerdictPass},                // Document has /P, no other elements
		{"testdata/struct-missing-parent.pdf", engine.VerdictFail}, // child P element omits /P
		{"testdata/untagged.pdf", engine.VerdictNA},                // no structure tree
	}
	check := structure.ParentEntry{}
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
