package graphics

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MTextChildren fails when a MathML mtext structure element has a
// PDF tag child that is not in the BPG-permitted subset (Reference,
// Link, Strong, Code, Em, Span, Lbl).
//
// BPG §"Inclusion of PDF structure elements into MathML" Table 2
// lists the seven PDF tags MathML mtext can legitimately carry as
// children, each with its HTML5 equivalent. Any other PDF tag
// inside mtext is a tagging error — assistive technology that
// honours the MathML namespace will not know how to interpret it.
//
// Until MH-17-005 introduces /NS-map-based namespace resolution,
// this check operates on a conservative heuristic: it flags only
// children whose tag matches a known PDF structure type (the names
// in ISO 32000-2 §14.8.4 / ISO 14289). Children with unknown tag
// names are assumed to live in the MathML namespace and are passed
// over. The trade-off is no false positives at the cost of missing
// custom-rolemap PDF tags that happen to share a name with neither
// the standard set nor MathML.
//
// PDF/UA-2 only.
type MTextChildren struct{}

func (MTextChildren) ID() string                { return "MH-17-006" }
func (MTextChildren) Title() string             { return "MathML mtext only carries permitted PDF tag children" }
func (MTextChildren) Category() engine.Category { return engine.CategoryGraphics }
func (MTextChildren) Severity() engine.Severity { return engine.SeverityError }
func (MTextChildren) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (MTextChildren) WCAG() []string            { return []string{"1.3.1"} }
func (MTextChildren) Description() string {
	return "PDF/UA-2 (BPG \"Math in PDF\" §\"Inclusion of PDF structure elements into MathML\", Table 2) restricts the PDF structure elements permitted as direct children of MathML's mtext element to seven tags: Reference, Link, Strong, Code, Em, Span, Lbl. Any other PDF tag inside mtext is a tagging error. Heuristic: until namespace resolution lands, the check fires only when the offending child tag matches a known PDF structure type."
}

// mtextWhitelist enumerates the seven PDF tags permitted as direct
// children of MathML mtext. Source: BPG "Math in PDF" Table 2.
var mtextWhitelist = map[string]bool{
	"Reference": true,
	"Link":      true,
	"Strong":    true,
	"Code":      true,
	"Em":        true,
	"Span":      true,
	"Lbl":       true,
}

// knownPDFTags is the conservative set of standard PDF structure
// types we use to recognise "this child is definitely a PDF tag, not
// MathML". Sources: ISO 32000-2 §14.8.4 (Standard structure types)
// and ISO 14289 grouping/inline/illustration extensions. The list is
// deliberately not exhaustive — a custom rolemap that introduces a
// non-standard tag name will read here as "unknown" and pass the
// check. MH-17-005 will replace this heuristic with namespace-based
// resolution.
var knownPDFTags = map[string]bool{
	// Grouping
	"Document": true, "Part": true, "Art": true, "Sect": true, "Div": true,
	"BlockQuote": true, "Caption": true, "TOC": true, "TOCI": true,
	"Index": true, "NonStruct": true, "Private": true, "Quote": true,
	"Note": true, "BibEntry": true,
	// Paragraph-like
	"P":  true,
	"H":  true,
	"H1": true, "H2": true, "H3": true, "H4": true, "H5": true, "H6": true,
	// Inline (whitelisted ones are deliberately also listed in
	// mtextWhitelist so we treat them as PDF tags and then accept them.)
	"Span": true, "Em": true, "Strong": true, "Code": true,
	"Link": true, "Reference": true, "Annot": true,
	// Lists
	"L": true, "LI": true, "Lbl": true, "LBody": true,
	// Tables
	"Table": true, "TR": true, "TH": true, "TD": true,
	"THead": true, "TBody": true, "TFoot": true,
	// Illustration / other
	"Figure": true, "Formula": true, "Form": true,
	"Ruby": true, "RB": true, "RT": true, "RP": true,
	"Warichu": true, "WT": true, "WP": true,
	"Title": true,
}

func (c MTextChildren) Run(doc model.Document) []engine.Finding {
	if !isPDFUA2(doc) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "check applies to PDF/UA-2 only; document does not declare pdfuaid:part = 2",
		}}
	}
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	mtextCount := 0
	walkMText(root, "/"+root.Type(), func(mtext model.StructElement, path string) {
		mtextCount++
		for _, child := range mtext.Children() {
			t := child.Type()
			if mtextWhitelist[t] {
				continue
			}
			if !knownPDFTags[t] {
				// Unknown tag — heuristic punts to "probably MathML",
				// no finding. MH-17-005 will tighten this.
				continue
			}
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message: fmt.Sprintf(
					"mtext child %q is not in the BPG-permitted set (Reference, Link, Strong, Code, Em, Span, Lbl)",
					t),
				Hint:     "Re-tag the child as one of the permitted PDF tags, or move it outside the mtext.",
				Location: &engine.Location{Page: mtext.Page(), StructPath: path + "/" + t},
			})
		}
	})
	if mtextCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no mtext structure elements -- nothing to inspect",
		}}
	}
	return findings
}

// walkMText invokes fn for every mtext structure element under elem,
// threading the slash-separated tag path through. Companion to
// walkFormulas in formula_mathml_af.go; the two could share a
// generic walker but stay separate so each check's tag predicate
// reads as a literal in its own file.
func walkMText(elem model.StructElement, path string, fn func(model.StructElement, string)) {
	if elem.Type() == "mtext" {
		fn(elem, path)
	}
	for _, child := range elem.Children() {
		walkMText(child, path+"/"+child.Type(), fn)
	}
}

func init() { engine.Register(MTextChildren{}) }
