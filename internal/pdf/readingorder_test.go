package pdf_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestStructTreeOrder verifies the linearisation runs end-to-end on a
// real PDF/UA fixture: emits leaves, page numbers stay monotonic and
// valid, every MCID is one the page actually announced, and the
// struct-path chain is non-empty for leaves below the root.
func TestStructTreeOrder(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	order, err := doc.StructTreeOrder()
	if err != nil {
		t.Fatalf("StructTreeOrder: %v", err)
	}
	if len(order) == 0 {
		t.Fatalf("expected leaves on a tagged PDF, got none")
	}

	pages, err := doc.Pages()
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	pageMCIDs := make(map[int]map[int]bool, len(pages))
	for _, p := range pages {
		pageMCIDs[p.Number] = p.ContentMCIDs
	}

	lastPage := 0
	for i, e := range order {
		if e.Page == 0 {
			// Allowed (un-locatable leaf) but worth surfacing.
			t.Logf("entry %d (%s, MCID %d) has Page=0 — unresolved /Pg", i, e.StructPath, e.MCID)
			continue
		}
		if e.Page < lastPage {
			t.Errorf("entry %d: page %d < previous page %d (DFS over /Pg should be monotonic for well-tagged PDFs)", i, e.Page, lastPage)
		}
		lastPage = e.Page
		if mcids := pageMCIDs[e.Page]; mcids != nil && !mcids[e.MCID] {
			t.Errorf("entry %d: page %d MCID %d not in ContentMCIDs", i, e.Page, e.MCID)
		}
		if e.StructPath == "" {
			t.Errorf("entry %d: empty StructPath", i)
		}
	}
	t.Logf("linearised %d MCID leaves across %d page(s); first=%+v last=%+v",
		len(order), len(pages), order[0], order[len(order)-1])
}
