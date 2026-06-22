package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestArtifactSubtypes_WatermarkNotFlagged guards the fix that removed
// Watermark from the pure-artifact set: ISO 14289-2 §8.9.2.4.16 allows
// a Watermark "used as real content", so a Watermark with /StructParent
// is conformant. Neither watermark fixture is a pure-artifact subtype
// (PrinterMark/TrapNet), so the check declines (N/A) on both.
func TestArtifactSubtypes_WatermarkNotFlagged(t *testing.T) {
	fixtures := []string{
		"testdata/watermark-no-structparent.pdf",
		"testdata/watermark-with-structparent.pdf",
	}
	check := annotations.ArtifactSubtypes{}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if result.State() != engine.VerdictNA {
				t.Fatalf("State() = %v, want N/A (Watermark must not be flagged) (findings: %+v)", result.State(), result.Findings)
			}
		})
	}
}

// TestArtifactSubtypes_NoArtifacts asserts N/A on a document whose
// annotations are all "real content" subtypes (Link, Widget, Text).
func TestArtifactSubtypes_NoArtifacts(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/link-with-contents.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := annotations.ArtifactSubtypes{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
