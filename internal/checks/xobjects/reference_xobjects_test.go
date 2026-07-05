package xobjects_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/xobjects"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestReferenceXObjects covers UA-30-001: a document must not contain reference
// XObjects (form XObjects with a /Ref entry; ISO 14289-1 §7.20, veraPDF
// UA1:7.20-1).
func TestReferenceXObjects(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/reference-xobject.pdf", engine.VerdictFail},          // form XObject carries /Ref
		{"testdata/plain-xobject.pdf", engine.VerdictPass},              // form XObject without /Ref
		{"../navigation/testdata/outlines-missing.pdf", engine.VerdictNA}, // no form XObjects at all
	}
	check := xobjects.ReferenceXObjects{}
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
