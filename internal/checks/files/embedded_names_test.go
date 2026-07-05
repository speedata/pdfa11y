package files_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/files"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestEmbeddedNames covers UA-12-003: an embedded file's specification must
// carry non-empty /F and /UF keys (ISO 14289-1 §7.11; veraPDF UA1:7.11-1).
func TestEmbeddedNames(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/embedded-desc.pdf", engine.VerdictPass},     // /F and /UF present
		{"testdata/embedded-no-names.pdf", engine.VerdictFail}, // /EF but no /F or /UF
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no embedded files
	}
	check := files.EmbeddedNames{}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.want {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.want, result.Findings)
			}
		})
	}
}
