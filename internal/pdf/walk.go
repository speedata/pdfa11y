package pdf

import (
	"errors"
	"io"
	"strings"

	pdd "github.com/speedata/pdfdisassembler"
	"github.com/speedata/pdfdisassembler/contentstream"

	"github.com/speedata/pdfa11y/internal/model"
)

// maxUntaggedOpsPerPage caps how many untagged-content findings we
// record per page. One broken page can emit hundreds; the check only
// needs enough samples to point the user at the problem.
const maxUntaggedOpsPerPage = 32

// Pages returns one PageReport per page in document order. The result
// is cached; subsequent calls return the same slice.
func (d *document) Pages() ([]model.PageReport, error) {
	if d.pageReports != nil {
		return d.pageReports, nil
	}
	pages, err := d.collectPages()
	if err != nil {
		return nil, err
	}
	treeMCIDs := d.collectStructTreeMCIDs()
	treeMCIDLang := d.collectStructTreeMCIDLang()
	treeMCIDAlt := d.collectStructTreeMCIDAlt()
	catalogLang := ""
	if cat, err := d.r.Catalog(); err == nil && cat != nil {
		if l, ok := cat.String("Lang"); ok {
			catalogLang = l
		}
	}
	reports := make([]model.PageReport, 0, len(pages))
	for _, p := range pages {
		rep := model.PageReport{
			Number:          p.Number,
			UsedFonts:       map[string]model.Font{},
			ContentMCIDs:    map[int]bool{},
			StructTreeMCIDs: treeMCIDs[p.Ref],
			Tabs:            p.Tabs,
			FontCodes:       map[string]map[uint32]bool{},
			MCIDBoxes:       map[int]model.Rect{},
			MCIDText:        map[int]string{},
		}
		if rep.StructTreeMCIDs == nil {
			rep.StructTreeMCIDs = map[int]bool{}
		}
		if err := d.scanPageContent(p, &rep, treeMCIDLang[p.Ref], catalogLang); err != nil {
			// One broken page should not abort the rest of the report;
			// record what we managed and move on.
			rep.UntaggedOps = append(rep.UntaggedOps, model.UntaggedOp{Operator: "<scan-error>"})
		}
		// Structure-level exemption for PUA real content: drop MCIDs whose
		// owning structure element (or an ancestor) carries /Alt or
		// /ActualText (UA-01-020).
		if structAlt := treeMCIDAlt[p.Ref]; len(rep.PUAContentMCIDs) > 0 && len(structAlt) > 0 {
			for mcid := range rep.PUAContentMCIDs {
				if structAlt[mcid] {
					delete(rep.PUAContentMCIDs, mcid)
				}
			}
		}
		reports = append(reports, rep)
	}
	d.pageReports = reports
	return reports, nil
}

// pageInfo bundles the per-page data we resolve up-front so the
// content-stream scanner does not need to re-walk inheritance.
type pageInfo struct {
	Ref        pdd.Reference
	Number     int
	PageDict   *pdd.Dict
	Fonts      *pdd.Dict // /Resources/Font, possibly inherited; may be nil
	Properties *pdd.Dict // /Resources/Properties, possibly inherited; may be nil
	Tabs       string    // /Tabs value, possibly inherited; "" if absent everywhere
}

// collectPages walks the page tree once and returns one pageInfo per
// leaf Page node, in document order. /Resources is inherited from
// ancestors per PDF 32000-1:2008 §7.7.3.4 (Inheritance of Attributes).
func (d *document) collectPages() ([]pageInfo, error) {
	cat, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, errors.New("no document catalog")
	}
	pagesObj, ok := cat.Get("Pages")
	if !ok {
		return nil, errors.New("catalog has no /Pages")
	}
	ref, ok := pagesObj.(pdd.Reference)
	if !ok {
		return nil, errors.New("/Pages is not an indirect reference")
	}
	var pages []pageInfo
	counter := 0
	d.collectPagesWalk(ref, nil, nil, "", &pages, &counter)
	return pages, nil
}

func (d *document) collectPagesWalk(ref pdd.Reference, inheritedFonts, inheritedProps *pdd.Dict, inheritedTabs string, out *[]pageInfo, counter *int) {
	pageDict, err := d.r.ResolveDict(ref)
	if err != nil || pageDict == nil {
		return
	}
	fonts := inheritedFonts
	props := inheritedProps
	tabs := inheritedTabs
	if res, ok := pageDict.Dict("Resources"); ok && res != nil {
		if f, ok := res.Dict("Font"); ok {
			fonts = f
		}
		if p, ok := res.Dict("Properties"); ok {
			props = p
		}
	}
	if t, ok := pageDict.Name("Tabs"); ok {
		tabs = string(t)
	}
	typeName, _ := pageDict.Name("Type")
	switch typeName {
	case "Page":
		*counter++
		*out = append(*out, pageInfo{
			Ref:        ref,
			Number:     *counter,
			PageDict:   pageDict,
			Fonts:      fonts,
			Properties: props,
			Tabs:       tabs,
		})
	case "Pages":
		kids, ok := pageDict.Array("Kids")
		if !ok {
			return
		}
		for _, kid := range kids {
			kidRef, ok := kid.(pdd.Reference)
			if !ok {
				continue
			}
			d.collectPagesWalk(kidRef, fonts, props, tabs, out, counter)
		}
	}
}

