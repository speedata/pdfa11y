package fonts_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/fonts"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestToUnicodeValues covers UA-10-003: /ToUnicode values must be valid
// Unicode (> U+0000, not U+FEFF/U+FFFE) -- ISO 14289-1 §7.21.7 /
// ISO 14289-2 §8.4.5.8; veraPDF UA1:7.21.7-2 / UA2:8.4.5.8-2.
func TestToUnicodeValues(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/font-ok.pdf", engine.VerdictPass},             // valid /ToUnicode
		{"testdata/font-tounicode-null.pdf", engine.VerdictFail}, // maps a code to U+0000
		{"testdata/font-no-tounicode.pdf", engine.VerdictNA},     // no /ToUnicode CMap
	}
	check := fonts.ToUnicodeValues{}
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
