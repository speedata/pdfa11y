package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestGeometry covers UA-15-006: table regularity (grid model with RowSpan /
// ColSpan).
func TestGeometry(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/grid-regular.pdf", engine.VerdictPass},         // 2x2
		{"testdata/grid-rowspan-regular.pdf", engine.VerdictPass}, // [TD rs2, TD] / [TD]
		{"testdata/grid-short-row.pdf", engine.VerdictFail},       // second row too short
		{"testdata/grid-overlap.pdf", engine.VerdictFail},         // colspan runs into a rowspan
		{"testdata/table-ok.pdf", engine.VerdictPass},             // existing regular-table fixture
	}
	check := tables.Geometry{}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.want {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.want, result.Findings)
			}
		})
	}
}
