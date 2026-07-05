package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestStampNameOrContents covers UA-28-021: a rubber-stamp annotation must
// carry a /Name or /Contents entry (ISO 14289-2 §8.9.2.4.7; veraPDF
// UA2:8.9.2.4.7-1).
func TestStampNameOrContents(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/stamp-none.pdf", engine.VerdictFail},        // neither Name nor Contents
		{"testdata/stamp-contents.pdf", engine.VerdictPass},    // Contents present
		{"testdata/stamp-name.pdf", engine.VerdictPass},        // Name present
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no Stamp annotations
	}
	check := annotations.StampNameOrContents{}
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
