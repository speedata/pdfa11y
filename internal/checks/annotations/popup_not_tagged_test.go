package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestPopupNotTagged covers UA-28-017: a Popup annotation must not be present
// in the structure tree (ISO 14289-2 §8.9.2.4.9; veraPDF UA2:8.9.2.4.9-1).
func TestPopupNotTagged(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/popup-in-p.pdf", engine.VerdictFail},        // Popup enclosed in a P element
		{"testdata/popup-untagged.pdf", engine.VerdictPass},    // Popup present but not tagged
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no Popup annotations
	}
	check := annotations.PopupNotTagged{}
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
