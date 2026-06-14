package annotations_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestDeprecatedAnnotTypes(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string
	}{
		{
			"PDF/UA-2 with Sound annotation fails",
			"testdata/sound-in-ua2.pdf",
			engine.VerdictFail,
			"Sound annotation",
		},
		{
			// link-with-contents.pdf is a PDF/UA-1 link fixture, no
			// deprecated annot types -- check declines.
			"PDF/UA-1 document is N/A",
			"testdata/link-with-contents.pdf",
			engine.VerdictNA,
			"",
		},
	}
	check := annotations.DeprecatedAnnotTypes{}
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
			if tc.wantMessage != "" {
				if len(findings) == 0 {
					t.Fatalf("expected finding mentioning %q, got none", tc.wantMessage)
				}
				if !strings.Contains(findings[0].Message, tc.wantMessage) {
					t.Errorf("Message = %q, want substring %q", findings[0].Message, tc.wantMessage)
				}
			}
		})
	}
}
