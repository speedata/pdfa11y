package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestRichAnnotContents covers UA-28-011: Ink/Screen/3D/RichMedia annotations
// must declare a /Contents alternate description.
func TestRichAnnotContents(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/screen-with-contents.pdf", engine.VerdictPass}, // Screen + /Contents
		{"testdata/screen-no-contents.pdf", engine.VerdictFail},   // Screen, no /Contents
		{"testdata/ink-no-contents.pdf", engine.VerdictFail},      // Ink, no /Contents
		{"testdata/link-with-contents.pdf", engine.VerdictNA},     // no rich-media subtype present
	}
	check := annotations.RichAnnotContents{}
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
