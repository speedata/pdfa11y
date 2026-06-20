package structure

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MarkInfo fails if the catalog has no /MarkInfo dictionary with
// /Marked set to true. PDF/UA-1 §6 requires a conforming document to
// declare itself as containing marked content -- without it, even a
// document that ships a structure tree is not technically "tagged".
type MarkInfo struct{}

func (MarkInfo) ID() string    { return "UA-01-002" }
func (MarkInfo) Title() string { return "MarkInfo declares the document as marked" }
func (MarkInfo) Description() string {
	return "PDF/UA-1 §6 requires the catalog to declare the document as tagged via /MarkInfo<</Marked true>>. A structure tree alone is not sufficient -- consumers use MarkInfo to recognise tagged content."
}
func (MarkInfo) Category() engine.Category { return engine.CategoryStructure }
func (MarkInfo) Severity() engine.Severity { return engine.SeverityError }
func (MarkInfo) Spec() engine.Spec         { return engine.SpecBoth }
func (MarkInfo) WCAG() []string            { return []string{"1.3.1"} }

func (c MarkInfo) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}

	miObj, found := catalog.Find("MarkInfo")
	if !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "no MarkInfo dictionary — document is not declared as marked",
			Hint:     "Add /MarkInfo << /Marked true >> to the document Catalog.",
		}}
	}

	mi, err := doc.DereferenceDict(miObj)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read MarkInfo dictionary: " + err.Error(),
		}}
	}

	markedObj, found := mi.Find("Marked")
	if !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "MarkInfo has no Marked entry — defaults to false",
			Hint:     "Set /Marked true inside the MarkInfo dictionary.",
		}}
	}

	value, err := doc.DereferenceBoolean(markedObj)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read MarkInfo/Marked value: " + err.Error(),
		}}
	}
	if !value {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "MarkInfo/Marked is false",
			Hint:     "Change /Marked from false to true so the document is recognised as tagged.",
		}}
	}
	return nil
}

func init() { engine.Register(MarkInfo{}) }
