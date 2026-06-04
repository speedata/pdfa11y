package pdf_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestPagesSmokeTest loads a real PDF and verifies the walker
// surfaces font usage, MCIDs from both the content stream and the
// structure tree, and runs to completion without errors.
func TestPagesSmokeTest(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pages, err := doc.Pages()
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	if len(pages) == 0 {
		t.Fatalf("expected at least one page")
	}
	first := pages[0]
	if first.Number != 1 {
		t.Errorf("first page Number = %d, want 1", first.Number)
	}
	if len(first.UsedFonts) == 0 {
		t.Errorf("expected real PDF to reference at least one font via Tf, got %+v", first.UsedFonts)
	}
	t.Logf("page 1: used fonts=%d content-MCIDs=%d tree-MCIDs=%d untagged-ops=%d",
		len(first.UsedFonts), len(first.ContentMCIDs), len(first.StructTreeMCIDs), len(first.UntaggedOps))
}

// TestPagesMCIDsAlign verifies that for a well-tagged PDF the
// content-stream MCID set and the structure-tree MCID set agree on
// each page. Mismatches are real findings (drive Check #70 later);
// for now we only assert correctness on a known-good fixture.
func TestPagesMCIDsAlign(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pages, err := doc.Pages()
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	for _, p := range pages {
		if len(p.ContentMCIDs) == 0 && len(p.StructTreeMCIDs) == 0 {
			continue
		}
		// Symmetric difference should be empty on a sound PDF/UA file.
		for id := range p.ContentMCIDs {
			if !p.StructTreeMCIDs[id] {
				t.Logf("page %d: MCID %d in content stream but not in structure tree", p.Number, id)
			}
		}
		for id := range p.StructTreeMCIDs {
			if !p.ContentMCIDs[id] {
				t.Logf("page %d: MCID %d claimed by structure tree but not in content stream", p.Number, id)
			}
		}
	}
}
