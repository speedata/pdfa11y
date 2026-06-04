package structure_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestMCIDConsistency(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string
	}{
		{
			// glu-pdfua-demo: walker observes 12 ContentMCIDs and
			// 12 StructTreeMCIDs on page 1, fully aligned.
			name:      "well-formed PDF passes",
			fixture:   "../../realworld/testdata/glu-pdfua-demo.pdf",
			wantState: engine.VerdictPass,
		},
		{
			// untagged-content.pdf: BDC /P /MCID 0 in content, but
			// the structure tree has no MCR pointing at it. The
			// content's MCID is orphan.
			name:        "content MCID without struct-tree pointer fails",
			fixture:     "testdata/untagged-content.pdf",
			wantState:   engine.VerdictFail,
			wantMessage: "no structure element references them",
		},
		{
			// Synthetic fixtures without /Contents and without MCRs:
			// neither side has MCIDs, so the check has nothing to
			// inspect.
			name:      "MCID-less document is N/A",
			fixture:   "testdata/tagged.pdf",
			wantState: engine.VerdictNA,
		},
	}
	check := structure.MCIDConsistency{}
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
				found := false
				for _, f := range findings {
					if strings.Contains(f.Message, tc.wantMessage) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no finding contains %q; messages: %+v", tc.wantMessage, findings)
				}
			}
		})
	}
}
