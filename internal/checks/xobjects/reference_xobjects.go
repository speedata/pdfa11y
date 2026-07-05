// Package xobjects groups checks that inspect form XObjects (schedule 30).
package xobjects

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ReferenceXObjects fails when the document contains a reference XObject: a
// form XObject carrying a /Ref entry, which imports page content from an
// external file (ISO 32000-1 §8.10.4). ISO 14289-1 §7.20 (veraPDF UA1:7.20-1)
// forbids reference XObjects because their imported content is not part of the
// document's own structure tree and cannot be tagged, so assistive technology
// cannot reach it. PDF 2.0 deprecates the feature, so PDF/UA-2 documents shall
// not use it either.
//
// One finding per reference XObject. N/A when the document has none.
type ReferenceXObjects struct{}

func (ReferenceXObjects) ID() string                { return "UA-30-001" }
func (ReferenceXObjects) Title() string             { return "Document contains no reference XObjects" }
func (ReferenceXObjects) Category() engine.Category { return engine.CategoryXObjects }
func (ReferenceXObjects) Severity() engine.Severity { return engine.SeverityError }
func (ReferenceXObjects) Spec() engine.Spec         { return engine.SpecBoth }
func (ReferenceXObjects) WCAG() []string            { return []string{"1.3.1"} }
func (ReferenceXObjects) Description() string {
	return "ISO 14289-1 §7.20 forbids reference XObjects (form XObjects with a /Ref entry, ISO 32000-1 §8.10.4). A reference XObject imports page content from an external file; that content is not part of this document's structure tree and cannot be tagged, so assistive technology cannot reach it. PDF 2.0 deprecates the feature, so PDF/UA-2 documents must not use it either."
}

func (c ReferenceXObjects) Run(doc model.Document) []engine.Finding {
	xobjects := doc.FormXObjects()
	if len(xobjects) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no form XObjects -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, x := range xobjects {
		if !x.IsReference {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("form XObject (object %d) is a reference XObject (carries /Ref), which is forbidden", x.ObjectNumber),
			Hint:     "Remove the /Ref entry and embed the referenced content directly, tagged into this document's structure tree.",
			Location: &engine.Location{ObjectNumber: x.ObjectNumber},
		})
	}
	return findings
}

func init() { engine.Register(ReferenceXObjects{}) }
