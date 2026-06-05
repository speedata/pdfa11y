package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestFormTooltip(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Widget with /TU passes", "testdata/widget-with-tu.pdf", true, 0},
		{"Widget without /TU fails", "testdata/widget-no-tu.pdf", false, 1},
	}

	check := annotations.FormTooltip{}
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

// TestFormTooltip_NoWidgets asserts N/A on a document without any
// Widget annotations.
func TestFormTooltip_NoWidgets(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/link-with-contents.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := annotations.FormTooltip{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
