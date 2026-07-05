package navigation_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/navigation"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestTOCNesting covers UA-27-002: the table-of-contents sub-elements must
// follow the canonical parent/child nesting (ISO 14289-1 §7.2; veraPDF
// UA1:7.2-26/-27 plus the Caption rule -28).
func TestTOCNesting(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/toc-bad-child.pdf", engine.VerdictFail},              // TOC has a P child (7.2-27)
		{"../structure/testdata/toci-with-ref.pdf", engine.VerdictPass}, // canonical TOC -> TOCI
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},          // no TOC elements
	}
	check := navigation.TOCNesting{}
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
