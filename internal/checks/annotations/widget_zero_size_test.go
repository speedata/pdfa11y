package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestWidgetZeroSize covers UA-28-018: a zero-size Widget annotation must be an
// artifact, not tagged content (ISO 14289-2 §8.9.2.4.13; veraPDF
// UA2:8.9.2.4.13-1).
func TestWidgetZeroSize(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/widget-zero-tagged.pdf", engine.VerdictFail},   // zero-size widget in a P element
		{"testdata/widget-zero-untagged.pdf", engine.VerdictPass}, // zero-size widget, not in the tree
		{"testdata/widget-in-form.pdf", engine.VerdictNA},         // widget present but not zero-size
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},    // no Widget annotations
	}
	check := annotations.WidgetZeroSize{}
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
