package optionalcontent

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// OCGNoAS fails when the default Optional Content configuration
// dictionary carries an /AS key.
//
// "The AS key shall not appear in any optional content configuration
// dictionary." The /AS key declares automatic state-change rules that
// adjust OCG visibility based on usage information (Print, Export,
// View). PDF/UA forbids it so optional-content visibility cannot
// silently flip behind AT's back during reading.
//
// Spec gating: both PDF/UA-1 and PDF/UA-2. The prohibition is present
// verbatim in ISO 14289-1 §7.10 (Matterhorn 20-003) and ISO 14289-2
// §8.7 -- so the check must fire on UA-1 documents too.
//
// Limitation: only /OCProperties/D (the default configuration) is
// inspected today. /OCProperties/Configs (alternate
// configurations) is rarely used in practice; checking it requires
// a model-side array accessor which is tracked as future work.
type OCGNoAS struct{}

func (OCGNoAS) ID() string                { return "UA-20-002" }
func (OCGNoAS) Title() string             { return "Optional Content default configuration does not declare /AS" }
func (OCGNoAS) Category() engine.Category { return engine.CategoryOptionalContent }
func (OCGNoAS) Severity() engine.Severity { return engine.SeverityError }
func (OCGNoAS) Spec() engine.Spec         { return engine.SpecBoth }
func (OCGNoAS) WCAG() []string            { return []string{"1.3.1"} }
func (OCGNoAS) Description() string {
	return "ISO 14289-1 §7.10 and ISO 14289-2 §8.7 forbid the /AS key in any Optional Content configuration dictionary. /AS declares automatic state-change rules that adjust layer visibility based on usage information (Print, Export, View). PDF/UA disallows it so optional-content visibility cannot silently change behind assistive technology's back during reading."
}

func (c OCGNoAS) Run(doc model.Document) []engine.Finding {
	part, found, err := pdfua.DetectPart(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata: " + err.Error(),
		}}
	}
	if !found || (part != 1 && part != 2) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document does not declare a PDF/UA conformance level (pdfuaid:part 1 or 2)",
		}}
	}
	cat, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	ocpObj, ok := cat.Find("OCProperties")
	if !ok {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no /OCProperties -- no optional-content configurations to inspect",
		}}
	}
	ocp, err := doc.DereferenceDict(ocpObj)
	if err != nil || ocp == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "/OCProperties does not resolve to a dictionary",
		}}
	}
	dObj, ok := ocp.Find("D")
	if !ok {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "/OCProperties has no /D (default configuration) to inspect",
		}}
	}
	d, err := doc.DereferenceDict(dObj)
	if err != nil || d == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "/OCProperties/D does not resolve to a dictionary",
		}}
	}
	if _, hasAS := d.Find("AS"); !hasAS {
		return nil
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "default optional-content configuration /OCProperties/D declares /AS, which is forbidden by PDF/UA",
		Hint:     "Remove the /AS entry from the /D configuration dictionary. Layer state should be authored statically, not driven by usage-based automatic rules.",
	}}
}

func init() { engine.Register(OCGNoAS{}) }
