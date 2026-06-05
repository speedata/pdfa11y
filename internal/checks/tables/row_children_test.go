package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestRowChildren(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"TR with only TD passes", "testdata/table-row-cells.pdf", true, 0},
		{"TR with P child fails", "testdata/table-row-mixed.pdf", false, 1},
	}

	check := tables.RowChildren{}
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

// TestRowChildren_NoTR asserts N/A on a document whose structure tree
// contains no TR at all.
func TestRowChildren_NoTR(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/table-no-tr.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := tables.RowChildren{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
