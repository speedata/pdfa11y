package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestTrapNet covers UA-28-012: a visible TrapNet annotation is forbidden
// (ISO 14289-1 §7.18.2; veraPDF UA1:7.18.2-1).
func TestTrapNet(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/trapnet-visible.pdf", engine.VerdictFail},   // on-page, not hidden
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no TrapNet
	}
	check := annotations.TrapNet{}
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
