package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestPUARealContent covers UA-01-020: real content that maps (via /ToUnicode)
// to a Unicode Private Use Area value must be covered by an /ActualText or /Alt
// entry, on the marked-content sequence or an enclosing structure element (ISO
// 14289-2 §8.4.3; veraPDF UA2:8.4.3-1).
func TestPUARealContent(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/pua-content-no-alt.pdf", engine.VerdictFail},     // PUA, no ActualText/Alt
		{"testdata/pua-content-actualtext.pdf", engine.VerdictPass}, // PUA + content /ActualText
		{"testdata/pua-content-struct-alt.pdf", engine.VerdictPass}, // PUA + structure /Alt
		{"testdata/pua-content-nonpua.pdf", engine.VerdictNA},       // content maps to a normal glyph
	}
	check := structure.PUARealContent{}
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
