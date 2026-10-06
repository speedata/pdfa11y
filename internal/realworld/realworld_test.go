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
//   - UA-15-003: no Table element present.
func TestGluPDFUADemo(t *testing.T) {
	knownLimitations := map[string]string{
		// The glu demo predates ISO 14289-2 §5's pdfuaid:rev
		// requirement: the file declares pdfuaid:part but not
		// pdfuaid:rev. Tolerated until the demo regenerates.
		"UA-06-006": "fixture predates the ISO 14289-2 §5 pdfuaid:rev requirement",
		// The demo declares pdfuaid:part 1: it is a PDF/UA-1 file, and
		// its outline items legitimately use page destinations. Structure
		// destinations (ISO 14289-2 §8.8) are a PDF/UA-2-only requirement;
		// this test runs every check regardless of the document's declared
		// spec, so the UA-2-only rule fires vacuously here.
		"UA-27-003": "PDF/UA-1 document; structure destinations are a PDF/UA-2-only requirement",
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

// TestGluPDFUA2Namespaces runs the PDF/UA-2 check set against a glu
// document rendered with --format PDF/UA-2. glu writes most structure
// elements in the XHTML namespace (h1, p, li, table, ...) and maps
// them to PDF 2.0 types via the namespace's /RoleMapNS, while some
// (L, LBody) sit directly in the PDF 2.0 namespace.
//
// Besides "no failures", the test asserts that the list and table
// checks actually ran: before StructElement.Type() resolved
// /RoleMapNS, the XHTML elements kept their raw names, so the table
// checks found no Table (vacuous N/A) and the list checks rejected
// the li children of L.
func TestGluPDFUA2Namespaces(t *testing.T) {
	doc, err := pdf.LoadFile("testdata/glu-pdfua2-namespaces.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	mustApply := map[string]bool{
		"UA-15-003": true, "UA-15-004": true, "UA-15-006": true,
		"UA-16-001": true, "UA-16-002": true, "UA-16-004": true,
	}
	for _, r := range engine.Run(doc, engine.ForSpec(engine.SpecPDFUA2)) {
		if !r.Passed() {
			t.Errorf("%s (%s) failed unexpectedly on a conforming document\n  findings: %+v",
				r.Check.ID(), r.Check.Title(), r.Findings)
			continue
		}
		if mustApply[r.Check.ID()] && r.State() == engine.VerdictNA {
			t.Errorf("%s (%s) is N/A: namespaced structure types were not resolved\n  findings: %+v",
				r.Check.ID(), r.Check.Title(), r.Findings)
		}
	}
}
