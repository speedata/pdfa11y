package structure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// UntaggedContent fails when a page paints real document content
// outside any marked-content sequence. "Real content" means text
// showing (Tj/TJ/'/"), XObject invocation (Do), path painting
// (S/s/f/F/f*/B/B*/b/b*) and inline images (EI). PDF/UA-1 §7.1
// requires every such operator either to live inside a marked-content
// sequence linked into the structure tree, or to be explicitly tagged
// as /Artifact -- decorative content that screen readers should ignore.
//
// One finding per page that has any untagged operators, summarising
// the offending operators by kind and count. A finer per-operator
// listing would flood the report on broken pages.
type UntaggedContent struct{}

func (UntaggedContent) ID() string                { return "UA-14-001" }
func (UntaggedContent) Title() string             { return "Real content is inside a marked-content sequence or Artifact" }
func (UntaggedContent) Category() engine.Category { return engine.CategoryStructure }
func (UntaggedContent) Severity() engine.Severity { return engine.SeverityError }
func (UntaggedContent) Spec() engine.Spec         { return engine.SpecBoth }
func (UntaggedContent) WCAG() []string            { return []string{"1.3.1"} }
func (UntaggedContent) Description() string {
	return "PDF/UA-1 §7.1 requires every content-painting operator to either live inside a marked-content sequence (BDC.../EMC) wired into the structure tree, or to be explicitly tagged as an /Artifact so assistive technology knows to skip it. Untagged real content is invisible to the structure tree: a screen reader cannot present it and cannot place it in the reading order."
}

func (c UntaggedContent) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no page content available for inspection",
		}}
	}
	// If no page has any content-stream activity at all (no fonts
	// referenced, no MCIDs, no painting operators), the check is
	// vacuous -- typical for synthetic test fixtures.
	if !anyContentActivity(pages) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document pages carry no painting content -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, p := range pages {
		if len(p.UntaggedOps) == 0 {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("page %d has %d painting operator(s) outside any marked-content sequence (%s)", p.Number, len(p.UntaggedOps), summariseOps(p.UntaggedOps)),
			Hint:     "Wrap each painting operator in BDC.../EMC referenced by a structure element. If the content is purely decorative (page header/footer rules, watermarks, background art), tag it /Artifact BDC.../EMC instead.",
			Location: &engine.Location{Page: p.Number},
		})
	}
	return findings
}

func anyContentActivity(pages []model.PageReport) bool {
	for _, p := range pages {
		if len(p.UsedFonts) > 0 || len(p.ContentMCIDs) > 0 || len(p.UntaggedOps) > 0 {
			return true
		}
	}
	return false
}

// summariseOps returns a stable "Tj ×3, Do ×1" rendering of the
// operator histogram, sorted by operator name so message-equality
// tests are deterministic.
func summariseOps(ops []model.UntaggedOp) string {
	counts := map[string]int{}
	for _, op := range ops {
		counts[op.Operator]++
	}
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%s ×%d", n, counts[n]))
	}
	return strings.Join(parts, ", ")
}

func init() { engine.Register(UntaggedContent{}) }
