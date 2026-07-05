package language

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// MetadataLang fails when the document's XMP metadata carries a language
// alternative tagged xml:lang="x-default" but the catalog declares no /Lang, so
// the natural language of the metadata cannot be determined. ISO 14289-1 §7.2
// (veraPDF UA1:7.2-33): x-default is a fallback, not a language, so the
// document-level language must come from the catalog /Lang. Without it,
// assistive technology cannot voice the title and other metadata correctly.
//
// One document-level finding. N/A when the metadata carries no x-default
// language alternative.
type MetadataLang struct{}

func (MetadataLang) ID() string                { return "UA-11-007" }
func (MetadataLang) Title() string             { return "Document metadata has a determinable language" }
func (MetadataLang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (MetadataLang) Severity() engine.Severity { return engine.SeverityError }
func (MetadataLang) Spec() engine.Spec         { return engine.SpecBoth }
func (MetadataLang) WCAG() []string            { return []string{"3.1.2"} }
func (MetadataLang) Description() string {
	return "ISO 14289-1 §7.2 requires the natural language of document metadata to be determinable. XMP language alternatives (e.g. dc:title) carry an xml:lang=\"x-default\" fallback, which is not itself a language, so the document must declare a catalog /Lang. Metadata with an x-default alternative but no catalog /Lang leaves its language undeterminable."
}

func (c MetadataLang) Run(doc model.Document) []engine.Finding {
	hasXDefault, err := pdfua.HasXDefaultLangAlt(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata: " + err.Error(),
		}}
	}
	if !hasXDefault {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document metadata has no x-default language alternative -- nothing to inspect",
		}}
	}
	if cat, err := doc.Catalog(); err == nil {
		if lang, ok := cat.String("Lang"); ok && lang != "" {
			return nil // language is determinable
		}
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "document metadata carries an x-default language alternative but the catalog declares no /Lang, so its language cannot be determined",
		Hint:     "Declare a document-wide /Lang on the catalog so the natural language of the x-default metadata is determinable.",
	}}
}

func init() { engine.Register(MetadataLang{}) }
