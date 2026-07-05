package fonts_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/fonts"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestNotdefGlyph covers UA-10-004: a text-showing operator must not reference
// the .notdef glyph. A Type0/Identity font showing code 0 (CID 0 = glyph 0 =
// .notdef) fails (ISO 14289-1 §7.21.8 / ISO 14289-2 §8.4.5.9; veraPDF
// UA1:7.21.8-1 / UA2:8.4.5.9-1).
func TestNotdefGlyph(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/notdef-code0.pdf", engine.VerdictFail},            // Identity font shows code 0
		{"testdata/notdef-realglyph.pdf", engine.VerdictPass},        // shows a real code (1)
		{"testdata/notdef-cidgidmap-stream.pdf", engine.VerdictPass}, // remapping stream: glyph 0 unconfirmable
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},       // no font renders text
	}
	check := fonts.NotdefGlyph{}
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
