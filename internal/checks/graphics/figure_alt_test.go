package graphics_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestFigureAlt(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Figure with /Alt passes", "testdata/figure-with-alt.pdf", true, 0},
		{"Figure without /Alt fails", "testdata/figure-no-alt.pdf", false, 1},
	}

	check := graphics.FigureAlt{}
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
			if !tc.wantPass {
				if findings[0].Location == nil {
					t.Errorf("finding has no Location; expected struct path")
				} else if !strings.Contains(findings[0].Location.StructPath, "Figure") {
					t.Errorf("location path %q does not mention Figure", findings[0].Location.StructPath)
				}
			}
		})
	}
}

// TestFigureAlt_NoStructTree confirms the check declares itself N/A
// on an untagged document instead of vacuously passing. MH-01-005 is
// the one reporting the missing structure tree as an error.
func TestFigureAlt_NoStructTree(t *testing.T) {
	doc, err := pdf.LoadFile("../structure/testdata/untagged.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := graphics.FigureAlt{}
	findings := check.Run(doc)
	result := engine.Result{Check: check, Findings: findings}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
