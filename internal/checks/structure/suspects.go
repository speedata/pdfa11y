package structure

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Suspects fails when the document's /MarkInfo dictionary sets /Suspects to
// true. ISO 14289-1 §7.1 (ISO 32000-1 Table 321) requires a conforming file to
// have /Suspects false: a true value is the producer's own declaration that
// the tag tree may not faithfully represent the content ("suspect" tagging),
// which is incompatible with the PDF/UA promise that the structure is
// trustworthy.
//
// The flag defaults to false, so a document without /MarkInfo or without a
// /Suspects entry passes. One finding when /Suspects is true.
type Suspects struct{}

func (Suspects) ID() string                { return "UA-01-016" }
func (Suspects) Title() string             { return "MarkInfo /Suspects is not true" }
func (Suspects) Category() engine.Category { return engine.CategoryStructure }
func (Suspects) Severity() engine.Severity { return engine.SeverityError }
func (Suspects) Spec() engine.Spec         { return engine.SpecBoth }
func (Suspects) WCAG() []string            { return []string{"1.3.1"} }
func (Suspects) Description() string {
	return "ISO 14289-1 §7.1 requires the catalog's /MarkInfo /Suspects entry to be false (ISO 32000-1 Table 321). A true value marks the tagging as unreliable -- the producer flags that the structure tree may not match the real content -- which contradicts the PDF/UA guarantee that assistive technology can trust the tags."
}

func (c Suspects) Run(doc model.Document) []engine.Finding {
	cat, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	miObj, found := cat.Find("MarkInfo")
	if !found {
		return nil // no /MarkInfo -> Suspects defaults to false
	}
	mi, err := doc.DereferenceDict(miObj)
	if err != nil {
		return nil // malformed /MarkInfo is not this check's concern
	}
	susObj, ok := mi.Find("Suspects")
	if !ok {
		return nil // entry absent -> defaults to false
	}
	suspect, err := doc.DereferenceBoolean(susObj)
	if err != nil || !suspect {
		return nil
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "catalog /MarkInfo /Suspects is true -- the document declares its own tagging as unreliable",
		Hint:     "Set /MarkInfo /Suspects to false (or remove the entry) once the tag tree faithfully represents the content.",
	}}
}

func init() { engine.Register(Suspects{}) }
