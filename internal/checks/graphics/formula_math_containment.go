package graphics

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MathContainment fails when a `math` structure element in the
// MathML namespace appears outside any Formula structure element.
//
// ISO 14289-2 §8.2.5.29.1: "The math structure type, as defined in
// ISO 32000-2:2020, 14.8.6.3, shall occur only as a child of a
// Formula structure element."
//
// The literal reading -- "shall occur only as a *direct* child" --
// would flag the nested-math pattern the BPG itself demonstrates
// for piecewise expressions (math → mtext → math). The PDF
// Association's own "Best Practice Guide: Math in PDF" example of
// nested math under mtext fails such a strict reading. The
// pragmatic and BPG-compatible reading is: every `math` element
// must have a `Formula` ancestor somewhere up the structure-tree
// chain. This catches stray math elements outside any Formula
// while accepting the legitimate nested-math idiom.
//
// We only flag `math` elements that resolve to the W3C MathML
// namespace (per StructElement.Namespace()); a stray "math"-named
// element in the default PDF namespace is UA-31-008's territory
// (custom tag without a role map), not this check's.
//
// PDF/UA-2 only.
type MathContainment struct{}

func (MathContainment) ID() string                { return "UA-17-015" }
func (MathContainment) Title() string             { return "math struct element appears only as a child of Formula" }
func (MathContainment) Category() engine.Category { return engine.CategoryGraphics }
func (MathContainment) Severity() engine.Severity { return engine.SeverityError }
func (MathContainment) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (MathContainment) WCAG() []string            { return []string{"1.3.1"} }
func (MathContainment) Description() string {
	return "ISO 14289-2 §8.2.5.29.1 (with ISO 32000-2 §14.8.6.3 for the MathML namespace) requires a `math` structure element to occur only as a child of a `Formula` structure element. A `math` element placed elsewhere -- nested deeper, or directly under a non-Formula container -- breaks the contract that a single Formula wraps the entire mathematical expression."
}

func (c MathContainment) Run(doc model.Document) []engine.Finding {
	if !isPDFUA2(doc) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "check applies to PDF/UA-2 only; document does not declare pdfuaid:part = 2",
		}}
	}
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
	mathCount := 0
	c.walk(root, "/"+root.Type(), false, &findings, &mathCount)
	if mathCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no math struct elements -- nothing to inspect",
		}}
	}
	return findings
}

// walk traverses the structure tree, tracking whether the current
// element has any Formula ancestor so we can flag math elements
// that appear outside one. insideFormula stays true once we
// descend into a Formula and remains true for the whole subtree.
func (c MathContainment) walk(elem model.StructElement, path string, insideFormula bool, out *[]engine.Finding, count *int) {
	if elem.Type() == "math" && elem.Namespace() == mathMLNamespace {
		*count++
		if !insideFormula {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "math element has no Formula ancestor; expected to live inside a Formula",
				Hint:     "Wrap the math element in a Formula structure element. ISO 14289-2 §8.2.5.29.1 requires math to occur only as a child of Formula.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	descendantInsideFormula := insideFormula || elem.Type() == "Formula"
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), descendantInsideFormula, out, count)
	}
}

func init() { engine.Register(MathContainment{}) }
