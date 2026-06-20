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
// Namespace-aware: elements that declare a non-default namespace
// via /NS (or inherit one through the /P chain) live in that
// namespace's type system, not the PDF default. ISO 32000-2
// §14.8.6.3 registers the W3C MathML namespace explicitly; tags
// like 'math', 'mi', 'mo' inside it are standard MathML, not
// custom PDF tags, and do not require /RoleMap. Same logic applies
// to any other registered namespace (LaTeX-project, custom XML
// vocabularies, etc.) -- the check only fires in the default PDF
// namespace, where ISO 32000 §14.8.4 defines the standard types.
//
// One finding per unique unmapped type, with the first occurrence's
// location and an occurrence count. Reporting every instance would
// flood the report for documents that use a custom tag dozens of
// times.
type RoleMap struct{}

func (RoleMap) ID() string                { return "UA-31-008" }
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
	if t != "" && inDefaultPDFNamespace(elem) && !standardStructType(t) {
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

// inDefaultPDFNamespace reports whether elem's namespace is the
// default PDF structure namespace (where ISO 32000 §14.8.4 defines
// the standard tag set). An element with no /NS at all, or one
// pointing at the standard PDF / PDF 2.0 namespace URIs, lives in
// that default; anything else (MathML, custom XML vocabularies)
// is governed by its own namespace's type system.
//
// The standard PDF structure namespace URIs are registered by
// ISO 32000-2 §14.8.6.3 / ISO/TS 32005:
//   - http://iso.org/pdf/ssn   (PDF 1.7 structure namespace)
//   - http://iso.org/pdf2/ssn  (PDF 2.0 structure namespace)
func inDefaultPDFNamespace(elem model.StructElement) bool {
	ns := elem.Namespace()
	if ns == "" {
		return true
	}
	switch ns {
	case "http://iso.org/pdf/ssn", "http://iso.org/pdf2/ssn":
		return true
	}
	return false
}

// standardStructType reports whether s is one of the PDF structure
// element types defined by ISO 32000-1, ISO 32000-2 or PDF/UA-2.
// The canonical set lives in package model so that role resolution
// (internal/pdf) and this check share a single source of truth.
func standardStructType(s string) bool {
	return model.IsStandardStructureType(s)
}

func init() { engine.Register(RoleMap{}) }
