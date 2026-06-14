package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestXFAForbidden(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"PDF/UA-2 with /AcroForm/XFA fails",
			"testdata/xfa-in-ua2.pdf",
			engine.VerdictFail,
		},
		{
			// acroform-linked.pdf is a PDF/UA-1 fixture with no XFA.
			"PDF/UA-1 document is N/A",
			"testdata/acroform-linked.pdf",
			engine.VerdictNA,
		},
	}
	check := annotations.XFAForbidden{}
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
