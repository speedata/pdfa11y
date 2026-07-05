package xobjects_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/xobjects"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestUniqueSemanticParent covers UA-30-002: a form XObject carrying marked
// content (an /MCID) must be invoked by Do at most once, so its MCIDs resolve
// to a unique semantic parent (ISO 14289-1 §7.20; veraPDF UA1:7.20-2).
func TestUniqueSemanticParent(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/xobj-mc-once.pdf", engine.VerdictPass},                 // marked content, drawn once
		{"testdata/xobj-mc-twice.pdf", engine.VerdictFail},                // marked content, drawn twice
		{"testdata/xobj-plain-twice.pdf", engine.VerdictPass},             // no marked content, drawn twice
		{"testdata/plain-xobject.pdf", engine.VerdictPass},                // no marked content, not drawn
		{"../navigation/testdata/outlines-missing.pdf", engine.VerdictNA}, // no form XObjects at all
	}
	check := xobjects.UniqueSemanticParent{}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.want {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.want, result.Findings)
			}
		})
	}
}
