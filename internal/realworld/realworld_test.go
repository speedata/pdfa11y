// Package realworld_test runs every registered check against real
// PDF/UA-conforming documents and asserts that all checks pass. It
// guards against false positives slipping into individual checks: a
// regression that breaks one of them in a way the synthetic fixtures
// don't catch will trip the assertion here on a document that is known
// to be PDF/UA-conforming.
//
// Add more fixtures here as the corpus grows; document expected
// vacuous passes (where the PDF lacks an element the check targets)
// so future readers know which signals are real vs trivial.
package realworld_test

import (
	"testing"

	_ "github.com/speedata/pdfa11y/internal/checks" // register every check
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestGluPDFUADemo runs the full check set against a tagged document
// produced by speedata's glu (Markdown → PDF/UA pipeline). It is
// PDF/UA-1 conforming and structurally rich: H1/H2 hierarchy, a list
// with four LI/LBody items, a Figure with /Alt, five Type0 embedded
// fonts with /ToUnicode CMaps, XMP with pdfuaid:part and dc:title,
// Catalog /Lang, ViewerPreferences/DisplayDocTitle, MarkInfo/Marked,
// /Tabs entry on every page.
//
// Vacuous passes (document does not exercise the check, so it
// passes by absence of a violation):
//   - MH-15-003: no Table element present.
func TestGluPDFUADemo(t *testing.T) {
	knownLimitations := map[string]string{
		// The glu demo predates ISO 14289-2 §5's pdfuaid:rev
		// requirement: the file declares pdfuaid:part but not
		// pdfuaid:rev. Tolerated until the demo regenerates.
		"MH-06-006": "fixture predates the ISO 14289-2 §5 pdfuaid:rev requirement",
	}
	doc, err := pdf.LoadFile("testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	for _, r := range engine.Run(doc, engine.All()) {
		if r.Passed() {
			continue
		}
		if reason, ok := knownLimitations[r.Check.ID()]; ok {
			t.Logf("%s tolerated on glu-pdfua-demo (%s): %+v", r.Check.ID(), reason, r.Findings)
			continue
		}
		t.Errorf("%s (%s) failed unexpectedly on a conforming document\n  findings: %+v",
			r.Check.ID(), r.Check.Title(), r.Findings)
	}
}
