package structure

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ParentEntry fails when a structure element dictionary omits its /P (parent)
// entry. Both PDF/UA specs require every structure element to carry /P: ISO
// 14289-1 §7.1 (via ISO 32000-1 Table 355) and ISO 14289-2 §8.2.1 (via ISO
// 32000-2:2020, 14.7.2, Table 323). The parent back-reference is what makes the
// structure hierarchy a doubly linked tree, so a missing /P breaks upward
// navigation for assistive technology and structure-aware tooling.
//
// One finding per offending element. N/A when the document has no structure
// tree. Covers veraPDF UA1:7.1-12 and UA2:8.2.1-2.
type ParentEntry struct{}

func (ParentEntry) ID() string                { return "UA-01-014" }
func (ParentEntry) Title() string             { return "Every structure element has a /P (parent) entry" }
func (ParentEntry) Category() engine.Category { return engine.CategoryStructure }
func (ParentEntry) Severity() engine.Severity { return engine.SeverityError }
func (ParentEntry) Spec() engine.Spec         { return engine.SpecBoth }
func (ParentEntry) WCAG() []string            { return []string{"1.3.1"} }
func (ParentEntry) Description() string {
	return "PDF/UA-1 §7.1 (ISO 32000-1 Table 355) and PDF/UA-2 §8.2.1 (ISO 32000-2:2020, 14.7.2, Table 323) require every structure element dictionary to contain the /P (parent) entry. The parent reference makes the structure tree navigable in both directions; an element without /P is detached from its ancestors and cannot be located from the leaf upward."
}

func (c ParentEntry) Run(doc model.Document) []engine.Finding {
	kids, hasRoot := doc.StructTreeRootKids()
	if !hasRoot {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}
	if len(kids) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "structure tree root has no structure-element children -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, k := range kids {
		c.walk(k, "/"+k.Type(), &findings)
	}
	return findings
}

func (c ParentEntry) walk(elem model.StructElement, path string, findings *[]engine.Finding) {
	if !elem.HasParent() {
		*findings = append(*findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("structure element at %s has no /P (parent) entry", path),
			Hint:     "Add a /P entry pointing at the element's parent structure element (or the StructTreeRoot for the Document element).",
			Location: &engine.Location{Page: elem.Page(), StructPath: path},
		})
	}
	for _, ch := range elem.Children() {
		c.walk(ch, path+"/"+ch.Type(), findings)
	}
}

func init() { engine.Register(ParentEntry{}) }
