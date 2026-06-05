package fonts_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/fonts"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestType1Removed(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantPass bool
		wantHits int
	}{
		{"TrueType under UA-2 passes", "testdata/truetype-in-ua2.pdf", true, 0},
		{"Type1 under UA-2 fails", "testdata/type1-in-ua2.pdf", false, 1},
	}

	check := fonts.Type1Removed{}
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
			// Error-level findings only -- ignore N/A entries the check
			// might add when the path is exercised differently.
			errors := 0
			for _, f := range findings {
				if f.Severity == engine.SeverityError {
					errors++
				}
			}
			if errors != tc.wantHits {
				t.Fatalf("error findings = %d, want %d (%+v)", errors, tc.wantHits, findings)
			}
		})
	}
}

// TestType1Removed_NotUA2 asserts the check declines on a PDF that
// does not declare pdfuaid:part = 2, even when Type 1 fonts are
// present. The fonts fixture without UA metadata stands in for "any
// non-UA-2 document".
func TestType1Removed_NotUA2(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/font-ok.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := fonts.Type1Removed{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