// scanPageContent walks the content stream of a single page and
// populates the report fields. Errors are returned for malformed
// streams; the caller decides whether to bail.
func (d *document) scanPageContent(p pageInfo, rep *model.PageReport, mcidLang map[int]string, catalogLang string) error {
	body, err := d.pageContentBytes(p.PageDict)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	mcDepth := 0
	currentFontKey := ""
	currentFontCodeBytes := 1 // 1-byte default for simple fonts
	// ToUnicode map of the active font, used to decode shown glyphs into
	// readable text for MCIDText. nil when the font has no /ToUnicode.
	var currentFontMappings map[uint32]string

	// Stack of MCIDs active on the marked-content stack. -1 entries are
	// BMC frames or BDC frames without an /MCID property — text inside
	// them is still tagged but contributes to no MCIDBox.
	mcidStack := []int{}

	// Parallel stack classifying each open marked-content frame as an
	// Artifact and/or structure-tagged (MCID) sequence, for detecting the
	// illegal cross-nesting UA-14-010 flags (ISO 14289-1 §7.1).
	mcFrames := []mcFrame{}

	// altDepth counts open marked-content frames carrying an /ActualText or
	// /Alt property; mcidContentAlt records, per MCID, whether such a frame
	// (its own or an ancestor's) covers its content. Used to exempt PUA real
	// content from UA-01-020 at the content-stream level.
	altDepth := 0
	mcidContentAlt := map[int]bool{}

	// Text-state position tracking. We only track translation components
	// (textX/textY for the text matrix, lineX/lineY for the line matrix)
	// and ignore CTM scale/rotation and glyph-advance widths. Pragmatic
	// approximation: good enough for reading-order heuristics that care
	// about which MCID lives where on the page, not pixel-perfect bounds.
	var textX, textY, lineX, lineY float64
	inText := false

	sc := contentstream.New(body)
	for {
		op, err := sc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		switch op.Operator {
		case "BMC":
			mcDepth++
			mcidStack = append(mcidStack, -1)
			tag := markedContentTag(op)
			recordNesting(rep, mcFrames, tag == "Artifact", false, tag, op.Offset)
			mcFrames = append(mcFrames, mcFrame{artifact: tag == "Artifact", mcid: -1})
		case "BDC":
			mcDepth++
			mcid, ok := extractMCID(op, p.Properties)
			if ok {
				rep.ContentMCIDs[mcid] = true
				mcidStack = append(mcidStack, mcid)
			} else {
				mcidStack = append(mcidStack, -1)
			}
			tag := markedContentTag(op)
			recordNesting(rep, mcFrames, tag == "Artifact", ok, tag, op.Offset)
			spanLang := markedContentLang(op, p.Properties)
			frameMCID := -1
			if ok {
				frameMCID = mcid
			}
			if tag == "Span" {
				recordSpanLang(rep, op, p.Properties, mcFrames, spanLang, frameMCID, mcidLang, catalogLang)
			}
			frameHasAlt := bdcPropHas(op, p.Properties, "ActualText") || bdcPropHas(op, p.Properties, "Alt")
			if ok {
				mcidContentAlt[mcid] = altDepth > 0 || frameHasAlt
			}
			mcFrames = append(mcFrames, mcFrame{artifact: tag == "Artifact", tagged: ok, lang: spanLang, mcid: frameMCID, hasAlt: frameHasAlt})
			if frameHasAlt {
				altDepth++
			}
		case "EMC":
			if mcDepth > 0 {
				mcDepth--
			}
			if n := len(mcFrames); n > 0 && mcFrames[n-1].hasAlt && altDepth > 0 {
				altDepth--
			}
			if n := len(mcidStack); n > 0 {
				mcidStack = mcidStack[:n-1]
			}
			if n := len(mcFrames); n > 0 {
				mcFrames = mcFrames[:n-1]
			}
		case "BT":
			inText = true
			textX, textY, lineX, lineY = 0, 0, 0, 0
		case "ET":
			inText = false
		case "Tm":
			if inText && len(op.Operands) >= 6 {
				e := numberOperand(op.Operands[4])
				f := numberOperand(op.Operands[5])
				textX, textY, lineX, lineY = e, f, e, f
			}
		case "Td", "TD":
			if inText && len(op.Operands) >= 2 {
				tx := numberOperand(op.Operands[0])
				ty := numberOperand(op.Operands[1])
				lineX += tx
				lineY += ty
				textX, textY = lineX, lineY
			}
		case "T*":
			// Spec: T* is Td(0, -TL). We do not track TL, so this is a
			// no-op for position tracking. Reading-order checks tolerate
			// this — same-line content stays at the same Y.
		case "Tf":
			if len(op.Operands) >= 1 && op.Operands[0].Kind == contentstream.KindName {
				key := op.Operands[0].Name
				currentFontKey = key
				var f model.Font
				var ok bool
				if f, ok = rep.UsedFonts[key]; !ok {
					f, ok = d.resolveResourceFont(p.Fonts, key)
					if ok {
						rep.UsedFonts[key] = f
					}
				}
				currentFontCodeBytes = codeBytesFor(f, ok)
				currentFontMappings = f.ToUnicodeMappings
			}
		case "Tj", "'", `"`:
			recordTextCodes(rep, currentFontKey, currentFontCodeBytes, textArgBytes(op))
			recordMCIDText(rep, mcidStack, decodeCodes(textArgBytes(op), currentFontCodeBytes, currentFontMappings))
			recordPos(rep, mcidStack, textX, textY)
			recordIfUntagged(rep, op, mcDepth)
		case "TJ":
			recordTextCodes(rep, currentFontKey, currentFontCodeBytes, tjArrayBytes(op))
			recordMCIDText(rep, mcidStack, decodeTJText(op, currentFontCodeBytes, currentFontMappings))
			recordPos(rep, mcidStack, textX, textY)
			recordIfUntagged(rep, op, mcDepth)
		default:
			recordIfUntagged(rep, op, mcDepth)
		}
	}

	// Finalise PUA real-content detection: an MCID whose decoded text carries
	// a Unicode PUA code point and whose marked-content sequence supplies no
	// /ActualText or /Alt is a candidate. The owning structure element may
	// still carry /ActualText or /Alt (resolved later in Pages).
	for mcid, text := range rep.MCIDText {
		if !containsPUARune(text) {
			continue
		}
		rep.HadPUARealContent = true
		if mcidContentAlt[mcid] {
			continue
		}
		if rep.PUAContentMCIDs == nil {
			rep.PUAContentMCIDs = map[int]bool{}
		}
		rep.PUAContentMCIDs[mcid] = true
	}
	return nil
}

