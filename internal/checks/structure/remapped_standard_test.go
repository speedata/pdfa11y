package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestRemappedStandardType covers UA-31-010: standard structure types must not
// be role-mapped (PDF/UA-1 §7.1-7 / PDF/UA-2 §8.2.4-4).
func TestRemappedStandardType(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/rolemap-remaps-standard.pdf", engine.VerdictFail}, // H1 -> Foo
		{"testdata/rolemap-acyclic.pdf", engine.VerdictPass},         // Foo -> P (Foo is custom)
		{"testdata/tagged.pdf", engine.VerdictNA},                    // no role map
	}
	check := structure.RemappedStandardType{}
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
