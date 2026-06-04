// Package language groups checks that inspect natural-language declarations
// in the PDF (Catalog /Lang, structure-element /Lang, language tags inside
// content streams).
package language

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// CatalogLang warns when the document has no /Lang entry at the Catalog
// level. PDF/UA-1 §7.2 allows declaring natural language either at the
// Catalog or per structure element, so a missing Catalog /Lang is not
// definitively a violation — it requires the structure-element coverage to
// pick up the slack. Until that coverage check exists, we report a warning
// rather than a hard error.
type CatalogLang struct{}

func (CatalogLang) ID() string    { return "MH-11-001" }
func (CatalogLang) Title() string { return "Document declares a primary language" }
func (CatalogLang) Description() string {
	return "PDF/UA-1 §7.2 requires the document's natural language to be declared either as /Lang on the catalog or per structure element. Severity is Warning because per-element declarations may compensate; structure-element coverage is not yet verified by this tool."
}
func (CatalogLang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (CatalogLang) Severity() engine.Severity { return engine.SeverityWarning }
func (CatalogLang) Spec() engine.Spec         { return engine.SpecBoth }
func (CatalogLang) WCAG() []string            { return []string{"3.1.1"} }

func (c CatalogLang) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	if _, found := catalog.Find("Lang"); !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityWarning,
			Message:  "no /Lang at the Catalog — natural language may still be declared per structure element",
			Hint:     "Set the primary language at the Catalog (e.g. /Lang (en-US)); for mixed-language documents, also set /Lang on the relevant structure elements.",
		}}
	}
	return nil
}

func init() { engine.Register(CatalogLang{}) }
