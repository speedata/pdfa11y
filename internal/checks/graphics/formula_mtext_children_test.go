package graphics_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestMTextChildren(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string
	}{
		{
			"whitelisted PDF tag child passes",
			"testdata/mtext-children-ok.pdf",
			engine.VerdictPass,
			"",
		},
		{
			"non-whitelisted known PDF tag fails",
			"testdata/mtext-children-bad.pdf",
			engine.VerdictFail,
			`mtext child "P"`,
		},
		{
			"unknown tag (presumed MathML) is silently accepted",
			// 'mi' is a MathML element name; without /NS-map awareness
			// MH-17-006 treats it as "not a known PDF tag" and passes.
			"testdata/mtext-children-mathml-tag.pdf",
			engine.VerdictPass,
			"",
		},
		{
			"document with no mtext is N/A",
			// formula-mathml-af uses Variant B (AF), no mtext struct
			// elements anywhere.
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
	check := graphics.MTextChildren{}
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
