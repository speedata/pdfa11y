// Package language groups checks that inspect natural-language declarations
// in the PDF (Catalog /Lang, structure-element /Lang, language tags inside
// content streams).
package language

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// CatalogLang reports when the document has no /Lang entry at the
// Catalog level. Its severity is spec-dependent:
//
//   - PDF/UA-1 §7.2 allows natural language to be declared at the
//     Catalog AND/OR per structure element, so a missing Catalog /Lang
//     is not by itself a violation -- per-element /Lang (UA-11-002) can
//     compensate. Severity Warning.
//   - PDF/UA-2 §8.4.4 tightens this to an unconditional "shall": a
//     non-empty /Lang in the Catalog is mandatory and per-element /Lang
//     only marks language changes. Severity Error.
//
// A present-but-empty /Lang () declares no language and is treated the
// same as a missing entry (veraPDF UA2:8.4.4-1 containsLang).
type CatalogLang struct{}

func (CatalogLang) ID() string    { return "UA-11-001" }
func (CatalogLang) Title() string { return "Document declares a primary language" }
func (CatalogLang) Description() string {
	return "PDF/UA-1 §7.2 lets the document's natural language be declared as /Lang on the catalog OR per structure element, so a missing Catalog /Lang is a Warning (per-element coverage may compensate, see UA-11-002). PDF/UA-2 §8.4.4 makes a non-empty Catalog /Lang mandatory, so the same omission is a hard Error under PDF/UA-2."
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
	// §8.4.4 requires the Catalog /Lang to be present AND non-empty; an
	// empty /Lang () declares no language, so it is treated as missing.
	lang, _ := catalog.String("Lang")
	_, found := catalog.Find("Lang")
	if !found || lang == "" {
		// Severity is spec-dependent: a missing/empty Catalog /Lang is a
		// hard violation under PDF/UA-2 (§8.4.4) but only a Warning under
		// PDF/UA-1 (§7.2), where per-element /Lang can compensate.
		state := "no /Lang at the Catalog"
		if found {
			state = "empty /Lang at the Catalog"
		}
		severity := engine.SeverityWarning
		msg := state + " — natural language may still be declared per structure element"
		if part, ok, _ := pdfua.DetectPart(doc); ok && part == 2 {
			severity = engine.SeverityError
			msg = state + " — PDF/UA-2 §8.4.4 requires a non-empty Catalog /Lang"
		}
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: severity,
			Message:  msg,
			Hint:     "Set the primary language at the Catalog (e.g. /Lang (en-US)); for mixed-language documents, also set /Lang on the relevant structure elements.",
		}}
	}
	return nil
}

func init() { engine.Register(CatalogLang{}) }
