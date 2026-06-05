package files_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/files"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestAFRelationship(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"filespec with /AFRelationship passes", "testdata/af-with-relationship.pdf", true, 0},
		{"filespec without /AFRelationship fails", "testdata/af-no-relationship.pdf", false, 1},
	}

	check := files.AFRelationship{}
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

// TestAFRelationship_NoAF asserts N/A on a document with no /AF
// arrays anywhere (the overwhelmingly common case).
func TestAFRelationship_NoAF(t *testing.T) {
	doc, err := pdf.LoadFile("../metadata/testdata/no-title.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := files.AFRelationship{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
