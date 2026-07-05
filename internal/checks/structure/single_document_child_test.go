package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestSingleDocumentChild covers UA-01-015: the structure tree root must have
// a single Document element as its only child (ISO 14289-2 §8.2.5.2,
// veraPDF UA2:8.2.5.2-1).
func TestSingleDocumentChild(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/tagged.pdf", engine.VerdictPass},              // single Document child
		{"testdata/root-multi-document.pdf", engine.VerdictFail}, // two top-level Document elements
		{"testdata/untagged.pdf", engine.VerdictNA},              // no structure tree
	}
	check := structure.SingleDocumentChild{}
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
