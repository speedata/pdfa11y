package xobjects

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// UniqueSemanticParent fails when a form XObject that carries marked content
// (an /MCID inside its content stream) is invoked by a Do operator more than
// once. ISO 14289-1 §7.20 (veraPDF UA1:7.20-2) requires the content of form
// XObjects to be incorporated into structure elements per ISO 32000-1 §14.7.2.
// A marked-content sequence inside a form XObject carries an MCID that the
// structure tree resolves to exactly one place. If the XObject is drawn more
// than once, that MCID maps to several page positions at once, so its "semantic
// parent" in the structure tree is ambiguous and assistive technology cannot
// place the tagged content.
//
// One finding per offending XObject. N/A when the document has no form
// XObjects. PDF/UA-1 only (the veraPDF rule is scoped to PDFUA_1).
type UniqueSemanticParent struct{}

func (UniqueSemanticParent) ID() string { return "UA-30-002" }
func (UniqueSemanticParent) Title() string {
	return "Form XObjects with marked content are invoked at most once"
}
func (UniqueSemanticParent) Category() engine.Category { return engine.CategoryXObjects }
func (UniqueSemanticParent) Severity() engine.Severity { return engine.SeverityError }
func (UniqueSemanticParent) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (UniqueSemanticParent) WCAG() []string            { return []string{"1.3.1"} }
func (UniqueSemanticParent) Description() string {
	return "ISO 14289-1 §7.20 requires the content of form XObjects to be incorporated into structure elements (ISO 32000-1 §14.7.2). A form XObject carrying an /MCID marked-content sequence is tied to a single place in the structure tree; drawing it via Do more than once makes that MCID map to several positions at once, so its semantic parent is ambiguous. A marked-content form XObject must be invoked at most once."
}

func (c UniqueSemanticParent) Run(doc model.Document) []engine.Finding {
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
		if !x.HasMarkedContent || x.RefCount <= 1 {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("form XObject (object %d) carries marked content but is invoked %d times, so its MCIDs have no unique semantic parent", x.ObjectNumber, x.RefCount),
			Hint:     "Draw the marked-content form XObject only once, or give each instance its own XObject so each MCID resolves to a single structure element.",
			Location: &engine.Location{ObjectNumber: x.ObjectNumber},
		})
	}
	return findings
}

func init() { engine.Register(UniqueSemanticParent{}) }
