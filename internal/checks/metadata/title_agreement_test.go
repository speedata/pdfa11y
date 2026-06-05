package metadata_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/metadata"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestTitleAgreement(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"matching titles pass", "testdata/title-agreement-ok.pdf", true, 0},
		{"diverging titles fail", "testdata/title-agreement-mismatch.pdf", false, 1},
	}

	check := metadata.TitleAgreement{}
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

// TestTitleAgreement_DeclinesWithoutOneSide covers the two N/A paths:
// a document with only XMP dc:title (no DocumentInfo /Title) and one
// with neither title at all should both decline rather than fail.
func TestTitleAgreement_DeclinesWithoutOneSide(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{"only XMP dc:title", "testdata/with-xmp-title.pdf"},
		{"no title at all", "testdata/no-title.pdf"},
	}
	check := metadata.TitleAgreement{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if result.State() != engine.VerdictNA {
				t.Errorf("State() = %v, want N/A (findings: %+v)", result.State(), result.Findings)
			}
		})
	}
}
