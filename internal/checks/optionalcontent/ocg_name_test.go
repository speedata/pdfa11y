package optionalcontent_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/optionalcontent"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestOCGName(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"OCG with /Name passes", "testdata/ocg-with-name.pdf", true, 0},
		{"OCG without /Name fails", "testdata/ocg-no-name.pdf", false, 1},
	}

	check := optionalcontent.OCGName{}
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

// TestOCGName_NoOCGs asserts N/A on a document without /OCProperties.
func TestOCGName_NoOCGs(t *testing.T) {
	doc, err := pdf.LoadFile("../metadata/testdata/no-title.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := optionalcontent.OCGName{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
