// Package structure groups checks that inspect the PDF structure tree.
package structure

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// StructTreeRoot fails if the document has no structure tree at all,
// i.e. content is not tagged. Without a structure tree the document cannot
// be PDF/UA-conforming regardless of any other property.
//
// Matterhorn: 01-005 (mapped from "01 Real content" — no tags at all).
type StructTreeRoot struct{}

func (StructTreeRoot) ID() string    { return "MH-01-005" }
func (StructTreeRoot) Title() string { return "Document has a structure tree" }
func (StructTreeRoot) Description() string {
	return "A PDF/UA-conforming document must ship a structure tree (Tagged PDF). Without it, assistive technology sees only positioned glyphs -- there is no semantic outline to announce."
}
func (StructTreeRoot) Category() engine.Category { return engine.CategoryStructure }
func (StructTreeRoot) Severity() engine.Severity { return engine.SeverityError }
func (StructTreeRoot) Spec() engine.Spec         { return engine.SpecBoth }
func (StructTreeRoot) WCAG() []string            { return []string{"1.3.1"} }

func (c StructTreeRoot) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	if _, found := catalog.Find("StructTreeRoot"); !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "document has no StructTreeRoot — content is not tagged",
			Hint:     "Re-export the PDF with tagged structure enabled (Tagged PDF).",
		}}
	}
	return nil
}

func init() { engine.Register(StructTreeRoot{}) }
