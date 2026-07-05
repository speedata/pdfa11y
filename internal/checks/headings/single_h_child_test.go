package headings_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/headings"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestSingleHChild covers UA-14-008: a node may contain at most one direct
// child H (unnumbered heading) tag (ISO 14289-1 §7.4.4; veraPDF UA1:7.4.4-1).
func TestSingleHChild(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/heading-two-h.pdf", engine.VerdictFail},    // Sect with two H children
		{"testdata/heading-single-h.pdf", engine.VerdictPass}, // Sect with one H, one P
		{"testdata/heading-ok.pdf", engine.VerdictPass},       // numbered headings only, no H
		{"../structure/testdata/untagged.pdf", engine.VerdictNA},
	}
	check := headings.SingleHChild{}
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
