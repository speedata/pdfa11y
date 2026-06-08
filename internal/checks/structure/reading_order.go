package structure

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ReadingOrder fires when the structure-tree linearisation walks the
// page in a markedly different order than the visual layout. The
// content layer (Matterhorn G4 family — "Glyphs are tagged in
// reading order") is a Matterhorn human check, so this check is a
// pragmatic heuristic, not a definitive verdict.
//
// Heuristic (two passes, both per page):
//
//  1. Cluster MCID bounding boxes into columns by MinX. Any gap of
//     more than columnGapThreshold between consecutive sorted MinX
//     values starts a new column. A single-column page is one
//     cluster; a typical two-column layout is two.
//
//  2. Walk the structure-tree linearisation. Two patterns are
//     flagged:
//
//     - within-column upward jump: the next intended MCID sits more
//       than withinColumnYFraction × column-height (with an absolute
//       floor) above the previous one in the *same* column.
//
//     - column step backwards: the next intended MCID lives in a
//       column to the left of the previous one. Left-to-right
//       reading order is the assumption; right-to-left documents
//       will produce false positives here.
//
// Severity Warning, not Error: Matterhorn lists reading order as a
// human check, and the heuristics above are conservative
// approximations. Multi-column heading bands often legitimately
// span both clusters, so the first MCID after such a band can
// trip column-step-backwards — treat findings as suggestive.
type ReadingOrder struct{}

func (ReadingOrder) ID() string                { return "MH-09-G4-001" }
func (ReadingOrder) Title() string             { return "Glyphs are tagged in reading order" }
func (ReadingOrder) Category() engine.Category { return engine.CategoryStructure }
func (ReadingOrder) Severity() engine.Severity { return engine.SeverityWarning }
func (ReadingOrder) Spec() engine.Spec         { return engine.SpecBoth }
func (ReadingOrder) WCAG() []string            { return []string{"1.3.2"} }
func (ReadingOrder) Description() string {
	return "PDF/UA-1 §7.2 / Matterhorn G4 require glyphs to be tagged in the same order they read visually. This check clusters each page's MCID bounding boxes into columns (gaps in MinX of more than 50pt start a new column), then walks the structure-tree linearisation: it flags upward jumps within a column and steps backwards across columns. The check is a heuristic and runs at Warning severity; right-to-left documents and content that spans multiple columns can produce false positives."
}

// columnGapThreshold is the smallest gap between two adjacent MinX
// values that starts a new column cluster (in PDF user-space points).
// 50pt is ~0.7cm — narrow enough that an indented paragraph stays in
// its parent column, wide enough that a two-column body always
// resolves to two clusters.
const columnGapThreshold = 50.0

// withinColumnYFraction is the column-height fraction an upward step
// in MaxY between two intended MCIDs in the *same* column must clear
// before it is flagged. Coupled with withinColumnYFloor below to
// guard against tiny columns silently relaxing the rule.
const withinColumnYFraction = 0.2

// withinColumnYFloor is the absolute minimum threshold for the
// within-column jump check. Even on a very short column the jump
// must be larger than this to register, so paragraph-spacing noise
// in single-column flow does not trip the check.
const withinColumnYFloor = 30.0

func (c ReadingOrder) Run(doc model.Document) []engine.Finding {
	order, err := doc.StructTreeOrder()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot linearise structure tree: " + err.Error(),
		}}
	}
	if len(order) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no MCID leaves in the structure tree -- nothing to inspect",
		}}
	}

	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	boxesByPage := make(map[int]map[int]model.Rect, len(pages))
	for _, p := range pages {
		boxesByPage[p.Number] = p.MCIDBoxes
	}

	byPage := map[int][]model.ReadingOrderEntry{}
	pageOrder := []int{}
	for _, e := range order {
		if e.Page == 0 {
			continue
		}
		if _, seen := byPage[e.Page]; !seen {
			pageOrder = append(pageOrder, e.Page)
		}
		byPage[e.Page] = append(byPage[e.Page], e)
	}
	sort.Ints(pageOrder)

	var findings []engine.Finding
	for _, pageNum := range pageOrder {
		entries := byPage[pageNum]
		boxes := boxesByPage[pageNum]
		findings = append(findings, c.checkPage(pageNum, entries, boxes)...)
	}
	return findings
}

