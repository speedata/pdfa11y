package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestPUAInAltActualText covers UA-01-013: /Alt and /ActualText must not
// contain Unicode Private Use Area characters (PDF/UA-2 §8.4.3-2/-3).
func TestPUAInAltActualText(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/figure-alt-pua.pdf", engine.VerdictFail},   // Alt contains U+E000
		{"testdata/figure-alt-clean.pdf", engine.VerdictPass}, // Alt is plain text
		{"testdata/tagged.pdf", engine.VerdictNA},             // no Alt/ActualText
	}
	check := structure.PUAInAltActualText{}
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
