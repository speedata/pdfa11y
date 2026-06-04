package headings_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/headings"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestHierarchy(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
		wantMessage  string
	}{
		{
			name:         "H1 H2 H3 passes",
			fixture:      "testdata/heading-ok.pdf",
			wantPass:     true,
			wantFindings: 0,
		},
		{
			name:         "H1 H3 fails on the skip",
			fixture:      "testdata/heading-skip.pdf",
			wantPass:     false,
			wantFindings: 1,
			wantMessage:  "H3 follows H1",
		},
		{
			// Role-mapped MyHeading -> H1, then H3. With role-map
			// resolution the sequence reads as H1 -> H3 and the
			// level skip is caught; without resolution the custom
			// type would be ignored.
			name:         "rolemapped MyHeading then H3 fails on the skip",
			fixture:      "testdata/heading-rolemap-skip.pdf",
			wantPass:     false,
			wantFindings: 1,
			wantMessage:  "H3 follows H1",
		},
		{
			// Document outline starts at H3. The H3 -> H4 step
			// itself is fine; only the missing top-level H1 is
			// reported, so the finding count stays at 1.
			name:         "starting heading is not H1",
			fixture:      "testdata/heading-not-h1.pdf",
			wantPass:     false,
			wantFindings: 1,
			wantMessage:  "first heading is H3, expected H1",
		},
	}

	check := headings.Hierarchy{}
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
				t.Fatalf("findings count = %d, want %d", len(findings), tc.wantFindings)
			}
			if tc.wantMessage != "" && !strings.Contains(findings[0].Message, tc.wantMessage) {
				t.Errorf("message %q does not contain %q", findings[0].Message, tc.wantMessage)
			}
		})
	}
}

// TestHierarchy_NoStructTree confirms the check declares itself N/A
// on an untagged document. The tagging gap belongs to MH-01-005;
// reporting PASS here would be misleading.
func TestHierarchy_NoStructTree(t *testing.T) {
	doc, err := pdf.LoadFile("../structure/testdata/untagged.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	check := headings.Hierarchy{}
	findings := check.Run(doc)
	result := engine.Result{Check: check, Findings: findings}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
