package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestArtifactSubtypes(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Watermark without /StructParent passes", "testdata/watermark-no-structparent.pdf", true, 0},
		{"Watermark with /StructParent fails", "testdata/watermark-with-structparent.pdf", false, 1},
	}

	check := annotations.ArtifactSubtypes{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.Passed(); got != tc.wantPass {
				t.Fatalf("Passed() = %v, want %v (findings: %+v)", got, tc.wantPass, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
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
