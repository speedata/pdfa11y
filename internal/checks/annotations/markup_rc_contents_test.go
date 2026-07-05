package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestMarkupRCContents covers UA-28-034: when a markup annotation carries both
// an /RC rich-text contents entry and a /Contents entry, they must be textually
// equivalent (ISO 14289-2 §8.9.2.3; veraPDF UA2:8.9.2.3-2).
func TestMarkupRCContents(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/rc-mismatch.pdf", engine.VerdictFail},  // /RC and /Contents differ
		{"testdata/rc-match.pdf", engine.VerdictPass},     // /RC plain text equals /Contents
		{"testdata/rc-no-contents.pdf", engine.VerdictNA}, // /RC but no /Contents: nothing to compare
	}
	check := annotations.MarkupRCContents{}
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
