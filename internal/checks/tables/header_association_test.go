package tables_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/tables"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestHeaderAssociation covers UA-15-007: a table with header cells must
// associate them via /Scope or /Headers+/ID (PDF/UA-2 §8.2.5.26-5/-6).
func TestHeaderAssociation(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/header-scope.pdf", engine.VerdictPass},     // every TH scoped
		{"testdata/header-ids.pdf", engine.VerdictPass},       // TD /Headers -> defined /ID
		{"testdata/header-none.pdf", engine.VerdictFail},      // TH present, no mechanism
		{"testdata/header-undefined.pdf", engine.VerdictFail}, // TD /Headers -> undefined ID
		{"testdata/table-ok.pdf", engine.VerdictNA},           // table without TH cells
	}
	check := tables.HeaderAssociation{}
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
