package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestFormSingleWidget covers UA-28-027: a Form structure element must enclose
// at most one widget annotation (ISO 14289-2 §8.10.1; veraPDF UA2:8.10.1-2).
func TestFormSingleWidget(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/form-two-widgets.pdf", engine.VerdictFail}, // Form encloses two widgets
		{"testdata/form-one-widget.pdf", engine.VerdictPass},  // Form encloses one widget
		{"testdata/widget-in-p.pdf", engine.VerdictNA},        // no Form structure element
	}
	check := annotations.FormSingleWidget{}
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
