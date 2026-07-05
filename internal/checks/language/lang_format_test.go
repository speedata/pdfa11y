package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestLangFormat covers UA-11-003: declared /Lang values must be well-formed
// RFC 3066 language tags.
func TestLangFormat(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/with-lang.pdf", engine.VerdictPass},            // Catalog /Lang "en-US"
		{"testdata/lang-format-invalid.pdf", engine.VerdictFail},  // Catalog /Lang "portugues-pt" (9-letter primary subtag)
		{"testdata/no-lang.pdf", engine.VerdictNA},                // no /Lang anywhere
	}
	check := language.LangFormat{}
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
