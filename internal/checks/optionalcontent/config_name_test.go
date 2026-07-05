package optionalcontent_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/optionalcontent"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestConfigName covers UA-20-003: optional-content configurations must
// declare a non-empty /Name -- unconditionally for PDF/UA-1 §7.10-1, and for
// PDF/UA-2 §8.7-1 only when a /Configs array is present (veraPDF UA1:7.10-1 /
// UA2:8.7-1).
func TestConfigName(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/oc-ua1-unnamed.pdf", engine.VerdictFail},         // UA-1: /D lacks /Name (unconditional)
		{"testdata/oc-ua2-noconfigs-unnamed.pdf", engine.VerdictNA}, // UA-2: no /Configs -> exempt
		{"testdata/oc-ua2-configs-unnamed.pdf", engine.VerdictFail}, // UA-2: /Configs entry lacks /Name
		{"testdata/oc-ua2-configs-named.pdf", engine.VerdictPass},   // UA-2: all named
		{"../structure/testdata/tagged.pdf", engine.VerdictNA},      // no /OCProperties
	}
	check := optionalcontent.ConfigName{}
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
