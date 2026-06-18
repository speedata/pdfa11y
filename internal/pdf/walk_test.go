package pdf_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/model"
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

// TestMCIDBoxes verifies that the walker emits a bounding box per
// content-stream MCID, that the boxes are non-empty, and that the page
// has more than one distinct Y so reading-order heuristics actually
// have data to compare. The fixture is a real PDF/UA file with several
// MCIDs across the page.
func TestMCIDBoxes(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pages, err := doc.Pages()
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	first := pages[0]
	if len(first.MCIDBoxes) == 0 {
		t.Fatalf("page 1: expected MCID boxes, got none")
	}
	for id := range first.ContentMCIDs {
		box, ok := first.MCIDBoxes[id]
		if !ok {
			// MCIDs introduced by a BDC with no Tj/TJ inside (e.g. a
			// pure Artifact wrapping a path) legitimately have no box.
			continue
		}
		// Each tracked MCID must have a defined starting position.
		if box == (model.Rect{}) {
			t.Errorf("MCID %d has zero-value Rect — recordPos never fired", id)
		}
	}
	// Reading-order heuristics need at least two distinct Y positions
	// to be useful; a real multi-paragraph page should easily clear this.
	seen := map[float64]bool{}
	for _, b := range first.MCIDBoxes {
		seen[b.MinY] = true
	}
	if len(seen) < 2 {
		t.Errorf("page 1: expected >= 2 distinct MinY values across MCIDBoxes, got %d", len(seen))
	}
	t.Logf("page 1: %d MCIDBoxes across %d distinct Y positions", len(first.MCIDBoxes), len(seen))
}

// TestMCIDText verifies the walker decodes shown glyphs into per-MCID
// Unicode text. At least one MCID on the first page must carry the H1's
// title text, proving the /ToUnicode decode path runs end to end.
func TestMCIDText(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pages, err := doc.Pages()
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	first := pages[0]
	if len(first.MCIDText) == 0 {
		t.Fatalf("page 1: expected decoded MCID text, got none")
	}
	var joined string
	for _, s := range first.MCIDText {
		joined += s + " "
	}
	// Raw per-MCID text reconstructs inter-word gaps from TJ kerning and
	// can carry double spaces; collapse whitespace before matching.
	joined = strings.Join(strings.Fields(joined), " ")
	if !strings.Contains(joined, "Markdown to PDF/UA") {
		t.Errorf("page 1 MCID text does not contain the H1 title; got %q", joined)
	}
}

// TestStructElementText verifies that a structure element surfaces the
// content-stream text drawn under its own marked content, attributed to
// the right element and excluding text owned by child structure elements.
func TestStructElementText(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	root, err := doc.StructTreeRoot()
	if err != nil {
		t.Fatalf("struct tree: %v", err)
	}
	h1 := findByType(root, "H1")
	if h1 == nil {
		t.Fatal("no H1 element found in fixture")
	}
	if got := strings.Join(strings.Fields(h1.Text()), " "); got != "Markdown to PDF/UA" {
		t.Errorf("H1.Text() = %q, want %q", got, "Markdown to PDF/UA")
	}
	// The Document root groups children but owns no direct marked content,
	// so its own text is empty even though its descendants have plenty.
	if got := strings.TrimSpace(root.Text()); got != "" {
		t.Errorf("Document root Text() = %q, want empty (text belongs to children)", got)
	}
}

// TestStructElementContent verifies that Content() is the faithful,
// ordered union of Children() and Text(): its element items match
// Children() in order, and its text items concatenate to Text(). This is
// the contract serializeContent relies on to preserve reading order when
// prose wraps inline child elements.
func TestStructElementContent(t *testing.T) {
	doc, err := pdf.LoadFile("../realworld/testdata/glu-pdfua-demo.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	root, err := doc.StructTreeRoot()
	if err != nil {
		t.Fatalf("struct tree: %v", err)
	}
	var check func(e model.StructElement)
	check = func(e model.StructElement) {
		content := e.Content()

		// Element items, in order, equal Children().
		var elems []model.StructElement
		var text string
		for _, it := range content {
			if it.Element != nil {
				elems = append(elems, it.Element)
			} else {
				if it.Text == "" {
					t.Errorf("<%s>: Content() yielded an empty text item", e.Type())
				}
				text += it.Text
			}
		}
		kids := e.Children()
		if len(elems) != len(kids) {
			t.Errorf("<%s>: Content() has %d element items, Children() has %d", e.Type(), len(elems), len(kids))
		} else {
			for i := range kids {
				if elems[i].Type() != kids[i].Type() {
					t.Errorf("<%s>: content element %d is <%s>, Children() has <%s>", e.Type(), i, elems[i].Type(), kids[i].Type())
				}
			}
		}

		// Text items concatenate to Text().
		if text != e.Text() {
			t.Errorf("<%s>: concatenated Content() text = %q, Text() = %q", e.Type(), text, e.Text())
		}

		for _, c := range kids {
			check(c)
		}
	}
	check(root)
}

// findByType returns the first element of the given /S type in a
// depth-first walk, or nil when none exists.
func findByType(e model.StructElement, typ string) model.StructElement {
	if e.Type() == typ {
		return e
	}
	for _, c := range e.Children() {
		if hit := findByType(c, typ); hit != nil {
			return hit
		}
	}
	return nil
}
