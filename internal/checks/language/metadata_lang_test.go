package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestMetadataLang covers UA-11-007: XMP metadata with an x-default language
// alternative must have a determinable language via a catalog /Lang (ISO
// 14289-1 §7.2; veraPDF UA1:7.2-33).
func TestMetadataLang(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/metadata-no-lang.pdf", engine.VerdictFail},      // x-default, no catalog /Lang
		{"testdata/metadata-with-lang.pdf", engine.VerdictPass},    // x-default + catalog /Lang
		{"testdata/metadata-no-xdefault.pdf", engine.VerdictNA},    // no x-default alternative
	}
	check := language.MetadataLang{}
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
