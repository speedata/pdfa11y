package viewerprefs_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/viewerprefs"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestDisplayDocTitle(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantPass    bool
		wantMessage string
	}{
		{
			name:        "no ViewerPreferences fails",
			fixture:     "testdata/no-viewerprefs.pdf",
			wantPass:    false,
			wantMessage: "no ViewerPreferences",
		},
		{
			name:        "DisplayDocTitle=false fails",
			fixture:     "testdata/displaydoctitle-false.pdf",
			wantPass:    false,
			wantMessage: "DisplayDocTitle is false",
		},
		{
			name:     "DisplayDocTitle=true passes",
			fixture:  "testdata/displaydoctitle-true.pdf",
			wantPass: true,
		},
	}

	check := viewerprefs.DisplayDocTitle{}
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
			if tc.wantMessage != "" {
				if len(findings) == 0 {
					t.Fatalf("expected a finding mentioning %q, got none", tc.wantMessage)
				}
				if !strings.Contains(findings[0].Message, tc.wantMessage) {
					t.Errorf("message %q does not contain %q", findings[0].Message, tc.wantMessage)
				}
			}
		})
	}
}
