package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestArtifactNesting covers UA-14-010: an Artifact and structure-tagged
// marked content must not nest inside one another, in either direction (ISO
// 14289-1 §7.1; veraPDF UA1:7.1-1 and UA1:7.1-2).
func TestArtifactNesting(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/mc-artifact-in-tagged.pdf", engine.VerdictFail}, // Artifact inside tagged (7.1-1)
		{"testdata/mc-tagged-in-artifact.pdf", engine.VerdictFail}, // tagged inside Artifact (7.1-2)
		{"testdata/mc-nesting-ok.pdf", engine.VerdictPass},         // Artifact and tagged as siblings
		{"testdata/tagged.pdf", engine.VerdictNA},                  // no page content activity
	}
	check := structure.ArtifactNesting{}
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
