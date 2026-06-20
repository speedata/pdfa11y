package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestLinkContents(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantPass     bool
		wantFindings int
	}{
		{"Link with /Contents passes", "testdata/link-with-contents.pdf", true, 0},
		{"Link without /Contents fails", "testdata/link-no-contents.pdf", false, 1},
	}

	check := annotations.LinkContents{}
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

// TestLinkContents_UA2Advisory asserts that under PDF/UA-2 a missing
// /Contents is reported as an advisory (Info), not an error: the
// document still conforms (a missing /Contents is only a
// recommendation there), and even --strict must not promote it.
func TestLinkContents_UA2Advisory(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/link-no-contents-ua2.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := annotations.LinkContents{}
	findings := check.Run(doc)
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1 (%+v)", len(findings), findings)
	}
	if findings[0].Severity != engine.SeverityInfo {
		t.Errorf("Severity = %v, want Info", findings[0].Severity)
	}

	results := []engine.Result{{Check: check, Findings: findings}}
	if !engine.Summarize(results).Conforming() {
		t.Errorf("UA-2 document with missing /Contents should still conform")
	}
	engine.PromoteWarnings(results)
	if findings[0].Severity != engine.SeverityInfo {
		t.Errorf("Severity after PromoteWarnings = %v, want Info (advisories survive --strict)", findings[0].Severity)
	}
}

// TestLinkContents_NoLinks asserts the check reports N/A on a document
// without any Link annotations rather than vacuously passing.
func TestLinkContents_NoLinks(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/widget-with-tu.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := annotations.LinkContents{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
