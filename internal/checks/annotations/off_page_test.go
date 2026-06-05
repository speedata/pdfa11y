package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestOffPage(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Off-page annotation with Hidden flag passes", "testdata/offpage-hidden.pdf", true, 0},
		{"Off-page annotation without Hidden flag fails", "testdata/offpage-visible.pdf", false, 1},
	}

	check := annotations.OffPage{}
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

// TestOffPage_OnlyOnPage asserts N/A when no annotation in the document
// sits outside the page box: there is nothing for MH-28-008 to inspect.
func TestOffPage_OnlyOnPage(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/link-with-contents.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := annotations.OffPage{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
