package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestLinkEnclosureUA2 covers UA-28-016: a structure-tree Link annotation must
// be enclosed in a Link or Reference element (ISO 14289-2 §8.2.5.20; veraPDF
// UA2:8.2.5.20-1).
func TestLinkEnclosureUA2(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/link-tagged.pdf", engine.VerdictPass},       // Link in a Link element
		{"testdata/link-in-reference.pdf", engine.VerdictPass}, // Link in a Reference element
		{"testdata/link-in-p.pdf", engine.VerdictFail},         // Link in a P element
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no Link annotations
	}
	check := annotations.LinkEnclosureUA2{}
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
