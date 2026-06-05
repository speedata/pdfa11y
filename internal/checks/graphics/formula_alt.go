package graphics

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FormulaAlt fails for every Formula structure element that carries
// neither /Alt nor /ActualText. PDF/UA-1 §7.5 (and ISO 32000-1
// §14.8.4.3 for structure types) requires Formula to provide an
// accessible text equivalent. The rendered glyphs of a math formula
// rarely round-trip through text extraction (greek letters, special
// operator code points, custom math fonts), so without /Alt or
// /ActualText assistive technology has no usable representation.
//
// Same shape as MH-13-004 for Figure: one finding per offending
// Formula, the check declines (N/A) when the document has no
// structure tree or no Formula elements at all.
type FormulaAlt struct{}

func (FormulaAlt) ID() string    { return "MH-17-001" }
func (FormulaAlt) Title() string { return "Formula has Alt or ActualText" }
func (FormulaAlt) Description() string {
	return "Every Formula structure element must provide an accessible text equivalent. PDF/UA-1 §7.5 accepts /Alt (a description such as 'quadratic formula') or /ActualText (a verbatim text representation, e.g. 'a^2 + b^2 = c^2'). Without it screen readers cannot announce the formula -- the rendered glyphs typically rely on specialised math fonts whose Unicode mapping is unreliable."
}
func (FormulaAlt) Category() engine.Category { return engine.CategoryGraphics }
func (FormulaAlt) Severity() engine.Severity { return engine.SeverityError }
func (FormulaAlt) Spec() engine.Spec         { return engine.SpecBoth }
func (FormulaAlt) WCAG() []string            { return []string{"1.1.1"} }

func (c FormulaAlt) Run(doc model.Document) []engine.Finding {
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	formulaCount := 0
	c.walk(root, "/"+root.Type(), &findings, &formulaCount)
	if formulaCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Formula structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c FormulaAlt) walk(elem model.StructElement, path string, out *[]engine.Finding, count *int) {
	if elem.Type() == "Formula" {
		*count++
		if elem.Attr("Alt") == "" && elem.Attr("ActualText") == "" {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "Formula has neither /Alt nor /ActualText",
				Hint:     "Add /Alt with a textual description of the formula (e.g. 'Pythagorean theorem') or /ActualText with the verbatim text form (e.g. 'a^2 + b^2 = c^2').",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

func init() { engine.Register(FormulaAlt{}) }
