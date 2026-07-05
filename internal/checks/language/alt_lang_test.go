package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestAltLang covers UA-11-004: /Alt, /ActualText and /E must have a
// determinable language (ISO 14289-1 §7.2; veraPDF UA1:7.2-21/-22/-23).
func TestAltLang(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/figure-alt-lang.pdf", engine.VerdictPass},             // Figure /Alt with own /Lang
		{"../graphics/testdata/figure-with-alt.pdf", engine.VerdictFail}, // /Alt, no /Lang anywhere
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},           // no Alt/ActualText/E
	}
	check := language.AltLang{}
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
