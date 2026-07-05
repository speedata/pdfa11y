package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestLinkEnclosure covers UA-28-013: a visible Link annotation must be
// enclosed in a Link structure element (ISO 14289-1 §7.18.5; veraPDF
// UA1:7.18.5-1).
func TestLinkEnclosure(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/link-tagged.pdf", engine.VerdictPass},        // Link annot in a Link element
		{"testdata/link-with-contents.pdf", engine.VerdictFail}, // Link annot not in a Link element
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},  // no Link annotations
	}
	check := annotations.LinkEnclosure{}
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
