package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestDynamicXFA covers UA-28-028: dynamic XFA forms are forbidden (ISO
// 14289-1 §7.15; veraPDF UA1:7.15-1). Static XFA is permitted under PDF/UA-1.
func TestDynamicXFA(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/xfa-dynamic.pdf", engine.VerdictFail},               // <dynamicRender>required
		{"testdata/xfa-static.pdf", engine.VerdictPass},                // <dynamicRender>forbidden
		{"../navigation/testdata/outlines-missing.pdf", engine.VerdictNA}, // no XFA form
	}
	check := annotations.DynamicXFA{}
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