// containsPUARune reports whether s contains a Unicode Private Use Area code
// point (U+E000-U+F8FF, U+F0000-U+FFFFD, U+100000-U+10FFFD).
func containsPUARune(s string) bool {
	for _, r := range s {
		switch {
		case r >= 0xE000 && r <= 0xF8FF,
			r >= 0xF0000 && r <= 0xFFFFD,
			r >= 0x100000 && r <= 0x10FFFD:
			return true
		}
	}
	return false
}

// numberOperand returns the numeric value of an Operand, or 0 if it is
// not a number-kind operand. Content-stream operators that take numeric
// arguments are well-defined by the spec; non-number values here would
// be a malformed stream and are silently treated as zero.
func numberOperand(o contentstream.Operand) float64 {
	if o.Kind == contentstream.KindNumber {
		return o.Number
	}
	return 0
}

// recordPos extends the MCIDBox for every MCID currently on the
// marked-content stack so nested tags (e.g. a Span inside a P) both
// see the content. -1 sentinel frames are skipped.
func recordPos(rep *model.PageReport, mcidStack []int, x, y float64) {
	for _, mcid := range mcidStack {
		if mcid < 0 {
			continue
		}
		rep.MCIDBoxes[mcid] = rep.MCIDBoxes[mcid].Extend(x, y)
	}
}

// recordIfUntagged appends the op to rep.UntaggedOps when it is a
// real-content operator running outside any marked-content sequence.
// Capped at maxUntaggedOpsPerPage so a single broken page does not
// flood the report.
func recordIfUntagged(rep *model.PageReport, op contentstream.Op, mcDepth int) {
	if mcDepth != 0 || !isRealContentOp(op.Operator) {
		return
	}
	if len(rep.UntaggedOps) >= maxUntaggedOpsPerPage {
		return
	}
	rep.UntaggedOps = append(rep.UntaggedOps, model.UntaggedOp{
		Operator: op.Operator,
		Offset:   op.Offset,
	})
}

// recordTextCodes splits raw text-show bytes into per-font codes and
// records them in rep.FontCodes. codeBytes is the active font's
// code width (1 for simple fonts and Type0 with single-byte CMaps;
// 2 for Type0 with Identity-H or other two-byte CMaps).
func recordTextCodes(rep *model.PageReport, fontKey string, codeBytes int, raw []byte) {
	if fontKey == "" || len(raw) == 0 {
		return
	}
	if codeBytes < 1 {
		codeBytes = 1
	}
	set, ok := rep.FontCodes[fontKey]
	if !ok {
		set = map[uint32]bool{}
		rep.FontCodes[fontKey] = set
	}
	switch codeBytes {
	case 1:
		for _, b := range raw {
			set[uint32(b)] = true
		}
	case 2:
		for i := 0; i+1 < len(raw); i += 2 {
			set[uint32(raw[i])<<8|uint32(raw[i+1])] = true
		}
	default:
		// >2-byte codes are rare in practice; pack big-endian.
		for i := 0; i+codeBytes <= len(raw); i += codeBytes {
			var v uint32
			for j := 0; j < codeBytes && j < 4; j++ {
				v = v<<8 | uint32(raw[i+j])
			}
			set[v] = true
		}
	}
}

