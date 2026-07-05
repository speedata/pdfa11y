package structure

import (
	"fmt"
	"slices"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// TOCIRef fails when a TOCI (table-of-contents item) structure element does
// not identify the target of its reference via a /Ref entry -- neither on
// the TOCI itself nor on any of its descendants. PDF/UA-2 §8.2.5.8 requires
// this link so assistive technology can navigate from a table-of-contents
// entry to the content it points at.
//
// The model exposes only /Ref targets that resolve to a structure element,
// so a TOCI whose /Ref exists but dangles is reported the same as one with
// no /Ref. One finding per TOCI without a resolvable /Ref. N/A when the
// document has no TOCI elements.
type TOCIRef struct{}

func (TOCIRef) ID() string                { return "UA-01-012" }
func (TOCIRef) Title() string             { return "TOC items reference their target via /Ref" }
func (TOCIRef) Category() engine.Category { return engine.CategoryStructure }
func (TOCIRef) Severity() engine.Severity { return engine.SeverityError }
func (TOCIRef) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (TOCIRef) WCAG() []string            { return []string{"1.3.1", "2.4.5"} }
func (TOCIRef) Description() string {
	return "PDF/UA-2 §8.2.5.8 requires every TOCI (table-of-contents item) to identify the target of its reference with a /Ref entry, either on the TOCI itself or on one of its child structure elements, so the table of contents links to the content it describes."
}

func (c TOCIRef) Run(doc model.Document) []engine.Finding {
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
	seen := c.walk(root, "/"+root.Type(), &findings)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no TOCI elements -- nothing to inspect",
		}}
	}
	return findings
}

// walk reports TOCI elements lacking a resolvable /Ref and returns the number
// of TOCI elements encountered.
func (c TOCIRef) walk(elem model.StructElement, path string, findings *[]engine.Finding) int {
	seen := 0
	for _, k := range elem.Children() {
		kpath := path + "/" + k.Type()
		if k.Type() == "TOCI" {
			seen++
			if !hasRefInSubtree(k) {
				*findings = append(*findings, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("TOCI at %s has no /Ref identifying its target (checked the TOCI and its descendants)", kpath),
					Hint:     "Add a /Ref entry to the TOCI (or a child) pointing at the structure element the table-of-contents entry refers to.",
					Location: &engine.Location{Page: k.Page(), StructPath: kpath},
				})
			}
		}
		seen += c.walk(k, kpath, findings)
	}
	return seen
}

// hasRefInSubtree reports whether elem or any descendant has a resolvable
// /Ref target.
func hasRefInSubtree(elem model.StructElement) bool {
	if len(elem.Refs()) > 0 {
		return true
	}
	return slices.ContainsFunc(elem.Children(), hasRefInSubtree)
}

func init() { engine.Register(TOCIRef{}) }
