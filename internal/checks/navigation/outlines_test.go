package navigation_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/navigation"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// Outlines is a Warning, not an Error, so missing outlines leave
// Passed() == true. The test asserts Verdict (Pass vs Warn) instead.
func TestOutlines(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantVerdict  engine.Verdict
		wantFindings int
		wantSeverity engine.Severity
	}{
		{"22-page doc with /Outlines passes", "testdata/outlines-present.pdf", engine.VerdictPass, 0, 0},
		{"22-page doc without /Outlines warns", "testdata/outlines-missing.pdf", engine.VerdictWarn, 1, engine.SeverityWarning},
	}

	check := navigation.Outlines{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantVerdict {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.wantVerdict, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
			if tc.wantFindings > 0 && findings[0].Severity != tc.wantSeverity {
				t.Errorf("severity = %v, want %v", findings[0].Severity, tc.wantSeverity)
			}
		})
	}
}

// TestOutlines_BelowThreshold asserts the check declines on a short
// document; the 21-page threshold means single-page fixtures are N/A.
func TestOutlines_BelowThreshold(t *testing.T) {
	doc, err := pdf.LoadFile("../metadata/testdata/no-title.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := navigation.Outlines{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
