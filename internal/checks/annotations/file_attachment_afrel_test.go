package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestFileAttachmentAFRel covers UA-28-024: a file attachment annotation's
// filespec must declare an /AFRelationship (ISO 14289-2 §8.9.2.4.10; veraPDF
// UA2:8.9.2.4.10-1).
func TestFileAttachmentAFRel(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/fileattach-afrel.pdf", engine.VerdictPass},
		{"testdata/fileattach-no-afrel.pdf", engine.VerdictFail},
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no file attachment annotations
	}
	check := annotations.FileAttachmentAFRel{}
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