// recordMCIDText appends decoded text to the innermost MCID currently on
// the marked-content stack. Unlike recordPos (which extends boxes for
// every ancestor MCID), text is attributed only to the tightest-enclosing
// tag, so a Span nested in a P does not duplicate its text into the P.
// Text drawn outside any MCID, or whose font yielded no decodable glyphs,
// is dropped.
func recordMCIDText(rep *model.PageReport, mcidStack []int, text string) {
	if text == "" {
		return
	}
	for i := len(mcidStack) - 1; i >= 0; i-- {
		if mcidStack[i] >= 0 {
			rep.MCIDText[mcidStack[i]] += text
			return
		}
	}
}

// decodeCodes maps a run of glyph codes to Unicode text via the font's
// /ToUnicode CMap. codeBytes is the active font's code width (see
// codeBytesFor). Codes with no mapping are skipped -- a partial or
// missing /ToUnicode therefore yields partial or empty text rather than
// mojibake. Returns "" when mappings is nil.
func decodeCodes(raw []byte, codeBytes int, mappings map[uint32]string) string {
	if len(raw) == 0 || mappings == nil {
		return ""
	}
	if codeBytes < 1 {
		codeBytes = 1
	}
	var b strings.Builder
	for i := 0; i+codeBytes <= len(raw); i += codeBytes {
		var code uint32
		for j := 0; j < codeBytes && j < 4; j++ {
			code = code<<8 | uint32(raw[i+j])
		}
		if s, ok := mappings[code]; ok {
			b.WriteString(s)
		}
	}
	return b.String()
}

