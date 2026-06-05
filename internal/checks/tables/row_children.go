package tables

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// RowChildren fails for every TR whose direct children are not all
// TH or TD. PDF/UA-1 §7.5 requires the canonical Table -> ... -> TR
// -> (TH|TD) shape; a TR carrying paragraphs, spans or other generic
// containers signals that the author tagged the visual layout
// (rows-of-anything) rather than semantic table cells. Screen
// readers then either narrate the wrong structure or skip the cell
// entirely.
//
// One finding per offending TR. Empty TRs are allowed -- they are
// degenerate but not actively wrong.
type RowChildren struct{}

func (RowChildren) ID() string    { return "MH-15-004" }
func (RowChildren) Title() string { return "Table rows contain only TH or TD" }
func (RowChildren) Description() string {
	return "Per PDF/UA-1 §7.5 the only valid children of a TR are TH (header cell) and TD (data cell). Other tags between TR and the cell content break the table semantics: AT cannot map content to row/column positions and may drop or misattribute cells."
}
func (RowChildren) Category() engine.Category { return engine.CategoryTables }
func (RowChildren) Severity() engine.Severity { return engine.SeverityError }
func (RowChildren) Spec() engine.Spec         { return engine.SpecBoth }
func (RowChildren) WCAG() []string            { return []string{"1.3.1"} }

func (c RowChildren) Run(doc model.Document) []engine.Finding {
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
	rowCount := 0
	c.walk(root, "/"+root.Type(), &findings, &rowCount)
	if rowCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no TR (table row) structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c RowChildren) walk(elem model.StructElement, path string, out *[]engine.Finding, count *int) {
	if elem.Type() == "TR" {
		*count++
		bad := unexpectedChildren(elem)
		if len(bad) > 0 {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("TR has non-cell child(ren): %v", bad),
				Hint:     "Wrap each cell in TH (header) or TD (data). Move any per-cell content (P, Span, Figure, ...) inside the TH/TD.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out, count)
	}
}

// unexpectedChildren returns the (deduplicated) types of TR children
// that are neither TH nor TD. Returns an empty slice when the TR is
// well-formed.
func unexpectedChildren(tr model.StructElement) []string {
	seen := map[string]bool{}
	var out []string
	for _, child := range tr.Children() {
		t := child.Type()
		if t == "TH" || t == "TD" {
			continue
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func init() { engine.Register(RowChildren{}) }
