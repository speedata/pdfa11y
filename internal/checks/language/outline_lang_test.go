package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestOutlineLang covers UA-11-006: a document with bookmarks must declare a
// catalog /Lang so the outline language is determinable (ISO 14289-1 §7.2;
// veraPDF UA1:7.2-2).
func TestOutlineLang(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/outline-no-lang.pdf", engine.VerdictFail},  // bookmarks, no catalog /Lang
		{"testdata/outline-with-lang.pdf", engine.VerdictPass}, // bookmarks + catalog /Lang
		{"testdata/no-outline.pdf", engine.VerdictNA},          // no bookmarks
	}
	check := language.OutlineLang{}
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
