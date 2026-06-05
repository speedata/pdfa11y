package fonts

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// CIDToGIDMap requires every TrueType-based CID font (CIDFontType2
// descendant of a Type 0 composite) to declare /CIDToGIDMap as either
// /Identity or a stream that maps CIDs to glyph IDs. PDF/UA-1 §7.21.4
// (font subset/embedding) leans on this entry so consumers know how
// to find a glyph for a given CID; a non-Identity Name value is not
// defined by ISO 32000-1 §9.7.4.2 and leaves AT-driven extraction
// without a deterministic CID → GID map.
//
// CIDFontType0 descendants (Adobe CFF source) have no /CIDToGIDMap
// at all -- the mapping lives in the embedded CFF charset. They are
// skipped by this check.
//
// /CIDToGIDMap absent on a CIDFontType2 is interpreted as the
// Identity default (ISO 32000-1 §9.7.4.2). The loader collapses
// "absent" into "Identity" so the check only sees the meaningful
// cases.
type CIDToGIDMap struct{}

func (CIDToGIDMap) ID() string                { return "MH-31-001" }
func (CIDToGIDMap) Title() string             { return "CIDFontType2 declares /CIDToGIDMap = Identity or a stream" }
func (CIDToGIDMap) Category() engine.Category { return engine.CategoryFonts }
func (CIDToGIDMap) Severity() engine.Severity { return engine.SeverityError }
func (CIDToGIDMap) Spec() engine.Spec         { return engine.SpecBoth }
func (CIDToGIDMap) WCAG() []string            { return []string{"1.3.1"} }
func (CIDToGIDMap) Description() string {
	return "ISO 32000-1 §9.7.4.2 requires the /CIDToGIDMap entry on a CIDFontType2 to be either /Identity (the default, optionally omitted) or a stream mapping CIDs to glyph IDs. Any other Name value leaves consumers without a defined CID → GID mapping; PDF/UA leans on this entry for assistive-technology text extraction from TrueType-based CID fonts."
}

func (c CIDToGIDMap) Run(doc model.Document) []engine.Finding {
	fonts, err := doc.Fonts()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate fonts: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	considered := 0
	for _, f := range fonts {
		if f.Subtype != "Type0" || f.CIDSubtype != "CIDFontType2" {
			continue
		}
		considered++
		switch f.CIDToGIDMap {
		case "Identity", "Stream":
			// OK
		case "":
			// The loader normalises an absent entry to "Identity";
			// "" here means the descendant was unresolvable. Skip
			// silently rather than emit a misleading finding.
		default:
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("CIDFontType2 %q declares /CIDToGIDMap = %s, expected Identity or a stream", f.BaseFont, f.CIDToGIDMap),
				Hint:     "Set /CIDToGIDMap to /Identity (or omit it entirely; Identity is the default) or provide a stream mapping CIDs to GIDs.",
			})
		}
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no CIDFontType2 descendants -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(CIDToGIDMap{}) }
