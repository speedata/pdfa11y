package language

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FieldTULang fails when a form field carries a /TU (alternate field name /
// tooltip) whose natural language cannot be determined. ISO 14289-1 §7.2
// (veraPDF UA1:7.2-25): the language of a form field's /TU text must be
// determinable, so assistive technology can voice the field description
// correctly. It is determinable from the /Lang on the field's own structure
// element or a document-wide catalog /Lang. A field whose /TU has neither
// leaves that description in an undetermined language.
//
// One finding per offending field widget. N/A when no field widget carries a
// /TU.
type FieldTULang struct{}

func (FieldTULang) ID() string                { return "UA-11-009" }
func (FieldTULang) Title() string             { return "Form field /TU has a determinable language" }
func (FieldTULang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (FieldTULang) Severity() engine.Severity { return engine.SeverityError }
func (FieldTULang) Spec() engine.Spec         { return engine.SpecBoth }
func (FieldTULang) WCAG() []string            { return []string{"3.1.2"} }
func (FieldTULang) Description() string {
	return "ISO 14289-1 §7.2 requires the natural language of a form field's /TU (alternate field name) to be determinable. The language comes from the /Lang on the field's own structure element or the catalog /Lang. A field whose /TU has neither leaves its description in an undetermined language, so assistive technology cannot voice it correctly."
}

func (c FieldTULang) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	catalogLang := ""
	if cat, err := doc.Catalog(); err == nil {
		if l, ok := cat.String("Lang"); ok {
			catalogLang = l
		}
	}
	var findings []engine.Finding
	seen := 0
	for _, a := range annots {
		if !a.IsFieldWidget || a.FieldTooltip == "" {
			continue
		}
		seen++
		// Determinable from the field's own structure-element /Lang, the
		// field/widget /Lang, or the catalog /Lang.
		if a.StructOwnLang != "" || a.Lang != "" || catalogLang != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "form field /TU has no determinable natural language (no /Lang on the field's structure element and no catalog /Lang)",
			Hint:     "Declare /Lang on the field's structure element, or a document-wide catalog /Lang, so the language of the field's /TU description is determinable.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no form field carries a /TU -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(FieldTULang{}) }
