package structure_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestUntaggedContent(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string
	}{
		{
			// A real well-tagged PDF: every painting op is inside a
			// marked-content sequence. Check should pass.
			name:      "fully-tagged real PDF passes",
			fixture:   "../../realworld/testdata/glu-pdfua-demo.pdf",
			wantState: engine.VerdictPass,
		},
		{
			// untagged-content.pdf paints a Tj outside any BDC and
			// another Tj inside one -- exactly one violation.
			name:        "loose Tj outside BDC fails",
			fixture:     "testdata/untagged-content.pdf",
			wantState:   engine.VerdictFail,
			wantMessage: "outside any marked-content sequence",
		},
		{
			// Synthetic fixtures without any /Contents return N/A:
			// nothing to walk.
			name:      "page-less document is N/A",
			fixture:   "testdata/tagged.pdf",
			wantState: engine.VerdictNA,
		},
	}
	check := structure.UntaggedContent{}
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
					t.Fatalf("expected a finding containing %q, got none", tc.wantMessage)
				}
				if !strings.Contains(findings[0].Message, tc.wantMessage) {
					t.Errorf("message %q does not contain %q", findings[0].Message, tc.wantMessage)
				}
			}
		})
	}
}
