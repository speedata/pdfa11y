package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestSuspects covers UA-01-016: catalog /MarkInfo /Suspects must not be true
// (ISO 14289-1 §7.1; veraPDF UA1:7.1-4).
func TestSuspects(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/suspects-true.pdf", engine.VerdictFail}, // /Suspects true
		{"testdata/tagged.pdf", engine.VerdictPass},        // /MarkInfo without /Suspects
	}
	check := structure.Suspects{}
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
