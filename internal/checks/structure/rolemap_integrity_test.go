package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestRoleMapIntegrity covers UA-31-009: role maps must be acyclic (and, for
// namespaces, must not remap within the same namespace -- exercised by the
// veraPDF corpus oracle). These fixtures cover the classic /RoleMap path.
func TestRoleMapIntegrity(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/rolemap-cycle.pdf", engine.VerdictFail},  // Foo<->Bar cycle
		{"testdata/rolemap-acyclic.pdf", engine.VerdictPass}, // Foo->P, terminates
		{"testdata/tagged.pdf", engine.VerdictNA},            // no role map
	}
	check := structure.RoleMapIntegrity{}
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