// decodeTJText decodes a TJ operand array to text. String elements are
// decoded via decodeCodes; the numeric kerning adjustments between them
// are inspected so that a large negative value -- the way producers
// signal an inter-word gap that has no space glyph -- becomes a single
// space. Small adjustments (ordinary kerning) are ignored. The threshold
// is in thousandths of a text-space unit per the TJ operator definition.
func decodeTJText(op contentstream.Op, codeBytes int, mappings map[uint32]string) string {
	if mappings == nil || len(op.Operands) == 0 || op.Operands[0].Kind != contentstream.KindArray {
		return ""
	}
	const wordGapThreshold = 100 // |adjustment| beyond this implies a space
	var b strings.Builder
	for _, el := range op.Operands[0].Array {
		switch el.Kind {
		case contentstream.KindString:
			b.WriteString(decodeCodes(el.Bytes, codeBytes, mappings))
		case contentstream.KindNumber:
			if el.Number <= -wordGapThreshold {
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}

// codeBytesFor picks the right code width for a Tf-referenced font.
//
// Simple fonts (Type1, TrueType, MMType1, Type3) always emit one
// byte per glyph in the Tj/TJ stream.
//
// For Type0 composite fonts the width is set by the /Encoding CMap
// (Identity-H/V and the predefined Adobe CMaps are two-byte; an
// embedded CMap declares its own), carried on the font as
// EncodingCodeBytes. This is deliberately independent of /ToUnicode:
// the extraction CMap's codespace may legally differ, and using it to
// tokenise the content stream manufactures codes that were never
// rendered. Default to 2 for a Type0 whose encoding width is unknown.
//
// resolved=false (unknown font dict) collapses to the 1-byte
// fallback so we never accidentally pack ASCII text as 2-byte CIDs.
func codeBytesFor(f model.Font, resolved bool) int {
	if !resolved {
		return 1
	}
	if f.Subtype != "Type0" {
		return 1
	}
	if f.EncodingCodeBytes > 0 {
		return f.EncodingCodeBytes
	}
	return 2
}

// textArgBytes returns the raw string operand of Tj, ', " operators.
func textArgBytes(op contentstream.Op) []byte {
	if len(op.Operands) == 0 {
		return nil
	}
	// Tj has one string operand; ' has one string; " has two
	// numbers (Tw, Tc) followed by the string.
	last := op.Operands[len(op.Operands)-1]
	if last.Kind == contentstream.KindString {
		return last.Bytes
	}
	return nil
}

// tjArrayBytes concatenates all string elements of the TJ operand
// array, ignoring the kerning numbers.
func tjArrayBytes(op contentstream.Op) []byte {
	if len(op.Operands) == 0 || op.Operands[0].Kind != contentstream.KindArray {
		return nil
	}
	var out []byte
	for _, el := range op.Operands[0].Array {
		if el.Kind == contentstream.KindString {
			out = append(out, el.Bytes...)
		}
	}
	return out
}

// pageContentBytes returns the decoded /Contents bytes for a page,
// joining stream arrays with a single newline separator (PDF
// 32000-1:2008 §7.8.2 allows any whitespace; a newline is unambiguous).
func (d *document) pageContentBytes(pageDict *pdd.Dict) ([]byte, error) {
	cObj, ok := pageDict.Get("Contents")
	if !ok {
		return nil, nil
	}
	switch v := cObj.(type) {
	case pdd.Reference:
		return d.r.DecodeStream(v)
	case pdd.Array:
		var out []byte
		for i, item := range v {
			b, err := d.r.DecodeStream(item)
			if err != nil {
				return nil, err
			}
			if i > 0 {
				out = append(out, '\n')
			}
			out = append(out, b...)
		}
		return out, nil
	}
	return nil, nil
}

// extractMCID pulls the /MCID integer from a BDC properties operand.
// BDC's second operand is either an inline dict (with /MCID directly)
// or a name pointing into /Resources/Properties (whose entry is the
// dict). Returns false when no MCID is present.
func extractMCID(op contentstream.Op, properties *pdd.Dict) (int, bool) {
	if len(op.Operands) < 2 {
		return 0, false
	}
	props := op.Operands[1]
	switch props.Kind {
	case contentstream.KindDict:
		if v, ok := props.Dict["MCID"]; ok {
			if n, ok := v.Int(); ok {
				return int(n), true
			}
		}
	case contentstream.KindName:
		if properties == nil {
			return 0, false
		}
		entry, ok := properties.Dict(props.Name)
		if !ok || entry == nil {
			return 0, false
		}
		if n, ok := entry.Int("MCID"); ok {
			return int(n), true
		}
	}
	return 0, false
}

// mcFrame classifies one open marked-content sequence for §7.1
// cross-nesting detection: whether it is an Artifact and/or a
// structure-tagged (MCID-carrying) sequence.
type mcFrame struct {
	artifact bool
	tagged   bool
	hasAlt   bool   // this sequence's BDC carries /ActualText or /Alt
	lang     string // /Lang property on this sequence's BDC, "" if none
	mcid     int    // MCID of this sequence, or -1
}

// markedContentTag returns the tag-name operand of a BMC/BDC operator
// (its first operand), or "" when absent.
func markedContentTag(op contentstream.Op) string {
	if len(op.Operands) >= 1 && op.Operands[0].Kind == contentstream.KindName {
		return op.Operands[0].Name
	}
	return ""
}

// recordNesting appends an MCNestingViolation when opening a marked-content
// sequence would nest an Artifact and structure-tagged content inside one
// another (ISO 14289-1 §7.1). frames is the stack of currently-open
// ancestor sequences; innerArtifact/innerTagged classify the sequence being
// opened.
func recordNesting(rep *model.PageReport, frames []mcFrame, innerArtifact, innerTagged bool, tag string, offset int64) {
	ancestorArtifact, ancestorTagged := false, false
	for _, f := range frames {
		if f.artifact {
			ancestorArtifact = true
		}
		if f.tagged {
			ancestorTagged = true
		}
	}
	if innerArtifact && ancestorTagged {
		rep.MCNestingViolations = append(rep.MCNestingViolations, model.MCNestingViolation{
			ArtifactInsideTagged: true,
			InnerTag:             tag,
			Offset:               offset,
		})
	}
	if innerTagged && ancestorArtifact {
		rep.MCNestingViolations = append(rep.MCNestingViolations, model.MCNestingViolation{
			ArtifactInsideTagged: false,
			InnerTag:             tag,
			Offset:               offset,
		})
	}
}

// bdcPropHas reports whether a BDC operator's property dictionary carries the
// given key, handling both an inline dictionary operand and a /Properties name
// reference resolved against the page's /Resources/Properties.
func bdcPropHas(op contentstream.Op, properties *pdd.Dict, key string) bool {
	if len(op.Operands) < 2 {
		return false
	}
	switch op.Operands[1].Kind {
	case contentstream.KindDict:
		_, ok := op.Operands[1].Dict[key]
		return ok
	case contentstream.KindName:
		if properties == nil {
			return false
		}
		if entry, ok := properties.Dict(op.Operands[1].Name); ok {
			return entry.Has(key)
		}
	}
	return false
}

// bdcPropString returns a string-valued key from a BDC operator's property
// dictionary (inline or /Properties name reference), or "".
func bdcPropString(op contentstream.Op, properties *pdd.Dict, key string) string {
	if len(op.Operands) < 2 {
		return ""
	}
	switch op.Operands[1].Kind {
	case contentstream.KindDict:
		if v, ok := op.Operands[1].Dict[key]; ok && v.Kind == contentstream.KindString {
			return string(v.Bytes)
		}
	case contentstream.KindName:
		if properties == nil {
			return ""
		}
		if entry, ok := properties.Dict(op.Operands[1].Name); ok {
			if s, ok := entry.String(key); ok {
				return s
			}
		}
	}
	return ""
}

// markedContentLang returns the /Lang string property of a BDC operator's
// property dictionary (inline or resolved through /Properties), or "".
func markedContentLang(op contentstream.Op, properties *pdd.Dict) string {
	return bdcPropString(op, properties, "Lang")
}

// recordSpanLang appends a SpanLangViolation for each /ActualText, /Alt or /E
// property on a Span BDC whose natural language cannot be determined -- no
// /Lang on the Span, no /Lang on any enclosing marked-content sequence, no
// /Lang on the owning structure element (via the nearest enclosing MCID), and
// no catalog /Lang (ISO 14289-1 §7.2, veraPDF UA1:7.2-30/-31/-32).
func recordSpanLang(rep *model.PageReport, op contentstream.Op, properties *pdd.Dict, frames []mcFrame, spanLang string, spanMCID int, mcidLang map[int]string, catalogLang string) {
	hasAT := bdcPropHas(op, properties, "ActualText")
	hasAlt := bdcPropHas(op, properties, "Alt")
	hasE := bdcPropHas(op, properties, "E")
	if !hasAT && !hasAlt && !hasE {
		return
	}
	if langDeterminable(frames, spanLang, spanMCID, mcidLang, catalogLang) {
		return
	}
	for _, attr := range []struct {
		present bool
		name    string
	}{{hasAT, "ActualText"}, {hasAlt, "Alt"}, {hasE, "E"}} {
		if attr.present {
			rep.SpanLangViolations = append(rep.SpanLangViolations, model.SpanLangViolation{
				Attribute: attr.name,
				Offset:    op.Offset,
			})
		}
	}
}

// langDeterminable reports whether a Span's natural language can be
// determined from its own /Lang, an enclosing sequence's /Lang, the owning
// structure element's inherited /Lang (nearest enclosing MCID), or the
// catalog /Lang.
func langDeterminable(frames []mcFrame, spanLang string, spanMCID int, mcidLang map[int]string, catalogLang string) bool {
	if spanLang != "" || catalogLang != "" {
		return true
	}
	for _, f := range frames {
		if f.lang != "" {
			return true
		}
	}
	// Nearest enclosing MCID: the Span's own, else the innermost ancestor's.
	mcid := spanMCID
	if mcid < 0 {
		for i := len(frames) - 1; i >= 0; i-- {
			if frames[i].mcid >= 0 {
				mcid = frames[i].mcid
				break
			}
		}
	}
	if mcid >= 0 {
		if l, ok := mcidLang[mcid]; ok && l != "" {
			return true
		}
	}
	return false
}

// resolveResourceFont resolves a font resource key (e.g. "F1") through
// /Resources/Font to a model.Font snapshot matching what Fonts()
// returns. The lookup is best-effort: missing keys return ok=false.
func (d *document) resolveResourceFont(fontsDict *pdd.Dict, key string) (model.Font, bool) {
	if fontsDict == nil {
		return model.Font{}, false
	}
	obj, ok := fontsDict.Get(key)
	if !ok {
		return model.Font{}, false
	}
	fd, err := d.r.ResolveDict(obj)
	if err != nil || fd == nil {
		return model.Font{}, false
	}
	return d.fontFromDict(fd), true
}

// fontFromDict builds a model.Font snapshot from a resolved font dict.
// Mirrors the logic in Fonts(); kept as a small helper so Pages() does
// not have to re-implement field population.
func (d *document) fontFromDict(fd *pdd.Dict) model.Font {
	subtypeName, _ := fd.Name("Subtype")
	subtype := string(subtypeName)
	baseFont, _ := fd.Name("BaseFont")
	encName, hasDiff := d.fontEncoding(fd)
	hasToU := fd.Has("ToUnicode")
	isSym := d.fontIsSymbolic(fd, subtype)
	mappings, codeBytes := d.parseToUnicodeFromFont(fd)
	cidSubtype, cidToGID := d.cidDescendantInfo(fd, subtype)
	return model.Font{
		Subtype:                subtype,
		BaseFont:               string(baseFont),
		Embedded:               d.fontIsEmbedded(fd, subtype),
		HasToUnicode:           hasToU,
		Encoding:               encName,
		HasEncodingDifferences: hasDiff,
		IsSymbolic:             isSym,
		HasUnicodeMapping:      hasToU || hasDeterministicUnicodeMapping(subtype, encName, hasDiff, isSym),
		ToUnicodeMappings:      mappings,
		ToUnicodeCodeBytes:     codeBytes,
		EncodingCodeBytes:      d.encodingCodeBytes(fd, subtype),
		CIDSubtype:             cidSubtype,
		CIDToGIDMap:            cidToGID,
	}
}

// encodingCodeBytes reports how many bytes make up one code in the
// content stream for this font, as fixed by its /Encoding. Simple
// fonts are one byte. Composite (Type0) fonts take their width from
// the /Encoding CMap: Identity-H/V and the predefined Adobe CMaps are
// two-byte, while an embedded CMap declares its own codespace. This is
// independent of /ToUnicode and is what the walker must use to split
// show strings into codes.
func (d *document) encodingCodeBytes(fd *pdd.Dict, subtype string) int {
	if subtype != "Type0" {
		return 1
	}
	if enc, ok := fd.Get("Encoding"); ok {
		if _, isName := enc.(pdd.Name); !isName {
			// Embedded CMap stream: use its declared codespace width.
			if body, err := d.r.DecodeStream(enc); err == nil {
				if cb := parseToUnicode(body).CodeBytes; cb > 0 {
					return cb
				}
			}
		}
	}
	// Identity-H/V, a predefined CMap, or an unreadable embedded CMap:
	// composite fonts are two-byte.
	return 2
}

// parseToUnicodeFromFont decodes the font's /ToUnicode stream (when
// present) and parses its bfchar / bfrange mappings together with
// the codespace byte width. Returns (nil, 0) for fonts without a
// /ToUnicode entry. Returns an empty (non-nil) map for streams that
// parse but contain no mappings.
func (d *document) parseToUnicodeFromFont(fd *pdd.Dict) (map[uint32]string, int) {
	touObj, ok := fd.Get("ToUnicode")
	if !ok {
		return nil, 0
	}
	body, err := d.r.DecodeStream(touObj)
	if err != nil {
		return nil, 0
	}
	cov := parseToUnicode(body)
	mappings := cov.Mappings
	if mappings == nil {
		mappings = map[uint32]string{}
	}
	return mappings, cov.CodeBytes
}

// isRealContentOp reports whether a content-stream operator paints
// something that participates in the document's semantic content:
// text-showing operators, XObject invocations, path-painting
// operators, and inline images. Path-construction operators (m, l,
// re, c, h, y, v) are not painting on their own, so they are not
// counted.
func isRealContentOp(op string) bool {
	switch op {
	case "Tj", "TJ", "'", `"`:
		return true
	case "Do":
		return true
	case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*":
		return true
	case "EI": // inline image
		return true
	}
	return false
}

// collectStructTreeMCIDs walks the structure tree and aggregates, per
// page reference, the set of MCIDs that the tree claims for that page.
//
// MCIDs are surfaced via two child shapes:
//   - integer K-entry under a structure element: MCID = that integer,
//     page = the element's effective /Pg (its own or inherited from
//     an ancestor).
//   - dict child {/Type /MCR /MCID n /Pg ref}: MCID = n, page = /Pg
//     if present, else the effective inherited /Pg.
func (d *document) collectStructTreeMCIDs() map[pdd.Reference]map[int]bool {
	out := map[pdd.Reference]map[int]bool{}
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return out
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return out
	}
	d.walkStructForMCIDs(stree, pdd.Reference{}, out)
	return out
}

// collectStructTreeMCIDLang mirrors collectStructTreeMCIDs but records, per
// page reference and MCID, the effective /Lang of the owning structure element
// -- its own /Lang or the nearest ancestor's, inherited down the tree. Used by
// UA-11-008 to decide whether a Span marked-content sequence inherits a
// determinable language from the structure element that contains it. Empty
// string when no ancestor declares /Lang.
func (d *document) collectStructTreeMCIDLang() map[pdd.Reference]map[int]string {
	out := map[pdd.Reference]map[int]string{}
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return out
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return out
	}
	d.walkStructForMCIDLang(stree, pdd.Reference{}, "", out)
	return out
}

func (d *document) walkStructForMCIDLang(elem *pdd.Dict, inheritedPage pdd.Reference, inheritedLang string, out map[pdd.Reference]map[int]string) {
	if elem == nil {
		return
	}
	page := inheritedPage
	if pg, ok := elem.Get("Pg"); ok {
		if ref, ok := pg.(pdd.Reference); ok {
			page = ref
		}
	}
	lang := inheritedLang
	if l, ok := elem.String("Lang"); ok && l != "" {
		lang = l
	}
	kObj, ok := elem.Get("K")
	if !ok {
		return
	}
	resolved, err := d.r.Resolve(kObj)
	if err != nil {
		return
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		items = []pdd.Object{v}
	}
	for _, item := range items {
		if n, ok := asInt(item); ok {
			recordMCIDLang(out, page, int(n), lang)
			continue
		}
		child, err := d.r.ResolveDict(item)
		if err != nil || child == nil {
			continue
		}
		tName, _ := child.Name("Type")
		switch string(tName) {
		case "MCR":
			mcid, ok := child.Int("MCID")
			if !ok {
				continue
			}
			ref := page
			if pg, ok := child.Get("Pg"); ok {
				if r, ok := pg.(pdd.Reference); ok {
					ref = r
				}
			}
			recordMCIDLang(out, ref, int(mcid), lang)
		case "OBJR":
			continue
		default:
			d.walkStructForMCIDLang(child, page, lang, out)
		}
	}
}

func recordMCIDLang(out map[pdd.Reference]map[int]string, page pdd.Reference, mcid int, lang string) {
	if page == (pdd.Reference{}) || lang == "" {
		return
	}
	m := out[page]
	if m == nil {
		m = map[int]string{}
		out[page] = m
	}
	m[mcid] = lang
}

// collectStructTreeMCIDAlt records, per page reference and MCID, whether the
// owning structure element or any ancestor carries a non-empty /Alt or
// /ActualText (both inherit down the tree: /ActualText replaces an element's
// content text, /Alt describes its whole subtree). Used by UA-01-020 to exempt
// PUA real content that a structure element already describes.
func (d *document) collectStructTreeMCIDAlt() map[pdd.Reference]map[int]bool {
	out := map[pdd.Reference]map[int]bool{}
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return out
	}
	stree, ok := cat.Dict("StructTreeRoot")
	if !ok {
		return out
	}
	d.walkStructForMCIDAlt(stree, pdd.Reference{}, false, out)
	return out
}

