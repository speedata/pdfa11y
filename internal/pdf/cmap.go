package pdf

import "bytes"

// cmapCoverage is the subset of a ToUnicode CMap needed for
// UA-10-002: the set of source codes for which the CMap actually
// provides a mapping (regardless of the mapped Unicode value's
// quality). Empty when the CMap declared only a codespace range.
type cmapCoverage struct {
	// Mappings maps each source code to its Unicode replacement.
	Mappings map[uint32]string
	// Empty is true when the CMap parses successfully but contains
	// no bfchar / bfrange entries -- the F01 scenario.
	Empty bool
	// CodeBytes is the source-code width declared by the CMap's
	// begincodespacerange. 1 when the codespace was <hh> <hh>, 2
	// when <hhhh> <hhhh>. Zero when no codespace was parsed (then
	// callers fall back to the font-subtype default).
	CodeBytes int
}

// parseToUnicode walks a PDF ToUnicode CMap stream and returns the
// codes for which a mapping exists. The CMap is a tiny stack-based
// PostScript-like language; we only handle the
// beginbfchar/endbfchar and beginbfrange/endbfrange operators, which
// cover the vast majority of producer output. begincidchar /
// begincidrange (CID-keyed) are different operators and are not used
// in /ToUnicode CMaps.
func parseToUnicode(stream []byte) cmapCoverage {
	out := cmapCoverage{Mappings: map[uint32]string{}, Empty: true}
	i := 0
	for i < len(stream) {
		i = skipWS(stream, i)
		if i >= len(stream) {
			break
		}
		switch {
		case hasWord(stream, i, "beginbfchar"):
			i += len("beginbfchar")
			i = parseBfChar(stream, i, &out)
		case hasWord(stream, i, "beginbfrange"):
			i += len("beginbfrange")
			i = parseBfRange(stream, i, &out)
		case hasWord(stream, i, "begincodespacerange"):
			i += len("begincodespacerange")
			i = parseCodespace(stream, i, &out)
		default:
			// Skip to next whitespace-delimited token.
			i = skipToken(stream, i)
		}
	}
	if len(out.Mappings) > 0 {
		out.Empty = false
	}
	return out
}

// parseCodespace records the byte-width of the first codespace range
// it encounters. Multiple ranges are allowed by the CMap spec; for
// UA-10-002 we only need the width, and producers almost always
// declare a single range. Reads up to "endcodespacerange".
func parseCodespace(src []byte, i int, out *cmapCoverage) int {
	for {
		i = skipWS(src, i)
		if hasWord(src, i, "endcodespacerange") {
			return i + len("endcodespacerange")
		}
		lo, ni, ok := readHexString(src, i)
		if !ok {
			return skipToEnd(src, i, "endcodespacerange")
		}
		i = ni
		i = skipWS(src, i)
		_, ni, ok = readHexString(src, i)
		if !ok {
			return skipToEnd(src, i, "endcodespacerange")
		}
		i = ni
		if out.CodeBytes == 0 && len(lo) > 0 {
			out.CodeBytes = len(lo)
		}
	}
}

// parseBfChar reads a sequence of <code> <unicode> pairs until
// "endbfchar". Both code and unicode are hex strings.
func parseBfChar(src []byte, i int, out *cmapCoverage) int {
	for {
		i = skipWS(src, i)
		if hasWord(src, i, "endbfchar") {
			return i + len("endbfchar")
		}
		code, ni, ok := readHexString(src, i)
		if !ok {
			return skipToEnd(src, i, "endbfchar")
		}
		i = ni
		i = skipWS(src, i)
		uni, ni, ok := readHexString(src, i)
		if !ok {
			return skipToEnd(src, i, "endbfchar")
		}
		i = ni
		out.Mappings[hexToCode(code)] = utf16BEToString(uni)
	}
}

