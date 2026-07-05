package structure_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestRubyStructure covers UA-01-017: a Ruby element must have children RB, RT
// or RB, RP, RT, RP (ISO 14289-2 §8.2.5.23; veraPDF UA2:8.2.5.23-1).
func TestRubyStructure(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/ruby-ok.pdf", engine.VerdictPass},  // RB, RT
		{"testdata/ruby-bad.pdf", engine.VerdictFail}, // RB only
		{"testdata/tagged.pdf", engine.VerdictNA},     // no Ruby elements
	}
	check := structure.RubyStructure{}
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

// TestWarichuStructure covers UA-01-018: a Warichu element must have children
// WP, WT, WP (ISO 14289-2 §8.2.5.24; veraPDF UA2:8.2.5.24-1).
func TestWarichuStructure(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/warichu-ok.pdf", engine.VerdictPass},  // WP, WT, WP
		{"testdata/warichu-bad.pdf", engine.VerdictFail}, // WT, WP
		{"testdata/tagged.pdf", engine.VerdictNA},        // no Warichu elements
	}
	check := structure.WarichuStructure{}
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
