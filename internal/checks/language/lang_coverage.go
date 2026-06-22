package language

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// LangCoverage requires every text-bearing structure element to have
// an effective /Lang -- either set on the element itself or inherited
// from an ancestor. PDF/UA-1 §7.2 allows the natural language to be
// declared at the Catalog OR per structure element; this check covers
// the second path and fires only when the Catalog has no /Lang of its
// own. With a Catalog /Lang every element inherits the default, so
// per-element coverage is not required and the check declines (N/A).
//
// Spec gating: PDF/UA-1 only. The "Catalog and/or per element"
// substitution this check models exists solely in ISO 14289-1 §7.2.
// ISO 14289-2 §8.4.4 removes the flexibility -- a non-empty Catalog
// /Lang is mandatory, and per-element /Lang only marks language
// *changes* -- so under UA-2 a missing Catalog /Lang is already a
// violation (UA-11-001's domain) regardless of per-element coverage.
//
// "Text-bearing" is defined pragmatically as the set of standard tags
// that carry the document's prose: P, H/H1-H6, Span, Lbl, LBody, TD,
// TH, Caption, Note, BibEntry, Quote, Reference, TOCI. Containers
// (Document, Sect, Table, TR, L, LI, Figure, Formula, ...) are
// skipped: Figure uses /Alt rather than /Lang for assistive
// technology, and the others are pure structure with no prose of
// their own that AT would announce in a specific language.
type LangCoverage struct{}

func (LangCoverage) ID() string                { return "UA-11-002" }
func (LangCoverage) Title() string             { return "Text-bearing structure elements declare /Lang" }
func (LangCoverage) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (LangCoverage) Severity() engine.Severity { return engine.SeverityError }
func (LangCoverage) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (LangCoverage) WCAG() []string            { return []string{"3.1.1", "3.1.2"} }
func (LangCoverage) Description() string {
	return "PDF/UA-1 §7.2 allows the document's natural language to be declared either at the Catalog or per structure element. The Catalog form is covered by UA-11-001; this check verifies the per-element coverage when the Catalog form is absent. Every text-bearing element (P, H/H1-H6, Span, Lbl, LBody, TD, TH, Caption, Note, BibEntry, Quote, Reference, TOCI) must carry /Lang or inherit it from an ancestor."
}

// textBearingTypes enumerates the standard structure types whose
// prose belongs to a specific natural language and must therefore
// be covered by some /Lang on the element or an ancestor. Document
// containers (Sect, Part, Table, TR, L, LI, Figure, ...) are not
// listed: they have no narratable prose of their own.
var textBearingTypes = map[string]bool{
	"P":         true,
	"H":         true,
	"H1":        true,
	"H2":        true,
	"H3":        true,
	"H4":        true,
	"H5":        true,
	"H6":        true,
	"Span":      true,
	"Lbl":       true,
	"LBody":     true,
	"TD":        true,
	"TH":        true,
	"Caption":   true,
	"Note":      true,
	"BibEntry":  true,
	"Quote":     true,
	"Reference": true,
	"TOCI":      true,
}

func (c LangCoverage) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	if _, found := catalog.Find("Lang"); found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "Catalog /Lang is set -- every element inherits it, per-element coverage is not required",
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
	considered := 0
	c.walk(root, "", "/"+root.Type(), &findings, &considered)
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no text-bearing structure elements -- nothing to inspect",
		}}
	}
	return findings
}

// walk recurses through the structure tree, propagating the closest
// ancestor /Lang as "inherited". A text-bearing element with no
// effective /Lang (neither self nor inherited) produces a finding.
func (c LangCoverage) walk(elem model.StructElement, inherited, path string, out *[]engine.Finding, count *int) {
	effective := inherited
	if l := elem.Attr("Lang"); l != "" {
		effective = l
	}
	if textBearingTypes[elem.Type()] {
		*count++
		if effective == "" {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("text-bearing %s element has no effective /Lang (no Catalog /Lang, no /Lang on self or ancestor)", elem.Type()),
				Hint:     "Set /Lang on the element itself, on an ancestor that covers it, or on the Catalog. The Catalog form is the simplest fix for single-language documents.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, effective, path+"/"+child.Type(), out, count)
	}
}

func init() { engine.Register(LangCoverage{}) }
