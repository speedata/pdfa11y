package lists_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/lists"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestItemBody(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"LI with LBody passes", "testdata/list-li-with-lbody.pdf", true, 0},
		{"LI without LBody fails", "testdata/list-li-no-lbody.pdf", false, 1},
	}

	check := lists.ItemBody{}
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

// TestItemBody_NoLI asserts N/A on a document whose structure tree
// contains no LI at all.
func TestItemBody_NoLI(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/list-no-li.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := lists.ItemBody{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
