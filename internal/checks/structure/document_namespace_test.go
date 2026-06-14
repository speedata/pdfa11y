package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestDocumentNamespace(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"Document in PDF 2.0 namespace passes",
			"testdata/document-ns-pdf2.pdf",
			engine.VerdictPass,
		},
		{
			"Document in PDF 1.7 namespace fails",
			"testdata/document-ns-pdf17.pdf",
			engine.VerdictFail,
		},
		{
			// tagged.pdf is a PDF/UA-1 fixture; spec gating yields N/A.
			"PDF/UA-1 document is N/A",
			"testdata/tagged.pdf",
			engine.VerdictNA,
		},
	}
	check := structure.DocumentNamespace{}
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
