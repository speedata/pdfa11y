package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestWidgetEnclosure covers UA-28-014: a visible Widget annotation must be
// nested within a Form structure element (ISO 14289-1 §7.18.4; veraPDF
// UA1:7.18.4-1).
func TestWidgetEnclosure(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/widget-in-form.pdf", engine.VerdictPass},    // Widget in a Form element
		{"testdata/widget-in-p.pdf", engine.VerdictFail},       // Widget in a P element
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no Widget annotations
	}
	check := annotations.WidgetEnclosure{}
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
