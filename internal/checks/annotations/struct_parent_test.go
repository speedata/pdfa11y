package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestStructLinked(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Text annotation with /StructParent passes", "testdata/annot-with-structparent.pdf", true, 0},
		{"Text annotation without /StructParent fails", "testdata/annot-no-structparent.pdf", false, 1},
	}

	check := annotations.StructLinked{}
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

// TestStructLinked_WatermarkSkipped asserts artifact subtypes (Watermark,
// PrinterMark, TrapNet, Popup) are not counted as structure-tree-eligible
// and therefore drive the check to N/A on a document whose only
// annotation is a Watermark.
func TestStructLinked_WatermarkSkipped(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/watermark-no-structparent.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := annotations.StructLinked{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
