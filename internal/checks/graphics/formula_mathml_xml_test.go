package graphics_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestFormulaMathMLXML(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string // substring of the first finding's Message, "" to skip
	}{
		{
			"well-formed MathML passes",
			"testdata/formula-mathml-af.pdf",
			engine.VerdictPass,
			"",
		},
		{
			"empty stream fails",
			"testdata/formula-mathml-empty.pdf",
			engine.VerdictFail,
			"embedded stream is empty",
		},
		{
			"malformed XML fails",
			"testdata/formula-mathml-malformed.pdf",
			engine.VerdictFail,
			"not well-formed XML",
		},
		{
			"wrong root element fails",
			"testdata/formula-mathml-wrong-root.pdf",
			engine.VerdictFail,
			"root element is <foo>",
		},
		{
			"wrong namespace fails",
			"testdata/formula-mathml-wrong-namespace.pdf",
			engine.VerdictFail,
			`namespace "http://example.org/wrong"`,
		},
		{
			"PDF/UA-1 document is N/A",
			"testdata/formula-with-alt.pdf",
			engine.VerdictNA,
			"",
		},
		{
			"PDF/UA-2 without MathML AF is N/A",
			// The /Source LaTeX fixture carries an AF but its Subtype is
			// application/x-tex, so MH-17-004 has nothing to inspect.
			"testdata/formula-tex-only-af.pdf",
			engine.VerdictNA,
			"",
		},
	}
	check := graphics.FormulaMathMLXML{}
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
					t.Fatalf("expected finding with %q, got none", tc.wantMessage)
				}
				if !strings.Contains(findings[0].Message, tc.wantMessage) {
					t.Errorf("Message = %q, want substring %q", findings[0].Message, tc.wantMessage)
				}
			}
		})
	}
}
