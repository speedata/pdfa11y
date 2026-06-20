package model

// standardStructureTypes is the union of the standard PDF structure
// element types defined by ISO 32000-1 (PDF 1.7) and ISO 32000-2
// (PDF 2.0), plus the PDF/UA-2 additions. It spans the grouping,
// block, inline, illustration and table categories.
//
// This is the single source of truth shared by two layers:
//   - the role-resolution in internal/pdf, which must not rewrite an
//     element that already declares a standard type in the PDF
//     standard namespace (a global /RoleMap that maps such a type is
//     only a PDF 1.7 compatibility hint), and
//   - UA-31-008 (internal/checks/structure), which flags custom types
//     that survive role-map resolution without becoming standard.
var standardStructureTypes = map[string]struct{}{
	// Grouping elements
	"Document":   {},
	"Part":       {},
	"Art":        {},
	"Sect":       {},
	"Div":        {},
	"BlockQuote": {},
	"Caption":    {},
	"TOC":        {},
	"TOCI":       {},
	"Index":      {},
	"NonStruct":  {},
	"Private":    {},
	// PDF 2.0 grouping additions
	"DocumentFragment": {},
	"Aside":            {},
	"Title":            {},
	// Block-level
	"P":     {},
	"H":     {},
	"H1":    {},
	"H2":    {},
	"H3":    {},
	"H4":    {},
	"H5":    {},
	"H6":    {},
	"Hn":    {}, // PDF 2.0 generic heading (level via /Lvl)
	"L":     {},
	"LI":    {},
	"Lbl":   {},
	"LBody": {},
	// Inline-level
	"Span":      {},
	"Quote":     {},
	"Note":      {},
	"FENote":    {}, // PDF 2.0 footnote/endnote; replaces Note in UA-2
	"Reference": {},
	"BibEntry":  {},
	"Code":      {},
	"Link":      {},
	"Annot":     {},
	"Ruby":      {},
	"RB":        {},
	"RT":        {},
	"RP":        {},
	"Warichu":   {},
	"WT":        {},
	"WP":        {},
	// PDF 2.0 inline additions
	"Em":     {},
	"Strong": {},
	"Sub":    {},
	// Illustration
	"Figure":  {},
	"Formula": {},
	"Form":    {},
	// Tables
	"Table": {},
	"TR":    {},
	"TH":    {},
	"TD":    {},
	"THead": {},
	"TBody": {},
	"TFoot": {},
}

// IsStandardStructureType reports whether name is one of the standard
// PDF structure element types defined by ISO 32000-1, ISO 32000-2 or
// PDF/UA-2.
func IsStandardStructureType(name string) bool {
	_, ok := standardStructureTypes[name]
	return ok
}