func (d *document) walkStructForMCIDAlt(elem *pdd.Dict, inheritedPage pdd.Reference, inheritedAlt bool, out map[pdd.Reference]map[int]bool) {
	if elem == nil {
		return
	}
	page := inheritedPage
	if pg, ok := elem.Get("Pg"); ok {
		if ref, ok := pg.(pdd.Reference); ok {
			page = ref
		}
	}
	hasAlt := inheritedAlt
	if a, ok := elem.String("Alt"); ok && a != "" {
		hasAlt = true
	}
	if a, ok := elem.String("ActualText"); ok && a != "" {
		hasAlt = true
	}
	kObj, ok := elem.Get("K")
	if !ok {
		return
	}
	resolved, err := d.r.Resolve(kObj)
	if err != nil {
		return
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		items = []pdd.Object{v}
	}
	for _, item := range items {
		if n, ok := asInt(item); ok {
			if hasAlt {
				recordMCIDAlt(out, page, int(n))
			}
			continue
		}
		child, err := d.r.ResolveDict(item)
		if err != nil || child == nil {
			continue
		}
		tName, _ := child.Name("Type")
		switch string(tName) {
		case "MCR":
			mcid, ok := child.Int("MCID")
			if !ok {
				continue
			}
			ref := page
			if pg, ok := child.Get("Pg"); ok {
				if r, ok := pg.(pdd.Reference); ok {
					ref = r
				}
			}
			if hasAlt {
				recordMCIDAlt(out, ref, int(mcid))
			}
		case "OBJR":
			continue
		default:
			d.walkStructForMCIDAlt(child, page, hasAlt, out)
		}
	}
}

