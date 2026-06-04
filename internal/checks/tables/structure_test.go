package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestStructure(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Table with TR passes", "testdata/table-ok.pdf", true, 0},
		{"Table without TR fails", "testdata/table-no-tr.pdf", false, 1},
	}

	check := tables.Structure{}
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
				t.Errorf("findings count = %d, want %d", len(findings), tc.wantFindings)
			}
		})
	}
}
