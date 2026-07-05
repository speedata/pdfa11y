package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestInvisibleArtifact covers UA-28-020: an invisible or no-view annotation
// must be an artifact, not tagged content (ISO 14289-2 §8.9.2.2; veraPDF
// UA2:8.9.2.2-1 and -2).
func TestInvisibleArtifact(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/annot-invisible-tagged.pdf", engine.VerdictFail},   // Invisible annotation in a P element
		{"testdata/annot-invisible-untagged.pdf", engine.VerdictPass}, // Invisible annotation, not in the tree
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},        // no invisible/no-view annotations
	}
	check := annotations.InvisibleArtifact{}
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
