package taborder_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/taborder"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestPageTabs(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"/Tabs = S passes", "testdata/tabs-s.pdf", true, 0},
		{"/Tabs = R fails", "testdata/tabs-r.pdf", false, 1},
	}

	check := taborder.PageTabs{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.Passed(); got != tc.wantPass {
				t.Fatalf("Passed() = %v, want %v (findings: %+v)", got, tc.wantPass, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
		})
	}
}

// TestPageTabs_Missing asserts a page without /Tabs (and no inherited
// /Tabs) fails, matching the empty-string branch in the check.
func TestPageTabs_Missing(t *testing.T) {
	doc, err := pdf.LoadFile("../metadata/testdata/no-title.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := taborder.PageTabs{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.Passed() {
		t.Fatalf("Passed() = true on a page lacking /Tabs; want false (findings: %+v)", result.Findings)
	}
	if len(result.Findings) != 1 {
		t.Errorf("findings = %d, want 1 (%+v)", len(result.Findings), result.Findings)
	}
}
