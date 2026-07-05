package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestAnnotLang covers UA-11-005: an annotation's /Contents must have a
// determinable language (ISO 14289-1 §7.2; veraPDF UA1:7.2-24).
func TestAnnotLang(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/annot-contents-lang.pdf", engine.VerdictPass},               // Contents + own /Lang
		{"../annotations/testdata/link-with-contents.pdf", engine.VerdictFail}, // Contents, no /Lang
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},                 // no annotation with /Contents
	}
	check := language.AnnotLang{}
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
