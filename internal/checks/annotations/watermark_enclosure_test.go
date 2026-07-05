package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestWatermarkEnclosure covers UA-28-019: a Watermark annotation used as real
// content must be enclosed in an Annot element (ISO 14289-2 §8.9.2.4.16;
// veraPDF UA2:8.9.2.4.16-1).
func TestWatermarkEnclosure(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/watermark-in-annot.pdf", engine.VerdictPass}, // Watermark in an Annot element
		{"testdata/watermark-in-p.pdf", engine.VerdictFail},     // Watermark in a P element
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},  // no Watermark annotations
	}
	check := annotations.WatermarkEnclosure{}
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
