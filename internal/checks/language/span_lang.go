package language

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// SpanLang fails when a Span marked-content sequence carries an /ActualText,
// /Alt or /E property whose natural language cannot be determined. ISO 14289-1
// §7.2 (veraPDF UA1:7.2-30/-31/-32) requires the language of such alternate or
// replacement text to be determinable, so assistive technology can voice it
// correctly. The language is determinable from the Span's own /Lang, a /Lang on
// an enclosing marked-content sequence, the /Lang inherited from the structure
// element that contains the Span, or a document-wide catalog /Lang. A Span that
// supplies none of these leaves its alternate text in an undetermined language.
//
// One finding per undeterminable Span attribute. N/A when the document carries
// no page content to inspect.
type SpanLang struct{}

func (SpanLang) ID() string                { return "UA-11-008" }
func (SpanLang) Title() string             { return "Span alternate text has a determinable language" }
func (SpanLang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (SpanLang) Severity() engine.Severity { return engine.SeverityError }
func (SpanLang) Spec() engine.Spec         { return engine.SpecBoth }
func (SpanLang) WCAG() []string            { return []string{"3.1.2"} }
func (SpanLang) Description() string {
	return "ISO 14289-1 §7.2 requires the natural language of a Span marked-content sequence's /ActualText, /Alt or /E text to be determinable. The language comes from the Span's own /Lang, a /Lang on an enclosing marked-content sequence, the /Lang inherited from the enclosing structure element, or the catalog /Lang. A Span that carries alternate text but supplies none of these leaves that text in an undetermined language, so assistive technology cannot voice it correctly."
}

func (c SpanLang) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no pages -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, p := range pages {
		for _, v := range p.SpanLangViolations {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d: Span /%s text has no determinable natural language (no /Lang on the Span, enclosing sequence or structure element, and no catalog /Lang)", p.Number, v.Attribute),
				Hint:     "Add a /Lang property to the Span (or its enclosing structure element), or declare a document-wide catalog /Lang, so the language of the alternate text is determinable.",
				Location: &engine.Location{Page: p.Number},
			})
		}
	}
	return findings
}

func init() { engine.Register(SpanLang{}) }
