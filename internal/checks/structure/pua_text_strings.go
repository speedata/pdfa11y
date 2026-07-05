package structure

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// PUAInTextStrings fails when a text string intended to be read by a human
// contains a Unicode Private Use Area code point. ISO 14289-2 §8.6 (veraPDF
// UA2:8.6-1): PUA code points carry no assigned meaning, so a document
// information entry, outline title, form-field name, annotation description or
// structure alternate text that uses them cannot be voiced by assistive
// technology. This is the general text-string counterpart to UA-01-013, which
// covers only /Alt and /ActualText.
//
// One finding per offending string. N/A when the document has no
// human-readable text strings to inspect.
type PUAInTextStrings struct{}

func (PUAInTextStrings) ID() string { return "UA-01-019" }
func (PUAInTextStrings) Title() string {
	return "Human-readable text strings do not contain Private Use Area characters"
}
func (PUAInTextStrings) Category() engine.Category { return engine.CategoryRealContent }
func (PUAInTextStrings) Severity() engine.Severity { return engine.SeverityError }
func (PUAInTextStrings) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (PUAInTextStrings) WCAG() []string            { return []string{"1.1.1"} }
func (PUAInTextStrings) Description() string {
	return "ISO 14289-2 §8.6 forbids Unicode Private Use Area (PUA) code points in text strings intended to be read by a human -- document information (Title, Author, Subject, Keywords), outline titles, form-field names, annotation contents and structure alternate text. PUA code points carry no assigned meaning that assistive technology could voice."
}

func (c PUAInTextStrings) Run(doc model.Document) []engine.Finding {
	strings := doc.HumanReadableStrings()
	if len(strings) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no human-readable text strings -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, s := range strings {
		if containsPUA(s.Value) {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("%s contains a Unicode Private Use Area character", s.Location),
				Hint:     "Replace the Private Use Area code point(s) with readable Unicode text.",
			})
		}
	}
	return findings
}

func init() { engine.Register(PUAInTextStrings{}) }
