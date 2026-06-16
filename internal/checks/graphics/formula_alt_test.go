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
		{"Formula without /Alt or /ActualText fails", "testdata/formula-no-alt.pdf", false, 1},
		// veraPDF UA-1 §7.7 corpus: /Alt and /ActualText are not
		// symmetric. An empty /Alt fails (no description), an empty
		// /ActualText passes ("render as silence").
		{"Formula with empty /Alt fails", "testdata/formula-alt-empty.pdf", false, 1},
		{"Formula with /ActualText passes", "testdata/formula-actualtext.pdf", true, 0},
		{"Formula with empty /ActualText passes", "testdata/formula-actualtext-empty.pdf", true, 0},
		// PDF/UA-2 paths (BPG "Math in PDF"): MathML AF satisfies the
		// check; an AF that is only a LaTeX source does not.
		{"PDF/UA-2 Formula with MathML AF passes", "testdata/formula-mathml-af.pdf", true, 0},
		{"PDF/UA-2 Formula with LaTeX-only AF fails", "testdata/formula-tex-only-af.pdf", false, 1},
		// math struct child is only accepted when it actually lives
		// in the MathML namespace. A misnamed 'math' tag in the
		// default PDF namespace conveys no math semantics to AT.
		{"PDF/UA-2 math child in MathML namespace passes", "testdata/formula-math-ns-mathml.pdf", true, 0},
		{"PDF/UA-2 math child in wrong namespace fails", "testdata/formula-math-ns-wrong.pdf", false, 1},
		{"PDF/UA-2 math child with no namespace fails", "testdata/formula-math-ns-none.pdf", false, 1},
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
