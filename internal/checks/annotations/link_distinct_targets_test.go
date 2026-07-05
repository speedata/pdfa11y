package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestLinkDistinctTargets covers UA-28-030: link annotations targeting
// different locations must be in separate Link/Reference structure elements
// (ISO 14289-2 §8.2.5.20; veraPDF UA2:8.2.5.20-2).
func TestLinkDistinctTargets(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/link-different-targets.pdf", engine.VerdictFail}, // one Link element, two targets
		{"testdata/link-same-target.pdf", engine.VerdictPass},       // one Link element, one target
		{"testdata/link-in-p.pdf", engine.VerdictNA},                // no Link/Reference element
	}
	check := annotations.LinkDistinctTargets{}
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
