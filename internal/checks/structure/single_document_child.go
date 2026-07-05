package structure

import (
	"fmt"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// SingleDocumentChild fails when the structure tree root does not have exactly
// one child and that child is not a Document structure element.
//
// ISO 14289-2 §8.2.5.2: "The structure tree root shall contain a single
// Document structure element as its only child" (see also ISO 32000-2:2020,
// Annex L and ISO/TS 32005). A root with several top-level elements, or with a
// non-Document child, has no single well-defined document container for
// assistive technology to anchor to.
//
// One finding. N/A when the document has no structure tree at all
// (UA-01-005 reports that). PDF/UA-2 only (veraPDF UA2:8.2.5.2-1); the
// PDF-2.0-namespace half of the clause is UA-01-008.
type SingleDocumentChild struct{}

func (SingleDocumentChild) ID() string { return "UA-01-015" }
func (SingleDocumentChild) Title() string {
	return "Structure tree root has a single Document element as its only child"
}
func (SingleDocumentChild) Category() engine.Category { return engine.CategoryStructure }
func (SingleDocumentChild) Severity() engine.Severity { return engine.SeverityError }
func (SingleDocumentChild) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (SingleDocumentChild) WCAG() []string            { return []string{"1.3.1"} }
func (SingleDocumentChild) Description() string {
	return "ISO 14289-2 §8.2.5.2 requires the /StructTreeRoot to contain a single Document structure element as its only child (ISO 32000-2:2020, Annex L; ISO/TS 32005). Multiple top-level elements, or a non-Document root child, leave the tree without one canonical document container."
}

func (c SingleDocumentChild) Run(doc model.Document) []engine.Finding {
	kids, hasRoot := doc.StructTreeRootKids()
	if !hasRoot {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}
	if len(kids) == 1 && kids[0].Type() == "Document" {
		return nil
	}

	msg := fmt.Sprintf("structure tree root contains %d element(s) instead of a single Document structure element", len(kids))
	if len(kids) > 0 {
		types := make([]string, 0, len(kids))
		for _, k := range kids {
			t := k.Type()
			if t == "" {
				t = "<untyped>"
			}
			types = append(types, t)
		}
		msg += " (child types: " + strings.Join(types, ", ") + ")"
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  msg,
		Hint:     "Wrap all top-level structure elements in a single Document element directly under /StructTreeRoot/K.",
		Location: &engine.Location{StructPath: "/StructTreeRoot"},
	}}
}

func init() { engine.Register(SingleDocumentChild{}) }
