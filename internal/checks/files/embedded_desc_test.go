package files_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/files"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestEmbeddedDesc covers UA-12-002: every file specification in the
// /Names /EmbeddedFiles name tree must declare /Desc (ISO 14289-2 §8.14.1;
// veraPDF UA2:8.14.1-1).
func TestEmbeddedDesc(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/embedded-desc.pdf", engine.VerdictPass},     // filespec has /Desc
		{"testdata/embedded-no-desc.pdf", engine.VerdictFail},  // filespec without /Desc
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no embedded files
	}
	check := files.EmbeddedDesc{}
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