func recordMCIDAlt(out map[pdd.Reference]map[int]bool, page pdd.Reference, mcid int) {
	if page == (pdd.Reference{}) {
		return
	}
	m := out[page]
	if m == nil {
		m = map[int]bool{}
		out[page] = m
	}
	m[mcid] = true
}

func (d *document) walkStructForMCIDs(elem *pdd.Dict, inheritedPage pdd.Reference, out map[pdd.Reference]map[int]bool) {
	if elem == nil {
		return
	}
	page := inheritedPage
	if pg, ok := elem.Get("Pg"); ok {
		if ref, ok := pg.(pdd.Reference); ok {
			page = ref
		}
	}
	kObj, ok := elem.Get("K")
	if !ok {
		return
	}
	resolved, err := d.r.Resolve(kObj)
	if err != nil {
		return
	}
	var items []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		items = []pdd.Object(v)
	default:
		items = []pdd.Object{v}
	}
	for _, item := range items {
		d.handleStructKEntry(item, page, out)
	}
}

func (d *document) handleStructKEntry(item pdd.Object, page pdd.Reference, out map[pdd.Reference]map[int]bool) {
	if n, ok := asInt(item); ok {
		// Bare integer: MCID on the current page.
		recordMCID(out, page, int(n))
		return
	}
	child, err := d.r.ResolveDict(item)
	if err != nil || child == nil {
		return
	}
	tName, _ := child.Name("Type")
	switch string(tName) {
	case "MCR":
		mcid, ok := child.Int("MCID")
		if !ok {
			return
		}
		ref := page
		if pg, ok := child.Get("Pg"); ok {
			if r, ok := pg.(pdd.Reference); ok {
				ref = r
			}
		}
		recordMCID(out, ref, int(mcid))
	case "OBJR":
		// Object reference, not an MCID.
		return
	default:
		// Treat as nested structure element.
		d.walkStructForMCIDs(child, page, out)
	}
}

func recordMCID(out map[pdd.Reference]map[int]bool, page pdd.Reference, mcid int) {
	if page == (pdd.Reference{}) {
		return
	}
	m := out[page]
	if m == nil {
		m = map[int]bool{}
		out[page] = m
	}
	m[mcid] = true
}

// asInt accepts the pdd.Integer concrete type and reports its value.
// Defined here so we do not depend on pdd internals beyond the public
// Object types.
func asInt(o pdd.Object) (int64, bool) {
	if n, ok := o.(pdd.Integer); ok {
		return int64(n), true
	}
	return 0, false
}
