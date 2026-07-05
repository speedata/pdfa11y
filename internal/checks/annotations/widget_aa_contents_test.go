package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestWidgetAAContents covers UA-28-029: a field widget carrying an /AA
// additional-actions dictionary must provide a /Contents description (ISO
// 14289-2 §8.10.2.3; veraPDF UA2:8.10.2.3-2).
func TestWidgetAAContents(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/widget-aa-no-contents.pdf", engine.VerdictFail}, // /AA without /Contents
		{"testdata/widget-aa-contents.pdf", engine.VerdictPass},    // /AA with /Contents
		{"testdata/form-one-widget.pdf", engine.VerdictNA},         // field widget, no /AA
	}
	check := annotations.WidgetAAContents{}
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
