package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestPUAInTextStrings covers UA-01-017: human-readable text strings (form
// field names, structure alternate text, document info, outline titles) must
// not contain Unicode Private Use Area characters (ISO 14289-2 §8.6; veraPDF
// UA2:8.6-1).
func TestPUAInTextStrings(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/puatext-field-pua.pdf", engine.VerdictFail},   // form field /T has PUA
		{"testdata/puatext-field-clean.pdf", engine.VerdictPass}, // form field /T clean
		{"testdata/figure-alt-clean.pdf", engine.VerdictPass},    // structure /Alt, no PUA
		{"testdata/tagged.pdf", engine.VerdictNA},                // no human-readable strings
	}
	check := structure.PUAInTextStrings{}
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
