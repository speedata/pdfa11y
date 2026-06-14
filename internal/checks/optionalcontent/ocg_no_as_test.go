package optionalcontent_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/optionalcontent"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestOCGNoAS(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"PDF/UA-2 with /OCProperties/D/AS fails",
			"testdata/ocg-with-as-ua2.pdf",
			engine.VerdictFail,
		},
		{
			// ocg-with-name.pdf is a PDF/UA-1 fixture with OCGs but
			// no /AS; UA-2 gating yields N/A.
			"PDF/UA-1 document is N/A",
			"testdata/ocg-with-name.pdf",
			engine.VerdictNA,
		},
	}
	check := optionalcontent.OCGNoAS{}
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
