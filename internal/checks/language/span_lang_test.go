package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestSpanLang covers UA-11-008: a Span marked-content sequence carrying
// /ActualText, /Alt or /E must have a determinable natural language, from the
// Span's own /Lang, an inherited structure /Lang, or the catalog /Lang (ISO
// 14289-1 §7.2; veraPDF UA1:7.2-30/-31/-32).
func TestSpanLang(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/span-actualtext-no-lang.pdf", engine.VerdictFail}, // ActualText, no lang (7.2-30)
		{"testdata/span-alt-no-lang.pdf", engine.VerdictFail},        // Alt, no lang (7.2-31)
		{"testdata/span-e-no-lang.pdf", engine.VerdictFail},          // E, no lang (7.2-32)
		{"testdata/span-span-lang.pdf", engine.VerdictPass},          // /Lang on the Span
		{"testdata/span-catalog-lang.pdf", engine.VerdictPass},       // catalog /Lang
		{"testdata/span-struct-lang.pdf", engine.VerdictPass},        // inherited structure /Lang
	}
	check := language.SpanLang{}
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
