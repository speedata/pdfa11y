package metadata_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/metadata"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestPDFUARevision(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"pdfuaid:part with pdfuaid:rev passes",
			"testdata/with-pdfua-id-rev.pdf",
			engine.VerdictPass,
		},
		{
			"pdfuaid:part without pdfuaid:rev fails",
			"testdata/with-pdfua-id.pdf",
			engine.VerdictFail,
		},
		{
			"no XMP metadata is N/A (MH-06-003 already handles it)",
			"testdata/no-pdfua-id.pdf",
			engine.VerdictNA,
		},
	}
	check := metadata.PDFUARevision{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantState {
				t.Errorf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
		})
	}
}
