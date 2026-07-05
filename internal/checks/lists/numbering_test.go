package lists_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/lists"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// Numbering produces a warning rather than an error, so the absence of
// /ListNumbering still leaves Passed() == true; the test instead asserts
// the resulting Verdict (Pass vs Warn) plus the per-finding severity.
func TestNumbering(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantVerdict  engine.Verdict
		wantFindings int
		wantSeverity engine.Severity
	}{
		{"L with /ListNumbering passes", "testdata/list-with-numbering.pdf", engine.VerdictPass, 0, 0},
		{"L without /ListNumbering warns", "testdata/list-no-numbering.pdf", engine.VerdictWarn, 1, engine.SeverityWarning},
		// A list carrying Lbl children but no /ListNumbering is still only
		// a Warning: whether the list is *ordered* (so §8.2.5.25's "shall"
		// bites) is not machine-decidable from the tree -- unordered
		// bullet lists legitimately omit the attribute. See Numbering's
		// doc comment and the pdfa.org L_07 / L_09 reference cases.
		{"L with Lbl but no /ListNumbering warns", "testdata/list-no-numbering-with-lbl.pdf", engine.VerdictWarn, 1, engine.SeverityWarning},
	}

	check := lists.Numbering{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantVerdict {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.wantVerdict, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Fatalf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
			if tc.wantFindings > 0 && findings[0].Severity != tc.wantSeverity {
				t.Errorf("severity = %v, want %v", findings[0].Severity, tc.wantSeverity)
			}
		})
	}
}

// TestNumbering_NoLists asserts N/A on a document whose structure tree
// contains no L at all.
func TestNumbering_NoLists(t *testing.T) {
	doc, err := pdf.LoadFile("../tables/testdata/table-row-cells.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := lists.Numbering{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
