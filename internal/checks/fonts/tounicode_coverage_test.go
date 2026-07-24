package fonts_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/fonts"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestToUnicodeCoverageEncodingWidth is the issue #1 regression: UA-10-002
// must tokenise Type0 show strings by the font's /Encoding width, not by
// the /ToUnicode codespace. All three fixtures render identical two-byte
// Identity-H codes and differ only in /ToUnicode, so a checker that keyed
// the code width off /ToUnicode would mis-split the show strings and flag
// codes that were never rendered.
func TestToUnicodeCoverageEncodingWidth(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
		why     string
	}{
		{"testdata/cid-encoding-width-ok.pdf", engine.VerdictPass,
			"2-byte /ToUnicode codespace matches Identity-H"},
		{"testdata/cid-encoding-width-mismatch.pdf", engine.VerdictPass,
			"1-byte /ToUnicode codespace must not manufacture a phantom 0x0000"},
		{"testdata/cid-uncovered-rendered-code.pdf", engine.VerdictFail,
			"a genuinely unmapped rendered code is still reported"},
	}
	check := fonts.ToUnicodeCoverage{}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.want {
				t.Fatalf("State() = %v, want %v (%s; findings: %+v)",
					got, tc.want, tc.why, result.Findings)
			}
		})
	}
}
