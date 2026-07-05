package navigation_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/navigation"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestStructureDestinations covers UA-27-003: intra-document destinations
// must be structure destinations (ISO 14289-2 §8.8; veraPDF UA2:8.8-1/-2).
func TestStructureDestinations(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/dest-page.pdf", engine.VerdictFail},        // outline /Dest -> page (8.8-1)
		{"testdata/dest-goto-page.pdf", engine.VerdictFail},   // outline /A GoTo /D -> page (8.8-2)
		{"testdata/dest-structelem.pdf", engine.VerdictPass},  // outline /Dest -> StructElem
		{"testdata/outlines-missing.pdf", engine.VerdictNA},   // no destinations at all
	}
	check := navigation.StructureDestinations{}
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
