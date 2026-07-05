package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestSignatureGraphicAlt covers UA-28-036: a graphic (Figure) that is part of
// a signature field's appearance must carry /Alt (ISO 14289-2 §8.10.3.5;
// veraPDF UA2:8.10.3.5-1).
func TestSignatureGraphicAlt(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/sig-figure-no-alt.pdf", engine.VerdictFail},   // signature Figure without /Alt
		{"testdata/sig-figure-with-alt.pdf", engine.VerdictPass}, // signature Figure with /Alt
		{"testdata/sig-no-figure.pdf", engine.VerdictNA},         // signature with no graphic
	}
	check := annotations.SignatureGraphicAlt{}
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
