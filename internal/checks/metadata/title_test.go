package metadata_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/metadata"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestDocumentTitle(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantPass bool
	}{
		{"missing title fails", "testdata/no-title.pdf", false},
		{"DocInfo title passes", "testdata/with-title.pdf", true},
		// XMP dc:title alone is not enough: PDF/UA §7.1 requires the
		// title in DocumentInfo as well, and MH-06-001 verifies that
		// side strictly.
		{"XMP-only title fails (DocInfo strictly required)", "testdata/with-xmp-title.pdf", false},
	}

	check := metadata.DocumentTitle{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}

			if got := result.Passed(); got != tc.wantPass {
				t.Errorf("Passed() = %v, want %v (findings: %+v)", got, tc.wantPass, findings)
			}
			if !tc.wantPass && len(findings) == 0 {
				t.Errorf("expected at least one finding for failing fixture")
			}
		})
	}
}
