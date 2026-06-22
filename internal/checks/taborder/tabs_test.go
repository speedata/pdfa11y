package taborder_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/taborder"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestPageTabs(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantState engine.Verdict
	}{
		{"/Tabs = S passes", "testdata/tabs-s.pdf", engine.VerdictPass},
		{"/Tabs = R fails", "testdata/tabs-r.pdf", engine.VerdictFail},
		// PDF/UA-2 §8.9.3.3 widens the allowed set to S, A, W.
		{"PDF/UA-2 /Tabs = A passes", "testdata/tabs-a-ua2.pdf", engine.VerdictPass},
		{"PDF/UA-2 /Tabs = W passes", "testdata/tabs-w-ua2.pdf", engine.VerdictPass},
		{"PDF/UA-2 /Tabs = R still fails", "testdata/tabs-r-ua2.pdf", engine.VerdictFail},
		// The requirement binds only to pages with an annotation: a bad
		// /Tabs on an annotation-free page is N/A, not a failure.
		{"bad /Tabs but no annotation is N/A", "testdata/tabs-no-annot.pdf", engine.VerdictNA},
	}

	check := taborder.PageTabs{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.wantState {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.wantState, result.Findings)
			}
		})
	}
}

// TestPageTabs_Missing asserts a page that carries an annotation but
// has no /Tabs (nor an inherited one) fails.
func TestPageTabs_Missing(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/tabs-missing.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := taborder.PageTabs{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictFail {
		t.Fatalf("State() = %v on a page lacking /Tabs; want FAIL (findings: %+v)", result.State(), result.Findings)
	}
	if len(result.Findings) != 1 {
		t.Errorf("findings = %d, want 1 (%+v)", len(result.Findings), result.Findings)
	}
}
