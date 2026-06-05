package graphics_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestFormulaAlt(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Formula with /Alt passes", "testdata/formula-with-alt.pdf", true, 0},
		{"Formula without /Alt fails", "testdata/formula-no-alt.pdf", false, 1},
	}

	check := graphics.FormulaAlt{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.Passed(); got != tc.wantPass {
				t.Fatalf("Passed() = %v, want %v (findings: %+v)", got, tc.wantPass, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
		})
	}
}

// TestFormulaAlt_NoFormula asserts N/A on a document whose structure
// tree contains no Formula element. Reuses the figure fixture (a
// non-Formula tagged document with a Figure child).
func TestFormulaAlt_NoFormula(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/figure-with-alt.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := graphics.FormulaAlt{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
