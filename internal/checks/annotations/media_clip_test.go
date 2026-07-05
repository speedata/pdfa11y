package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestMediaClipCT covers UA-28-022: a media clip data dictionary must carry a
// /CT entry (ISO 14289-1 §7.18.6.2; veraPDF UA1:7.18.6.2-1).
func TestMediaClipCT(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/mediaclip-ok.pdf", engine.VerdictPass},
		{"testdata/mediaclip-no-ct.pdf", engine.VerdictFail},
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no media clips
	}
	check := annotations.MediaClipCT{}
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

// TestMediaClipAlt covers UA-28-023: a media clip data dictionary must carry a
// default /Alt description (ISO 14289-1 §7.18.6.2; veraPDF UA1:7.18.6.2-2).
func TestMediaClipAlt(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/mediaclip-ok.pdf", engine.VerdictPass},
		{"testdata/mediaclip-no-alt.pdf", engine.VerdictFail},
		{"../structure/testdata/tagged.pdf", engine.VerdictNA}, // no media clips
	}
	check := annotations.MediaClipAlt{}
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
