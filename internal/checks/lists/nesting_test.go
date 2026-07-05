package lists_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/lists"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestNesting covers UA-16-004: the list sub-elements must follow the
// canonical parent/child nesting (ISO 14289-1 §7.2; veraPDF UA1:7.2-17..-20
// plus the Caption rule -40).
func TestNesting(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/list-ok.pdf", engine.VerdictPass},           // canonical L -> LI -> Lbl/LBody
		{"testdata/list-no-li.pdf", engine.VerdictFail},        // L has a P child (7.2-19)
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no list elements
	}
	check := lists.Nesting{}
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
