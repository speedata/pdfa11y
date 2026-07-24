package pdf

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/model"
)

// TestCodeBytesForUsesEncodingNotToUnicode locks the rule that the
// content-stream code width comes from the font's /Encoding, not from
// /ToUnicode. A Type0/Identity-H font renders two-byte codes even when
// its /ToUnicode CMap declares a one-byte codespace; taking the width
// from /ToUnicode there mis-splits the show strings and manufactures
// codes that were never rendered (issue #1).
func TestCodeBytesForUsesEncodingNotToUnicode(t *testing.T) {
	cases := []struct {
		name     string
		font     model.Font
		resolved bool
		want     int
	}{
		// Identity-H: two-byte encoding, but /ToUnicode declares a
		// one-byte codespace. Must follow the encoding, not /ToUnicode.
		{"type0 identity-h vs 1-byte tounicode",
			model.Font{Subtype: "Type0", EncodingCodeBytes: 2, ToUnicodeCodeBytes: 1}, true, 2},
		// One-byte embedded CMap, two-byte /ToUnicode: follow the encoding.
		{"type0 one-byte encoding vs 2-byte tounicode",
			model.Font{Subtype: "Type0", EncodingCodeBytes: 1, ToUnicodeCodeBytes: 2}, true, 1},
		// Unknown encoding width: composite fonts default to two bytes.
		{"type0 unknown encoding defaults two",
			model.Font{Subtype: "Type0", EncodingCodeBytes: 0, ToUnicodeCodeBytes: 1}, true, 2},
		// Simple fonts are always one byte, whatever /ToUnicode says.
		{"simple truetype is one byte",
			model.Font{Subtype: "TrueType", EncodingCodeBytes: 0, ToUnicodeCodeBytes: 2}, true, 1},
		// Unresolved font dict falls back to one byte.
		{"unresolved falls back to one",
			model.Font{Subtype: "Type0", EncodingCodeBytes: 2}, false, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codeBytesFor(tc.font, tc.resolved); got != tc.want {
				t.Errorf("codeBytesFor() = %d, want %d", got, tc.want)
			}
		})
	}
}
