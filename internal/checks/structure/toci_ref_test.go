package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestTOCIRef covers UA-01-012: a TOCI must identify its target via /Ref.
func TestTOCIRef(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/toci-with-ref.pdf", engine.VerdictPass}, // TOCI with resolvable /Ref
		{"testdata/toci-no-ref.pdf", engine.VerdictFail},   // TOCI without /Ref
		{"testdata/tagged.pdf", engine.VerdictNA},          // no TOCI element
	}
	check := structure.TOCIRef{}
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
