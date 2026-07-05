package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestFieldTULang covers UA-11-009: a form field's /TU must have a determinable
// natural language, from the /Lang on the field's own structure element or the
// catalog /Lang (ISO 14289-1 §7.2; veraPDF UA1:7.2-25).
func TestFieldTULang(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/tu-no-lang.pdf", engine.VerdictFail},      // /TU, no lang anywhere
		{"testdata/tu-struct-lang.pdf", engine.VerdictPass},  // /Lang on the field structure element
		{"testdata/tu-catalog-lang.pdf", engine.VerdictPass}, // catalog /Lang
		{"testdata/tu-absent.pdf", engine.VerdictNA},         // no /TU
	}
	check := language.FieldTULang{}
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
