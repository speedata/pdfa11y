package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestStructTreeRoot(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantPass bool
	}{
		{"untagged document fails", "testdata/untagged.pdf", false},
		{"tagged document passes", "testdata/tagged.pdf", true},
	}

	check := structure.StructTreeRoot{}
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
			if !tc.wantPass && findings[0].Severity != engine.SeverityError {
				t.Errorf("severity = %v, want error", findings[0].Severity)
			}
		})
	}
}
