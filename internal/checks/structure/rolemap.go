package structure

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// RoleMap fails when the structure tree contains a structure type that
// does not resolve to a standard type:
//
//   - in the default PDF namespace, a sui-generis (custom) type that is
//     neither one of the standard PDF structure types nor mapped to one
//     via /RoleMap -- "FirstParagraph", "BoxedText", "MyCallout" and the
//     like are meaningful only to the producer. A /RoleMap entry whose
//     target is an empty name counts as unmapped.
//   - in any other explicit namespace (ISO 32000-2 §14.8.6.2), a type
//     that the namespace's /RoleMapNS does not map, directly or
//     transitively, into the PDF 1.7, PDF 2.0 or MathML namespace. The
//     classic /RoleMap does not apply to elements in an explicit
//     namespace. MathML types are standard in their own namespace
//     (ISO 32000-2 §14.8.6.3) and need no mapping.
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
	return "PDF/UA-1 §7.1 (PDF/UA-2 §8.2.4) requires every structure element type to be either a standard PDF structure type or mapped to one: via the /RoleMap entry on StructTreeRoot, or for an element in an explicit namespace via that namespace's /RoleMapNS. Custom names that survive role-map resolution are opaque to assistive technology."
}

type occurrence struct {
	message string
	hint    string
	path    string
	page    int
	count   int
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

	// Stable order: by key. Without this, map iteration would shuffle
	// findings between runs and break golden-file tests.
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var findings []engine.Finding
	for _, k := range keys {
		u := seen[k]
		msg := u.message
		if u.count > 1 {
			msg += fmt.Sprintf(" (used %d times; first occurrence reported)", u.count)
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  msg,
			Hint:     u.hint,
			Location: &engine.Location{Page: u.page, StructPath: u.path},
		})
	}
	return findings
}

func (c RoleMap) walk(elem model.StructElement, path string, seen map[string]*occurrence) {
	if key, msg, hint := c.unmapped(elem); key != "" {
		if u, ok := seen[key]; ok {
			u.count++
		} else {
			seen[key] = &occurrence{message: msg, hint: hint, path: path, page: elem.Page(), count: 1}
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), seen)
	}
}

// unmapped returns a dedup key, message and hint when elem's type does
// not resolve to a standard type, or an empty key when it does.
func (c RoleMap) unmapped(elem model.StructElement) (key, msg, hint string) {
	raw := elem.RawType()
	if raw == "" {
		return "", "", ""
	}
	if !elem.BelongsToStandardNamespace() {
		ns := elem.Namespace()
		return ns + " " + raw,
			fmt.Sprintf("structure type %q in namespace %s is not role-mapped into the PDF 1.7, PDF 2.0 or MathML namespace", raw, ns),
			fmt.Sprintf("Add /%s to the /RoleMapNS of the namespace dictionary for %s, e.g. /%s [/P <PDF 2.0 namespace>]; pick the closest standard type for this content.", raw, ns, raw)
	}
	if !inDefaultPDFNamespace(elem) {
		return "", "", ""
	}
	t := elem.Type()
	switch {
	case t == "":
		return raw,
			fmt.Sprintf("structure type %q is role-mapped to an empty name in /RoleMap", raw),
			fmt.Sprintf("Map /%s to a standard structure type in the document's /RoleMap, e.g. /RoleMap << /%s /P >>.", raw, raw)
	case !standardStructType(t):
		return t,
			fmt.Sprintf("structure type %q is not a standard PDF tag and is not declared in /RoleMap", t),
			fmt.Sprintf("Add the mapping to the document's StructTreeRoot, e.g. /RoleMap << /%s /P >>; pick the closest standard type for this content.", t)
	}
	return "", "", ""
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
