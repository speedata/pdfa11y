package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// AcroFormInTree fails for every AcroForm widget that has no
// /StructParent entry tying it back to the structure tree. PDF/UA-1
// §7.18 requires interactive form fields to be reachable from the
// structure tree so AT can place them in the reading order. The
// symmetric MH-28-004 check walks per-page /Annots arrays; this one
// walks /AcroForm/Fields. The two paths usually surface the same
// widgets, but a field can have /Kids widgets that are not also
// listed in any page's /Annots array -- such orphans are caught only
// here.
//
// Abstract fields (non-terminal field dicts whose subtree contains
// no widget) are not surfaced by the model walker because they have
// no on-page surface to link from the structure tree.
type AcroFormInTree struct{}

func (AcroFormInTree) ID() string                { return "MH-28-007" }
func (AcroFormInTree) Title() string             { return "AcroForm fields are referenced from the structure tree" }
func (AcroFormInTree) Category() engine.Category { return engine.CategoryInteractive }
func (AcroFormInTree) Severity() engine.Severity { return engine.SeverityError }
func (AcroFormInTree) Spec() engine.Spec         { return engine.SpecBoth }
func (AcroFormInTree) WCAG() []string            { return []string{"1.3.1"} }
func (AcroFormInTree) Description() string {
	return "PDF/UA-1 §7.18 requires every interactive form field to be reachable from the structure tree via its widget's /StructParent. This check walks /AcroForm/Fields and verifies the cross-reference -- the symmetric MH-28-004 check walks per-page /Annots, so the two together cover both directions of the field ↔ structure-tree relationship."
}

func (c AcroFormInTree) Run(doc model.Document) []engine.Finding {
	fields, err := doc.AcroFormFields()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate AcroForm fields: " + err.Error(),
		}}
	}
	if len(fields) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no /AcroForm/Fields -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, f := range fields {
		if f.StructParent >= 0 {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("AcroForm widget %q has no /StructParent", displayName(f.FullName)),
			Hint:     "Add /StructParent on the widget annotation and reference it from a structure element via OBJR, so AT can place the field in the reading order.",
		})
	}
	return findings
}

// displayName falls back to "(unnamed)" when the widget chain
// carried no /T anywhere -- otherwise the finding message would end
// in an empty pair of quotes.
func displayName(s string) string {
	if s == "" {
		return "(unnamed)"
	}
	return s
}

func init() { engine.Register(AcroFormInTree{}) }