// checkPage applies the column-aware heuristic to one page. Returns
// at most one finding per page: a flood of per-MCID complaints would
// drown a real report and the first violation usually points at the
// authoring error.
func (c ReadingOrder) checkPage(pageNum int, entries []model.ReadingOrderEntry, boxes map[int]model.Rect) []engine.Finding {
	if len(boxes) == 0 {
		return nil
	}
	columnOf := clusterColumns(boxes)
	if len(columnOf) == 0 {
		return nil
	}

	// Per-column extents so the within-column threshold can scale to
	// the actual column height. Falls back to the global block extent
	// for single-column pages.
	colMinY := map[int]float64{}
	colMaxY := map[int]float64{}
	for id, col := range columnOf {
		b := boxes[id]
		if y, ok := colMinY[col]; !ok || b.MinY < y {
			colMinY[col] = b.MinY
		}
		if y, ok := colMaxY[col]; !ok || b.MaxY > y {
			colMaxY[col] = b.MaxY
		}
	}

	var prev model.ReadingOrderEntry
	var prevTop float64
	prevCol := -1
	for _, e := range entries {
		box, ok := boxes[e.MCID]
		if !ok || box.Empty() {
			continue
		}
		col, hasCol := columnOf[e.MCID]
		if !hasCol {
			continue
		}
		top := box.MaxY

		if prevCol >= 0 {
			if col < prevCol {
				return []engine.Finding{{
					CheckID:  c.ID(),
					Severity: engine.SeverityWarning,
					Message: fmt.Sprintf("page %d: %s (MCID %d) is tagged after %s (MCID %d), but lives in a column to its left",
						pageNum, e.StructPath, e.MCID, prev.StructPath, prev.MCID),
					Hint:     "Reading order should advance left-to-right across columns. A backwards column step usually means the structure tree groups the columns in the wrong order.",
					Location: &engine.Location{Page: pageNum, StructPath: e.StructPath},
				}}
			}
			if col == prevCol {
				threshold := (colMaxY[col] - colMinY[col]) * withinColumnYFraction
				if threshold < withinColumnYFloor {
					threshold = withinColumnYFloor
				}
				if top-prevTop > threshold {
					return []engine.Finding{{
						CheckID:  c.ID(),
						Severity: engine.SeverityWarning,
						Message: fmt.Sprintf("page %d: %s (MCID %d) is tagged after %s (MCID %d), but lies %.0fpt above it in the same column",
							pageNum, e.StructPath, e.MCID, prev.StructPath, prev.MCID, top-prevTop),
						Hint:     "Within a column, the structure tree should walk top-to-bottom. Re-tag the elements so the intended next paragraph follows the previous one geometrically.",
						Location: &engine.Location{Page: pageNum, StructPath: e.StructPath},
					}}
				}
			}
		}
		prev = e
		prevTop = top
		prevCol = col
	}
	return nil
}

// clusterColumns groups MCIDs into column buckets by MinX. Boxes are
// sorted by MinX and a new bucket starts whenever the gap to the
// previous MinX exceeds columnGapThreshold. The result maps each
// MCID to a zero-based column index (left-to-right). Empty rects are
// skipped — they carry no spatial information.
func clusterColumns(boxes map[int]model.Rect) map[int]int {
	type entry struct {
		mcid int
		x    float64
	}
	var es []entry
	for id, b := range boxes {
		if b.Empty() {
			continue
		}
		es = append(es, entry{id, b.MinX})
	}
	if len(es) == 0 {
		return nil
	}
	sort.Slice(es, func(i, j int) bool { return es[i].x < es[j].x })
	out := make(map[int]int, len(es))
	col := 0
	out[es[0].mcid] = col
	for i := 1; i < len(es); i++ {
		if es[i].x-es[i-1].x > columnGapThreshold {
			col++
		}
		out[es[i].mcid] = col
	}
	return out
}

func init() { engine.Register(ReadingOrder{}) }
