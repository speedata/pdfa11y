package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestAnnotAltContents covers UA-28-031: an annotation's /Contents and its
// enclosing element's /Alt, when both present, must be identical (ISO 14289-2
// §8.9.4.2; veraPDF UA2:8.9.4.2-1).
func TestAnnotAltContents(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/annot-alt-contents-differ.pdf", engine.VerdictFail}, // Contents != Alt
		{"testdata/annot-alt-contents-match.pdf", engine.VerdictPass},  // Contents == Alt
		{"testdata/annot-alt-no-contents.pdf", engine.VerdictNA},       // Alt present, no Contents
	}
	check := annotations.AnnotAltContents{}
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
