package graphics

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MathChildNamespace fails when a Formula carries a direct child
// named "math" that is not declared in the W3C MathML namespace.
//
// ISO 32000-2 §14.8.6.3 registers http://www.w3.org/1998/Math/MathML
// as one of three standardized namespaces for tagged PDF; PDF/UA-2
// §8.2.5.29.1 expects a 'math' struct child of Formula to live in
// that namespace, not in the default PDF namespace. A coincidentally-
// named 'math' tag in the default namespace would be treated as a
// custom (unmapped) type by AT, defeating the inline-MathML route.
//
// PDF/UA-2 only. Companion to UA-17-006: 005 verifies the math
// element's namespace, 006 verifies that PDF-tag children inside
// MathML's mtext are from the permitted set.
type MathChildNamespace struct{}

func (MathChildNamespace) ID() string                { return "UA-17-005" }
func (MathChildNamespace) Title() string             { return "math struct child of Formula is in the MathML namespace" }
func (MathChildNamespace) Category() engine.Category { return engine.CategoryGraphics }
func (MathChildNamespace) Severity() engine.Severity { return engine.SeverityError }
func (MathChildNamespace) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (MathChildNamespace) WCAG() []string            { return []string{"1.3.1"} }
func (MathChildNamespace) Description() string {
	return "PDF/UA-2 §8.2.5.29.1 with ISO 32000-2 §14.8.6.3 requires a 'math' struct element nested under Formula to declare its namespace (directly via /NS or inherited via /P) as http://www.w3.org/1998/Math/MathML. A 'math' tag in the default PDF namespace is treated as an unmapped custom type by assistive technology -- the MathML semantics never surface."
}

func (c MathChildNamespace) Run(doc model.Document) []engine.Finding {
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
	walkFormulas(root, "/"+root.Type(), func(formula model.StructElement, path string) {
		for _, child := range formula.Children() {
			if child.Type() != "math" {
				continue
			}
			mathCount++
			ns := child.Namespace()
			if ns == mathMLNamespace {
				continue
			}
			msg := "math child of Formula declares no namespace; expected " + mathMLNamespace
			if ns != "" {
				msg = fmt.Sprintf(
					"math child of Formula is in namespace %q, expected %q",
					ns, mathMLNamespace)
			}
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  msg,
				Hint:     "Attach a /NS reference on the math struct element (or one of its ancestors) pointing to a Namespace dictionary whose /NS string is http://www.w3.org/1998/Math/MathML.",
				Location: &engine.Location{Page: child.Page(), StructPath: path + "/math"},
			})
		}
	})
	if mathCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no math struct children of Formula -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(MathChildNamespace{}) }
