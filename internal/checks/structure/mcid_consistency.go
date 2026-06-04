package structure

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MCIDConsistency fails when the set of MCIDs declared by BDC
// operators in a page's content stream and the set of MCIDs claimed
// by the structure tree for that page disagree. Both directions are
// real problems:
//
//   - MCID in structure tree but not in content: the tree promises
//     content under a tag (Span/MCID, MCR{MCID,Pg}) but no BDC on
//     that page actually emits the ID. Screen readers read nothing.
//
//   - MCID in content but not in structure tree: the page has a
//     marked-content sequence that the tree does not reach. The
//     content is technically marked but invisible to the structure
//     tree -- which is a more subtle variant of "untagged real
//     content".
//
// One finding per page per mismatch direction. The two cases are
// reported separately so the user knows which side to fix.
type MCIDConsistency struct{}

func (MCIDConsistency) ID() string                { return "MH-02-003" }
func (MCIDConsistency) Title() string             { return "Content-stream MCIDs match the structure tree" }
func (MCIDConsistency) Category() engine.Category { return engine.CategoryStructure }
func (MCIDConsistency) Severity() engine.Severity { return engine.SeverityError }
func (MCIDConsistency) Spec() engine.Spec         { return engine.SpecBoth }
func (MCIDConsistency) WCAG() []string            { return []string{"1.3.1"} }
func (MCIDConsistency) Description() string {
	return "PDF/UA-1 §7.1 requires every marked-content sequence with a /MCID to be the target of a structure tree MCR, and every structure tree MCR to point at a real marked-content sequence. A mismatch in either direction breaks the link between the document's visual content and its semantic structure: screen readers either narrate a tag with no content underneath, or skip over content that exists but is not in the reading order."
}

func (c MCIDConsistency) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 || !anyMCIDActivity(pages) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no marked-content / structure-tree MCID links to inspect",
		}}
	}
	var findings []engine.Finding
	for _, p := range pages {
		missing := setDifference(p.StructTreeMCIDs, p.ContentMCIDs)
		orphan := setDifference(p.ContentMCIDs, p.StructTreeMCIDs)
		if len(missing) > 0 {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d: structure tree references MCID(s) %s, but the content stream does not declare them", p.Number, joinInts(missing)),
				Hint:     "Either emit BDC sequences with these MCIDs on the page, or remove the MCR entries from the structure tree.",
				Location: &engine.Location{Page: p.Number},
			})
		}
		if len(orphan) > 0 {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d: content stream declares MCID(s) %s, but no structure element references them", p.Number, joinInts(orphan)),
				Hint:     "Add an MCR (or bare integer K-entry) under a structure element so the marked-content sequence becomes part of the reading order. If the content is decorative, retag the BDC sequence as /Artifact instead.",
				Location: &engine.Location{Page: p.Number},
			})
		}
	}
	return findings
}

func anyMCIDActivity(pages []model.PageReport) bool {
	for _, p := range pages {
		if len(p.ContentMCIDs) > 0 || len(p.StructTreeMCIDs) > 0 {
			return true
		}
	}
	return false
}

func setDifference(a, b map[int]bool) []int {
	var out []int
	for id := range a {
		if !b[id] {
			out = append(out, id)
		}
	}
	sort.Ints(out)
	return out
}

func joinInts(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ", ")
}

func init() { engine.Register(MCIDConsistency{}) }
