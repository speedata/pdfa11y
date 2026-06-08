package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// ReadingOrder fires at Warning severity, so Passed() stays true on
// the FAIL fixture; the test asserts Verdict (Pass vs Warn) and the
// per-finding severity instead.
func TestReadingOrder(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantVerdict  engine.Verdict
		wantFindings int
		wantSeverity engine.Severity
	}{
		{"top-to-bottom layout passes", "testdata/reading-order-ok.pdf", engine.VerdictPass, 0, 0},
		{"reversed layout warns", "testdata/reading-order-reversed.pdf", engine.VerdictWarn, 1, engine.SeverityWarning},
		{"two-column natural order passes", "testdata/reading-order-two-col-ok.pdf", engine.VerdictPass, 0, 0},
		{"two-column hop backwards warns", "testdata/reading-order-two-col-hop.pdf", engine.VerdictWarn, 1, engine.SeverityWarning},
	}

	check := structure.ReadingOrder{}
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
