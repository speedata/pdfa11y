package tables

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Geometry fails when a table is not "regular": its cells overlap, or its
// rows do not all span the same number of columns / its columns do not all
// span the same number of rows once /RowSpan and /ColSpan are taken into
// account. It implements the table-regularity requirements shared by
// PDF/UA-1 §7.2 (Table 7.2-15/41/42/43) and PDF/UA-2 §8.2.5.26 (tests 1-4).
//
// The check builds the cell grid the way a viewer would: each TH/TD is placed
// left-to-right into the next free column of its row, reserving ColSpan
// columns and RowSpan rows. A regular table fills an R×C rectangle exactly
// once. A slot covered twice is an intersection (§8.2.5.26-1); a slot left
// empty, or a row/column that comes up short, is an irregular shape
// (§8.2.5.26-2/-3/-4).
//
// Header-to-cell association (§8.2.5.26-5/-6) is a separate concern and not
// covered here. One finding per defective table. N/A when the document has no
// tables.
type Geometry struct{}

func (Geometry) ID() string                { return "UA-15-006" }
func (Geometry) Title() string             { return "Tables are regular (no overlaps, consistent rows and columns)" }
func (Geometry) Category() engine.Category { return engine.CategoryTables }
func (Geometry) Severity() engine.Severity { return engine.SeverityError }
func (Geometry) Spec() engine.Spec         { return engine.SpecBoth }
func (Geometry) WCAG() []string            { return []string{"1.3.1"} }
func (Geometry) Description() string {
	return "PDF/UA-1 §7.2 and PDF/UA-2 §8.2.5.26 require tables to be regular: cells must not overlap, every row must span the same number of columns and every column the same number of rows once /RowSpan and /ColSpan are applied. Irregular tables cannot be mapped to a reliable row/column model by assistive technology."
}

func (c Geometry) Run(doc model.Document) []engine.Finding {
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
	tables := 0
	c.walk(root, "/"+root.Type(), &tables, &findings)
	if tables == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no tables -- nothing to inspect",
		}}
	}
	return findings
}

func (c Geometry) walk(elem model.StructElement, path string, tables *int, findings *[]engine.Finding) {
	if elem.Type() == "Table" {
		*tables++
		if f, ok := c.inspect(elem, path); ok {
			*findings = append(*findings, f)
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), tables, findings)
	}
}

// inspect builds the cell grid for one table and returns a finding if it is
// irregular. The bool is false when the table is regular (or has no rows).
func (c Geometry) inspect(table model.StructElement, path string) (engine.Finding, bool) {
	rows := tableRows(table)
	if len(rows) == 0 {
		return engine.Finding{}, false
	}

	occ := map[[2]int]int{} // (row, col) -> number of cells covering it
	overlap := false
	maxCol := 0
	maxRow := len(rows)

	for r, row := range rows {
		col := 0
		for _, cell := range cellsOf(row) {
			// Skip columns already taken by a RowSpan from a row above.
			for occ[[2]int{r, col}] > 0 {
				col++
			}
			rs := spanOf(cell, "RowSpan")
			cs := spanOf(cell, "ColSpan")
			for dr := range rs {
				for dc := range cs {
					key := [2]int{r + dr, col + dc}
					if occ[key] > 0 {
						overlap = true
					}
					occ[key]++
					if key[0]+1 > maxRow {
						maxRow = key[0] + 1
					}
				}
			}
			col += cs
			if col > maxCol {
				maxCol = col
			}
		}
	}

	// A regular table fills the maxRow×maxCol rectangle exactly once.
	gaps := 0
	for r := range maxRow {
		for cc := range maxCol {
			switch occ[[2]int{r, cc}] {
			case 0:
				gaps++
			case 1:
			default:
				overlap = true
			}
		}
	}
	rowspanOverflow := maxRow > len(rows)

	if !overlap && gaps == 0 && !rowspanOverflow {
		return engine.Finding{}, false
	}

	msg := fmt.Sprintf("table at %s is not regular", path)
	switch {
	case overlap:
		msg += ": cells overlap (a RowSpan/ColSpan makes two cells cover the same grid position)"
	case rowspanOverflow:
		msg += fmt.Sprintf(": a RowSpan extends past the last row (grid spans %d rows for %d TR elements)", maxRow, len(rows))
	default:
		msg += fmt.Sprintf(": rows do not all span the same %d columns (RowSpan/ColSpan leave %d empty grid cells)", maxCol, gaps)
	}
	return engine.Finding{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  msg,
		Hint:     "Make the table rectangular: every row must cover the same number of columns and every column the same number of rows once /RowSpan and /ColSpan are counted. Check for missing cells or off-by-one span values.",
		Location: &engine.Location{Page: table.Page(), StructPath: path},
	}, true
}

// tableRows returns a table's rows in document order, descending through
// THead / TBody / TFoot groupings but not into nested tables.
func tableRows(table model.StructElement) []model.StructElement {
	var rows []model.StructElement
	for _, ch := range table.Children() {
		switch ch.Type() {
		case "TR":
			rows = append(rows, ch)
		case "THead", "TBody", "TFoot":
			for _, gch := range ch.Children() {
				if gch.Type() == "TR" {
					rows = append(rows, gch)
				}
			}
		}
	}
	return rows
}

// cellsOf returns the TH/TD children of a row in order.
func cellsOf(row model.StructElement) []model.StructElement {
	var cells []model.StructElement
	for _, ch := range row.Children() {
		if t := ch.Type(); t == "TH" || t == "TD" {
			cells = append(cells, ch)
		}
	}
	return cells
}

// spanOf reads /RowSpan or /ColSpan, defaulting to 1 (the PDF default) when
// absent, non-integer, or non-positive.
func spanOf(cell model.StructElement, name string) int {
	if v, ok := cell.AttributeInt(name); ok && v >= 1 {
		return v
	}
	return 1
}

func init() { engine.Register(Geometry{}) }
