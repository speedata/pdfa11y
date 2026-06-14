package graphics

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// FormulaAlt fails for every Formula structure element that lacks
// any of the accessible math representations the spec permits.
//
// PDF/UA-1 (ISO 14289-1 §7.5) recognises only the legacy path:
// /Alt or /ActualText on the Formula. PDF/UA-2 (ISO 14289-2
// §8.2.5.29.1, with the PDF Association BPG "Math in PDF" as
// companion guidance) adds two MathML paths -- a 'math' child
// element in the MathML namespace, or an Associated File whose
// embedded stream carries MathML (AFRelationship=/Supplement,
// Subtype=application/mathml+xml). When the spec is PDF/UA-2 any of
// the four representations passes; the legacy /Alt is optional but
// MathML, when present, takes precedence at AT consumption time
// (BPG §"Precedence").
//
// The check declines (N/A) when the document has no structure tree
// or no Formula elements at all. Spec selection follows the same
// pattern as MH-09-003: read pdfuaid:part from XMP so engine.All()
// callers (tests, the realworld driver) see UA-2 behaviour on
// PDF/UA-2 documents without the CLI having to filter.
type FormulaAlt struct{}

func (FormulaAlt) ID() string    { return "MH-17-001" }
func (FormulaAlt) Title() string { return "Formula has accessible math representation" }
func (FormulaAlt) Description() string {
	return "Every Formula structure element must provide an accessible math representation. PDF/UA-1 §7.5 requires /Alt or /ActualText. PDF/UA-2 §8.2.5.29.1 additionally accepts MathML -- either as a direct 'math' child of Formula or as an Associated File with AFRelationship /Supplement and Subtype application/mathml+xml (PDF Association BPG \"Math in PDF\"). Without one of these, assistive technology has no usable representation; the rendered glyphs rely on specialised math fonts whose Unicode mapping is unreliable."
}
func (FormulaAlt) Category() engine.Category { return engine.CategoryGraphics }
func (FormulaAlt) Severity() engine.Severity { return engine.SeverityError }
func (FormulaAlt) Spec() engine.Spec         { return engine.SpecBoth }
func (FormulaAlt) WCAG() []string            { return []string{"1.1.1"} }

// mathMLSubtype is the MIME type a MathML Associated File must
// declare per BPG §"Use of Associated files" / ISO 14289-2
// §8.2.5.29.1. PDF/UA-2 §14.8.6.3 also registers
// http://www.w3.org/1998/Math/MathML as the namespace for inline
// 'math' struct-element children; a future MH-17-005 will validate
// that namespace declaration explicitly.
const mathMLSubtype = "application/mathml+xml"

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

	// Spec detection: PDF/UA-2 accepts MathML paths; PDF/UA-1 (or no
	// declared part) accepts only /Alt or /ActualText. We do NOT
	// fall back to "any spec" when the part is missing because that
	// would silently accept MathML paths on documents that never
	// claimed UA-2 conformance.
	ua2 := false
	if part, found, err := pdfua.DetectPart(doc); err == nil && found && part == 2 {
		ua2 = true
	}

	var findings []engine.Finding
	formulaCount := 0
	c.walk(root, "/"+root.Type(), ua2, &findings, &formulaCount)
	if formulaCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Formula structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c FormulaAlt) walk(elem model.StructElement, path string, ua2 bool, out *[]engine.Finding, count *int) {
	if elem.Type() == "Formula" {
		*count++
		if !hasAccessibleMath(elem, ua2) {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  failureMessage(ua2),
				Hint:     failureHint(ua2),
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), ua2, out, count)
	}
}

// hasAccessibleMath reports whether the Formula element carries any
// representation that satisfies its target spec. Order of checks is
// legacy-first (cheapest) so PDF/UA-1 documents short-circuit
// without inspecting children or AFs.
//
// For the math-child path we require the child to live in the W3C
// MathML namespace, not merely be named "math" (an /NS-less or
// wrong-namespace 'math' is treated by AT as an unmapped custom
// type and conveys no math semantics). MH-17-005 reports the
// namespace problem in its own finding when present.
func hasAccessibleMath(elem model.StructElement, ua2 bool) bool {
	if elem.Attr("Alt") != "" || elem.Attr("ActualText") != "" {
		return true
	}
	if !ua2 {
		return false
	}
	for _, child := range elem.Children() {
		if child.Type() == "math" && child.Namespace() == mathMLNamespace {
			return true
		}
	}
	for _, af := range elem.AssociatedFiles() {
		if af.Relationship == "Supplement" && af.Subtype == mathMLSubtype {
			return true
		}
	}
	return false
}

func failureMessage(ua2 bool) string {
	if ua2 {
		return "Formula has no accessible math representation (no /Alt, /ActualText, math child, or MathML associated file)"
	}
	return "Formula has neither /Alt nor /ActualText"
}

func failureHint(ua2 bool) string {
	if ua2 {
		return "Add one of: a 'math' struct-element child in the MathML namespace; an Associated File with /AFRelationship /Supplement and /Subtype /application#2Fmathml+xml; or /Alt (a description) or /ActualText (a verbatim text form) on the Formula."
	}
	return "Add /Alt with a textual description of the formula (e.g. 'Pythagorean theorem') or /ActualText with the verbatim text form (e.g. 'a^2 + b^2 = c^2')."
}

func init() { engine.Register(FormulaAlt{}) }
