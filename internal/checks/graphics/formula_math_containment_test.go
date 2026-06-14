package graphics_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestMathContainment(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"math under Formula passes",
			"testdata/formula-math-ns-mathml.pdf",
			engine.VerdictPass,
		},
		{
			"math without Formula ancestor fails",
			"testdata/math-stranded.pdf",
			engine.VerdictFail,
		},
		{
			// formula-math-ns-wrong has a math element in a non-MathML
			// namespace; this check only fires on math elements that
			// resolve to the W3C MathML namespace, so it's N/A here.
			"math in non-MathML namespace is N/A",
			"testdata/formula-math-ns-wrong.pdf",
			engine.VerdictNA,
		},
		{
			"PDF/UA-1 document is N/A",
			"testdata/formula-with-alt.pdf",
			engine.VerdictNA,
		},
	}
	check := graphics.MathContainment{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantState {
				t.Errorf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
		})
	}
}