// parseBfRange reads sequences of <lo> <hi> <unicode-start> triples
// (range with sequential mapping) or <lo> <hi> [<u1> <u2> ...]
// (range with per-code mapping array) until "endbfrange".
func parseBfRange(src []byte, i int, out *cmapCoverage) int {
	for {
		i = skipWS(src, i)
		if hasWord(src, i, "endbfrange") {
			return i + len("endbfrange")
		}
		lo, ni, ok := readHexString(src, i)
		if !ok {
			return skipToEnd(src, i, "endbfrange")
		}
		i = ni
		i = skipWS(src, i)
		hi, ni, ok := readHexString(src, i)
		if !ok {
			return skipToEnd(src, i, "endbfrange")
		}
		i = ni
		loCode := hexToCode(lo)
		hiCode := hexToCode(hi)
		i = skipWS(src, i)
		if i < len(src) && src[i] == '[' {
			// Per-code array form: lo, hi, [<u_lo>, <u_lo+1>, ...]
			i++
			j := uint32(0)
			for {
				i = skipWS(src, i)
				if i < len(src) && src[i] == ']' {
					i++
					break
				}
				uni, ni, ok := readHexString(src, i)
				if !ok {
					return skipToEnd(src, i, "endbfrange")
				}
				i = ni
				if loCode+j <= hiCode {
					out.Mappings[loCode+j] = utf16BEToString(uni)
				}
				j++
			}
		} else {
			// Sequential form: lo, hi, <u_lo>; codes get +1 each.
			uni, ni, ok := readHexString(src, i)
			if !ok {
				return skipToEnd(src, i, "endbfrange")
			}
			i = ni
			base := utf16BEToCodepoints(uni)
			for c := loCode; c <= hiCode; c++ {
				offset := c - loCode
				cps := make([]rune, len(base))
				copy(cps, base)
				if len(cps) > 0 {
					cps[len(cps)-1] += rune(offset)
				}
				out.Mappings[c] = string(cps)
			}
		}
	}
}

// hexToCode converts a hex-decoded byte slice to a big-endian
// uint32. Codes longer than 4 bytes are truncated to the high four.
func hexToCode(b []byte) uint32 {
	var v uint32
	for i := 0; i < len(b) && i < 4; i++ {
		v = v<<8 | uint32(b[i])
	}
	return v
}

// utf16BEToString decodes a UTF-16 big-endian byte sequence to a Go
// string. Used both for the per-code Unicode value and for the
// stream's encoded run.
func utf16BEToString(b []byte) string {
	return string(utf16BEToCodepoints(b))
}

func utf16BEToCodepoints(b []byte) []rune {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	var out []rune
	for i := 0; i+1 < len(b); i += 2 {
		hi := uint16(b[i])<<8 | uint16(b[i+1])
		if hi >= 0xD800 && hi <= 0xDBFF && i+3 < len(b) {
			// Surrogate pair.
			lo := uint16(b[i+2])<<8 | uint16(b[i+3])
			if lo >= 0xDC00 && lo <= 0xDFFF {
				cp := 0x10000 + (uint32(hi-0xD800) << 10) + uint32(lo-0xDC00)
				out = append(out, rune(cp))
				i += 2
				continue
			}
		}
		out = append(out, rune(hi))
	}
	return out
}

// readHexString reads "<...>" (a CMap hex string) starting at i and
// returns the decoded bytes plus the next position. ok=false when
// the string is not a hex string or is malformed.
func readHexString(src []byte, i int) (decoded []byte, next int, ok bool) {
	if i >= len(src) || src[i] != '<' {
		return nil, i, false
	}
	end := bytes.IndexByte(src[i+1:], '>')
	if end < 0 {
		return nil, i, false
	}
	body := src[i+1 : i+1+end]
	// Drop whitespace, then decode hex pairs. Odd-length bodies pad
	// with a trailing 0 nibble per PDF spec.
	clean := make([]byte, 0, len(body))
	for _, c := range body {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		clean = append(clean, c)
	}
	if len(clean)%2 != 0 {
		clean = append(clean, '0')
	}
	out := make([]byte, len(clean)/2)
	for j := 0; j < len(out); j++ {
		hi, ok1 := hexNibble(clean[2*j])
		lo, ok2 := hexNibble(clean[2*j+1])
		if !ok1 || !ok2 {
			return nil, i, false
		}
		out[j] = hi<<4 | lo
	}
	return out, i + 1 + end + 1, true
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func skipWS(src []byte, i int) int {
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
			i++
			continue
		}
		break
	}
	return i
}

func skipToken(src []byte, i int) int {
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
			return i
		}
		i++
	}
	return i
}

// hasWord reports whether src starting at i matches word followed
// by a non-name character (whitespace, '<', or end of input).
func hasWord(src []byte, i int, word string) bool {
	if i+len(word) > len(src) {
		return false
	}
	if string(src[i:i+len(word)]) != word {
		return false
	}
	if i+len(word) == len(src) {
		return true
	}
	c := src[i+len(word)]
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '<'
}

// skipToEnd recovers from a parse error inside a beginbfchar/range
// block by skipping forward to the matching endXxx keyword. Keeps
// the rest of the CMap parseable.
func skipToEnd(src []byte, i int, marker string) int {
	idx := bytes.Index(src[i:], []byte(marker))
	if idx < 0 {
		return len(src)
	}
	return i + idx + len(marker)
}
