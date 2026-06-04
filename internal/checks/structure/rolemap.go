package structure

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// RoleMap fails when the structure tree contains a sui-generis
// (custom) structure type that is neither one of the standard PDF
// structure types nor declared in /RoleMap on StructTreeRoot. Without
// a mapping, assistive technology has no way to know what role the
// element plays -- "FirstParagraph", "BoxedText", "MyCallout" and
// the like are meaningful only to the producer.
//
// One finding per unique unmapped type, with the first occurrence's
// location and an occurrence count. Reporting every instance would
// flood the report for documents that use a custom tag dozens of
// times.
type RoleMap struct{}

func (RoleMap) ID() string                { return "MH-31-008" }
func (RoleMap) Title() string             { return "Custom structure types are mapped to standard types" }
func (RoleMap) Category() engine.Category { return engine.CategoryStructure }
func (RoleMap) Severity() engine.Severity { return engine.SeverityError }
func (RoleMap) Spec() engine.Spec         { return engine.SpecBoth }
func (RoleMap) WCAG() []string            { return []string{"1.3.1"} }
func (RoleMap) Description() string {
	return "PDF/UA-1 §7.1 (PDF/UA-2 §8.2.4) requires every structure element type to be either a standard PDF structure type or mapped to one via the /RoleMap entry on StructTreeRoot. Custom names that survive role-map resolution are opaque to assistive technology."
}

type occurrence struct {
	path  string
	page  int
	count int
}

func (c RoleMap) Run(doc model.Document) []engine.Finding {
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

	seen := map[string]*occurrence{}
	c.walk(root, "/"+root.Type(), seen)

	// Stable order: by type name. Without this, map iteration would
	// shuffle findings between runs and break golden-file tests.
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	var findings []engine.Finding
	for _, name := range names {
		u := seen[name]
		msg := fmt.Sprintf("structure type %q is not a standard PDF tag and is not declared in /RoleMap", name)
		if u.count > 1 {
			msg += fmt.Sprintf(" (used %d times; first occurrence reported)", u.count)
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  msg,
			Hint:     fmt.Sprintf("Add the mapping to the document's StructTreeRoot, e.g. /RoleMap << /%s /P >>; pick the closest standard type for this content.", name),
			Location: &engine.Location{Page: u.page, StructPath: u.path},
		})
	}
	return findings
}

func (c RoleMap) walk(elem model.StructElement, path string, seen map[string]*occurrence) {
	t := elem.Type()
	if t != "" && !standardStructType(t) {
		if u, ok := seen[t]; ok {
			u.count++
		} else {
			seen[t] = &occurrence{path: path, page: elem.Page(), count: 1}
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), seen)
	}
}

// standardStructType reports whether s is one of the PDF structure
// element types defined by ISO 32000-1, ISO 32000-2 or PDF/UA-2.
// "Document" appears at the root of every tagged PDF; the rest spans
// grouping, block, inline, illustration and table categories.
func standardStructType(s string) bool {
	_, ok := standardTypes[s]
	return ok
}

var standardTypes = map[string]struct{}{
	// Grouping elements
	"Document":  {},
	"Part":      {},
	"Art":       {},
	"Sect":      {},
	"Div":       {},
	"BlockQuote": {},
	"Caption":   {},
	"TOC":       {},
	"TOCI":      {},
	"Index":     {},
	"NonStruct": {},
	"Private":   {},
	// PDF 2.0 grouping additions
	"DocumentFragment": {},
	"Aside":            {},
	"Title":            {},
	// Block-level
	"P":     {},
	"H":     {},
	"H1":    {},
	"H2":    {},
	"H3":    {},
	"H4":    {},
	"H5":    {},
	"H6":    {},
	"Hn":    {}, // PDF 2.0 generic heading (level via /Lvl)
	"L":     {},
	"LI":    {},
	"Lbl":   {},
	"LBody": {},
	// Inline-level
	"Span":      {},
	"Quote":     {},
	"Note":      {},
	"Reference": {},
	"BibEntry":  {},
	"Code":      {},
	"Link":      {},
	"Annot":     {},
	"Ruby":      {},
	"RB":        {},
	"RT":        {},
	"RP":        {},
	"Warichu":   {},
	"WT":        {},
	"WP":        {},
	// PDF 2.0 inline additions
	"Em":     {},
	"Strong": {},
	"Sub":    {},
	// Illustration
	"Figure":  {},
	"Formula": {},
	"Form":    {},
	// Tables
	"Table": {},
	"TR":    {},
	"TH":    {},
	"TD":    {},
	"THead": {},
	"TBody": {},
	"TFoot": {},
}

func init() { engine.Register(RoleMap{}) }
