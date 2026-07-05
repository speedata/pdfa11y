package language

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// OutlineLang fails when a document carries outline entries (bookmarks) but no
// catalog /Lang, so the natural language of the outline text cannot be
// determined. ISO 14289-1 §7.2 (veraPDF UA1:7.2-2): outline items carry no
// /Lang of their own, so the only place their language can be declared is the
// document catalog. Without it, assistive technology cannot voice the bookmark
// labels correctly.
//
// One document-level finding. N/A when the document has no bookmarks.
type OutlineLang struct{}

func (OutlineLang) ID() string                { return "UA-11-006" }
func (OutlineLang) Title() string             { return "Outline entries have a determinable language" }
func (OutlineLang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (OutlineLang) Severity() engine.Severity { return engine.SeverityError }
func (OutlineLang) Spec() engine.Spec         { return engine.SpecBoth }
func (OutlineLang) WCAG() []string            { return []string{"3.1.2"} }
func (OutlineLang) Description() string {
	return "ISO 14289-1 §7.2 requires the natural language of outline (bookmark) entries to be determinable. Outline items carry no /Lang of their own, so their language must come from the document catalog's /Lang. A document with bookmarks but no catalog /Lang leaves their language undeterminable, and assistive technology cannot voice them correctly."
}

func (c OutlineLang) Run(doc model.Document) []engine.Finding {
	cat, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	// Bookmarks present? /Outlines must resolve to a dict with a /First
	// entry (at least one top-level item).
	outlinesObj, ok := cat.Find("Outlines")
	if !ok {
		return c.notApplicable()
	}
	outlines, err := doc.DereferenceDict(outlinesObj)
	if err != nil || outlines == nil {
		return c.notApplicable()
	}
	if _, hasFirst := outlines.Find("First"); !hasFirst {
		return c.notApplicable()
	}

	if lang, ok := cat.String("Lang"); ok && lang != "" {
		return nil // language is determinable
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "document has outline entries but no catalog /Lang, so their language cannot be determined",
		Hint:     "Declare a document-wide /Lang on the catalog. Outline items cannot carry their own /Lang, so the catalog is the only source.",
	}}
}

func (c OutlineLang) notApplicable() []engine.Finding {
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityNotApplicable,
		Message:  "document has no outline entries -- nothing to inspect",
	}}
}

func init() { engine.Register(OutlineLang{}) }
