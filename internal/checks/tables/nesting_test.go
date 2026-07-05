package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestNesting covers UA-15-008: the table sub-elements must follow the
// canonical parent/child nesting (ISO 14289-1 §7.2; veraPDF UA1:7.2-3..-9,
// -36..-38 plus the Caption rules -16/-39).
func TestNesting(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/table-ok.pdf", engine.VerdictPass},          // canonical Table -> TR -> TH/TD
		{"testdata/table-no-tr.pdf", engine.VerdictFail},       // Table has a P child (7.2-3)
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no table elements
	}
	check := tables.Nesting{}
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
