package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestScope(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"TH with /Scope passes", "testdata/table-th-with-scope.pdf", true, 0},
		{"TH without /Scope fails", "testdata/table-th-no-scope.pdf", false, 1},
		// /Scope can be attached via /C → ClassMap as well (ISO
		// 32000-1 §14.7.5.4); pdfa11y must accept both routes.
		{"TH with /Scope via ClassMap passes", "testdata/table-th-scope-via-classmap.pdf", true, 0},
	}

	check := tables.Scope{}
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

// TestScope_NoTH asserts N/A on a document whose structure tree
// contains no TH at all.
func TestScope_NoTH(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/table-row-cells.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := tables.Scope{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
