package graphics_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestMathChildNamespace(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string
	}{
		{
			"math child in W3C MathML namespace passes",
			"testdata/formula-math-ns-mathml.pdf",
			engine.VerdictPass,
			"",
		},
		{
			"math child in wrong namespace fails",
			"testdata/formula-math-ns-wrong.pdf",
			engine.VerdictFail,
			`namespace "http://example.org/wrong"`,
		},
		{
			"math child with no namespace fails",
			"testdata/formula-math-ns-none.pdf",
			engine.VerdictFail,
			"declares no namespace",
		},
		{
			"document with no math child is N/A",
			// formula-mathml-af uses Variant B (AF), no math struct child.
			"testdata/formula-mathml-af.pdf",
			engine.VerdictNA,
			"",
		},
		{
			"PDF/UA-1 document is N/A",
			"testdata/formula-with-alt.pdf",
			engine.VerdictNA,
			"",
		},
	}
	check := graphics.MathChildNamespace{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantState {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
			if tc.wantMessage != "" {
				if len(findings) == 0 {
					t.Fatalf("expected finding containing %q, got none", tc.wantMessage)
				}
				if !strings.Contains(findings[0].Message, tc.wantMessage) {
					t.Errorf("Message = %q, want substring %q", findings[0].Message, tc.wantMessage)
				}
			}
		})
	}
}
