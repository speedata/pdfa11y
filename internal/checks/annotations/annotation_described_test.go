package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestAnnotationDescribed covers UA-28-025: a visible non-Widget annotation
// must expose a /Contents entry or an /Alt on its enclosing structure element
// (ISO 14289-1 §7.18.1; veraPDF UA1:7.18.1-2).
func TestAnnotationDescribed(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/annot-highlight-bare.pdf", engine.VerdictFail},     // no Contents, not in tree
		{"testdata/annot-highlight-contents.pdf", engine.VerdictPass}, // has /Contents
		{"testdata/annot-highlight-alt.pdf", engine.VerdictPass},      // enclosing element has /Alt
		{"testdata/widget-in-form.pdf", engine.VerdictNA},             // only a Widget annotation
	}
	check := annotations.AnnotationDescribed{}
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
