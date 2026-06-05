package fonts_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/fonts"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestCIDToGIDMap(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"/CIDToGIDMap = Identity passes", "testdata/cid-identity.pdf", true, 0},
		{"/CIDToGIDMap = arbitrary Name fails", "testdata/cid-bad-name.pdf", false, 1},
	}

	check := fonts.CIDToGIDMap{}
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

// TestCIDToGIDMap_NoCIDFonts asserts N/A on a document whose fonts
// are all simple (Type1/TrueType) -- no CIDFontType2 descendant.
func TestCIDToGIDMap_NoCIDFonts(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/font-ok.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := fonts.CIDToGIDMap{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
