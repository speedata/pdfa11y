package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestWidgetLabelContents covers UA-28-035: a field widget in the structure
// tree must have either an associated Lbl label or a /Contents entry (ISO
// 14289-2 §8.10.2.3; veraPDF UA2:8.10.2.3-1).
func TestWidgetLabelContents(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/widget-no-lbl-no-contents.pdf", engine.VerdictFail}, // neither label nor Contents
		{"testdata/widget-with-lbl.pdf", engine.VerdictPass},           // Lbl child present
		{"testdata/widget-with-contents.pdf", engine.VerdictPass},      // /Contents present
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},         // no field widget in tree
	}
	check := annotations.WidgetLabelContents{}
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
