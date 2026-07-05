package tables

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// HeaderAssociation fails when a table containing header cells (TH) does not
// provide enough semantic information to determine which header pertains to
// which cell, as required by PDF/UA-2 §8.2.5.26 (ISO 32000-2 §14.8.5.7).
//
// Association can be expressed two ways:
//   - a /Scope attribute (Row/Column/Both) on every TH, or
//   - the explicit /Headers + /ID mechanism, where cells carry a /Headers
//     array of the /ID values of the header cells that apply to them.
//
// The check reports two defects, matching veraPDF's 8.2.5.26-5 and -6:
//   - a /Headers reference to an /ID that does not exist in the table
//     ("undefined header"), and
//   - a table with TH cells but no association mechanism at all (no /Scope on
//     the THs, no /ID or /Headers in use).
//
// This is a pragmatic reading of §14.8.5.7: it verifies that a mechanism is
// present and that explicit references resolve, rather than replaying the full
// header-resolution algorithm for every data cell. N/A when no table has
// header cells.
type HeaderAssociation struct{}

func (HeaderAssociation) ID() string                { return "UA-15-007" }
func (HeaderAssociation) Title() string             { return "Table header cells are associated with their data cells" }
func (HeaderAssociation) Category() engine.Category { return engine.CategoryTables }
func (HeaderAssociation) Severity() engine.Severity { return engine.SeverityError }
func (HeaderAssociation) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (HeaderAssociation) WCAG() []string            { return []string{"1.3.1"} }
func (HeaderAssociation) Description() string {
	return "PDF/UA-2 §8.2.5.26 (ISO 32000-2 §14.8.5.7) requires a table with header cells to make clear which header applies to which cell -- via a /Scope on every TH, or via /Headers arrays referencing header /ID values. Without this, assistive technology cannot announce the headers for a data cell."
}

func (c HeaderAssociation) Run(doc model.Document) []engine.Finding {
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
	tablesWithHeaders := 0
	c.walk(root, "/"+root.Type(), &tablesWithHeaders, &findings)
	if tablesWithHeaders == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no tables with header cells -- nothing to inspect",
		}}
	}
	return findings
}

func (c HeaderAssociation) walk(elem model.StructElement, path string, tablesWithHeaders *int, findings *[]engine.Finding) {
	if elem.Type() == "Table" {
		if f, applies, ok := c.inspect(elem, path); applies {
			*tablesWithHeaders++
			if ok {
				*findings = append(*findings, f)
			}
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), tablesWithHeaders, findings)
	}
}

// inspect analyses one table. applies is false when the table has no TH cells
// (the rule does not apply). When applies is true, ok reports whether a
// finding was produced.
func (c HeaderAssociation) inspect(table model.StructElement, path string) (f engine.Finding, applies, ok bool) {
	var cells []model.StructElement
	for _, row := range tableRows(table) {
		cells = append(cells, cellsOf(row)...)
	}

	ths := 0
	allTHScoped := true
	idSet := map[string]bool{}
	var headerRefs []string
	usesIDMechanism := false

	for _, cell := range cells {
		if id := cell.ID(); id != "" {
			idSet[id] = true
			usesIDMechanism = true
		}
		h := cell.HeaderIDs()
		if h != nil {
			usesIDMechanism = true
		}
		// Only data-cell (TD) references are validated for resolution: a
		// TH referencing another header, or an empty entry, is not a
		// data-to-header association (matching veraPDF's unknownHeaders).
		if cell.Type() == "TD" {
			for _, ref := range h {
				if ref != "" {
					headerRefs = append(headerRefs, ref)
				}
			}
		}
		if cell.Type() == "TH" {
			ths++
			if cell.Attribute("Scope") == "" {
				allTHScoped = false
			}
		}
	}

	if ths == 0 {
		return engine.Finding{}, false, false
	}

	// §8.2.5.26-6: /Headers references that resolve to no /ID in the table.
	var undefined []string
	for _, ref := range headerRefs {
		if !idSet[ref] {
			undefined = append(undefined, ref)
		}
	}
	if len(undefined) > 0 {
		sort.Strings(undefined)
		return engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("table at %s has /Headers referencing undefined header ID(s): %v", path, dedup(undefined)),
			Hint:     "Point each /Headers entry at the /ID of an existing header cell, or fix the header cell's /ID.",
			Location: &engine.Location{Page: table.Page(), StructPath: path},
		}, true, true
	}

	// §8.2.5.26-5: header cells present but no association mechanism.
	if !allTHScoped && !usesIDMechanism {
		return engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("table at %s has %d header cell(s) but no header association: no /Scope on the TH cells and no /Headers/ID mechanism in use", path, ths),
			Hint:     "Give each TH a /Scope (Row, Column or Both), or connect cells to their headers with /ID on the headers and /Headers arrays on the data cells.",
			Location: &engine.Location{Page: table.Page(), StructPath: path},
		}, true, true
	}

	return engine.Finding{}, true, false
}

// dedup returns the sorted unique elements of s (s is assumed already sorted).
func dedup(s []string) []string {
	out := s[:0:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

func init() { engine.Register(HeaderAssociation{}) }
