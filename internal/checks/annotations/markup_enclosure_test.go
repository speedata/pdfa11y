package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestMarkupEnclosure covers UA-28-015: a structure-tree markup annotation
// must be enclosed in an Annot element (ISO 14289-2 §8.9.2.3; veraPDF
// UA2:8.9.2.3-1).
func TestMarkupEnclosure(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/markup-in-annot.pdf", engine.VerdictPass},   // Highlight in an Annot element
		{"testdata/markup-in-p.pdf", engine.VerdictFail},       // Highlight in a P element
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no markup annotations
	}
	check := annotations.MarkupEnclosure{}
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
