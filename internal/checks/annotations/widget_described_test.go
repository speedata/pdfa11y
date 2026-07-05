package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestWidgetDescribed covers UA-28-026: a visible Widget annotation must expose
// a /TU tooltip or an /Alt on its enclosing structure element (ISO 14289-1
// §7.18.1; veraPDF UA1:7.18.1-3).
func TestWidgetDescribed(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/widget-with-tu.pdf", engine.VerdictPass},     // has /TU
		{"testdata/widget-in-alt.pdf", engine.VerdictPass},      // enclosing element has /Alt
		{"testdata/widget-no-tu.pdf", engine.VerdictFail},       // neither /TU nor /Alt
		{"testdata/annot-highlight-bare.pdf", engine.VerdictNA}, // no Widget annotations
	}
	check := annotations.WidgetDescribed{}
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
