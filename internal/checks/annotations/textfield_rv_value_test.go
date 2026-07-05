package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestTextFieldRVValue covers UA-28-033: a text field with a rich-text value
// /RV must also carry a plain /V that is textually equivalent (ISO 14289-2
// §8.10.3.3; veraPDF UA2:8.10.3.3-1).
func TestTextFieldRVValue(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/rv-no-value.pdf", engine.VerdictFail},       // /RV present, no /V
		{"testdata/rv-value-mismatch.pdf", engine.VerdictFail}, // /RV and /V differ
		{"testdata/rv-value-match.pdf", engine.VerdictPass},    // /RV plain text equals /V
		{"testdata/rv-absent.pdf", engine.VerdictNA},           // text field without /RV
	}
	check := annotations.TextFieldRVValue{}
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
