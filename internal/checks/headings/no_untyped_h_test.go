package headings_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/headings"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestNoUntypedH(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{
			"PDF/UA-2 with untyped H fails",
			"testdata/heading-untyped-h-ua2.pdf",
			engine.VerdictFail,
		},
		{
			// heading-ok.pdf uses explicit H1 in PDF/UA-1.
			"PDF/UA-1 document is N/A",
			"testdata/heading-ok.pdf",
			engine.VerdictNA,
		},
	}
	check := headings.NoUntypedH{}
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
