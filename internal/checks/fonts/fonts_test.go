package fonts_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/fonts"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// All three fixtures share the canonical base PDF, which embeds four
// real fonts of its own (three lacking ToUnicode). The tests therefore
// assert against the *synthesised* font by its distinct BaseFont name
// rather than against overall finding counts, which would be brittle
// to changes in the base PDF.

func TestEmbedded(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		ourFontName    string
		wantOurFinding bool
	}{
		{"synthesised embedded font passes", "testdata/font-ok.pdf", "PDFA11YTestOK", false},
		{"synthesised non-embedded font fails", "testdata/font-not-embedded.pdf", "PDFA11YTestNoEmbed", true},
	}

	check := fonts.Embedded{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			got := mentions(findings, tc.ourFontName)
			if got != tc.wantOurFinding {
				t.Errorf("finding for %s: got=%v want=%v (all findings: %v)",
					tc.ourFontName, got, tc.wantOurFinding, findings)
			}
		})
	}
}

func TestToUnicode(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		ourFontName    string
		wantOurFinding bool
	}{
		{"synthesised font with /ToUnicode passes", "testdata/font-ok.pdf", "PDFA11YTestOK", false},
		{"synthesised font without /ToUnicode fails", "testdata/font-no-tounicode.pdf", "PDFA11YTestNoToU", true},
	}

	check := fonts.ToUnicode{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			got := mentions(findings, tc.ourFontName)
			if got != tc.wantOurFinding {
				t.Errorf("finding for %s: got=%v want=%v (all findings: %v)",
					tc.ourFontName, got, tc.wantOurFinding, findings)
			}
		})
	}
}

func mentions(findings []engine.Finding, fontName string) bool {
	for _, f := range findings {
		if strings.Contains(f.Message, fontName) {
			return true
		}
	}
	return false
}
