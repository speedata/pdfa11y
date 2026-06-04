package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestCatalogLang(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantFindings int
		wantSeverity engine.Severity
	}{
		{"missing /Lang yields a warning", "testdata/no-lang.pdf", 1, engine.SeverityWarning},
		{"present /Lang yields no findings", "testdata/with-lang.pdf", 0, 0},
	}

	check := language.CatalogLang{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings: got %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
			if tc.wantFindings > 0 && findings[0].Severity != tc.wantSeverity {
				t.Errorf("severity: got %v, want %v", findings[0].Severity, tc.wantSeverity)
			}
		})
	}
}

// TestCatalogLang_DoesNotBreakConformance documents the deliberate severity
// choice: a missing Catalog /Lang is a warning, not an error, so it must not
// drop the document out of conformance on its own.
func TestCatalogLang_DoesNotBreakConformance(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/no-lang.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := language.CatalogLang{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if !result.Passed() {
		t.Errorf("Passed() = false, want true (warnings must not fail the check)")
	}
}
