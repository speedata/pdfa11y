package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestGroups covers UA-15-009: a Table has at most one THead and one TFoot,
// with a TBody whenever it is grouped (ISO 14289-1 §7.2; veraPDF
// UA1:7.2-11..-14).
func TestGroups(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/table-groups-ok.pdf", engine.VerdictPass},      // THead + TBody + TFoot
		{"testdata/table-groups-no-body.pdf", engine.VerdictFail}, // THead + TFoot but no TBody
		{"testdata/table-ok.pdf", engine.VerdictPass},             // simple Table -> TR (no groups)
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},    // no table elements
	}
	check := tables.Groups{}
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
