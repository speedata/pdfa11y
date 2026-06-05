package headings_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/headings"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestStyleMix(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"only H<n> headings pass", "testdata/heading-style-hn-only.pdf", true, 0},
		{"H mixed with H<n> fails", "testdata/heading-style-mixed.pdf", false, 2},
	}

	check := headings.StyleMix{}
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

// TestStyleMix_NoHeadings asserts N/A on a document whose structure
// tree contains no heading elements at all.
func TestStyleMix_NoHeadings(t *testing.T) {
	doc, err := pdf.LoadFile("../structure/testdata/tagged.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := headings.StyleMix{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
