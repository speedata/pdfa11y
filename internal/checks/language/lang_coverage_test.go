package language_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/language"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestLangCoverage(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"P with /Lang passes", "testdata/lang-coverage-ok.pdf", true, 0},
		{"P without /Lang and no Catalog /Lang fails", "testdata/lang-coverage-missing.pdf", false, 1},
	}

	check := language.LangCoverage{}
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

// TestLangCoverage_CatalogLang asserts the check declines (N/A) on
// any document whose Catalog declares /Lang -- the per-element
// branch is only meaningful when the Catalog form is absent.
func TestLangCoverage_CatalogLang(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/with-lang.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := language.LangCoverage{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}

// TestLangCoverage_NoStructTree asserts the check declines on an
// untagged document. MH-01-005 covers the missing-struct-tree
// failure mode; here we only want N/A.
func TestLangCoverage_NoStructTree(t *testing.T) {
	doc, err := pdf.LoadFile("../structure/testdata/untagged.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := language.LangCoverage{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
