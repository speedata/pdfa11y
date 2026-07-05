package headings

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// SingleHChild fails when a structure element has more than one direct child
// whose standard type is the unnumbered heading "H". ISO 14289-1 §7.4.4
// (veraPDF UA1:7.4.4-1) requires that each node in the tag tree contain at
// most one child H tag: the /H type declares "a heading" without a level, so
// two sibling H tags under one node give assistive technology two competing,
// unordered headings for the same section.
//
// The count is over the *standard* type of each direct child, so a custom type
// role-mapped to H is included, while the numbered types H1..H6 (and the PDF
// 2.0 generic Hn) are distinct types and are not counted here -- their nesting
// is governed by the sequencing rule UA-14-003.
//
// One finding per offending node. N/A when the document has no structure tree.
// PDF/UA-1 only: PDF/UA-2 documents use numbered headings (or Hn with /Lvl) and
// veraPDF scopes this rule to ISO 14289-1.
type SingleHChild struct{}

func (SingleHChild) ID() string                { return "UA-14-008" }
func (SingleHChild) Title() string             { return "A node has at most one child H (unnumbered heading) tag" }
func (SingleHChild) Category() engine.Category { return engine.CategoryHeadings }
func (SingleHChild) Severity() engine.Severity { return engine.SeverityError }
func (SingleHChild) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (SingleHChild) WCAG() []string            { return []string{"1.3.1"} }
func (SingleHChild) Description() string {
	return "ISO 14289-1 §7.4.4 requires each node in the tag tree to contain at most one child H (unnumbered heading) tag. Two sibling H tags under the same node present assistive technology with competing headings that have no defined order or level. Use the numbered heading types H1..H6 when a section needs more than one heading."
}

func (c SingleHChild) Run(doc model.Document) []engine.Finding {
	kids, hasRoot := doc.StructTreeRootKids()
	if !hasRoot {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}
	roleMap := doc.RoleMap()
	var findings []engine.Finding
	for _, k := range kids {
		c.walk(k, "/"+k.Type(), roleMap, &findings)
	}
	return findings
}

func (c SingleHChild) walk(elem model.StructElement, path string, roleMap map[string]string, findings *[]engine.Finding) {
	children := elem.Children()
	hCount := 0
	for _, ch := range children {
		if standardType(ch.Type(), roleMap) == "H" {
			hCount++
		}
	}
	if hCount > 1 {
		*findings = append(*findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("structure element at %s has %d direct child H tags, expected at most one", path, hCount),
			Hint:     "Give the section a single H tag, or use the numbered heading types H1..H6 for multiple headings.",
			Location: &engine.Location{Page: elem.Page(), StructPath: path},
		})
	}
	for _, ch := range children {
		c.walk(ch, path+"/"+ch.Type(), roleMap, findings)
	}
}

// standardType resolves a raw structure type to its standard PDF type by
// following the classic /RoleMap chain (with a cycle guard). A type that is
// already standard maps to itself; a chain that never reaches a standard type
// yields its last link. Only the classic /RoleMap is consulted, which is the
// mechanism PDF/UA-1 documents use.
func standardType(raw string, roleMap map[string]string) string {
	seen := map[string]bool{}
	t := raw
	for {
		if model.IsStandardStructureType(t) {
			return t
		}
		if seen[t] {
			return t
		}
		seen[t] = true
		next, ok := roleMap[t]
		if !ok {
			return t
		}
		t = next
	}
}

func init() { engine.Register(SingleHChild{}) }
