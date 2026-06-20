// Package terminal renders check results to stdout as a human-readable
// report card.
package terminal

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
)

// Options control formatting of the terminal report.
type Options struct {
	ShowWCAG bool // include WCAG mapping next to each finding
	Color    bool // ANSI colors (reserved for later)
}

// Write renders a single document's results.
func Write(w io.Writer, path string, results []engine.Result, opts Options) {
	sum := engine.Summarize(results)

	fmt.Fprintf(w, "── %s ───────────────────────────────\n", path)
	naPart := ""
	if sum.NotApplicable > 0 {
		naPart = fmt.Sprintf(" · %d n/a", sum.NotApplicable)
	}
	fmt.Fprintf(w, "Verdict: %s   %d checks: %d pass%s · %d fail   errors: %d · warnings: %d · suggestions: %d\n\n",
		sum.Verdict(), sum.Total, sum.Passed, naPart, sum.Failed, sum.Errors, sum.Warnings, sum.Infos)

	for _, r := range groupByCategory(results) {
		fmt.Fprintf(w, "[%s]\n", r.category)
		for _, res := range r.results {
			writeResult(w, res, opts)
		}
		fmt.Fprintln(w)
	}
}

type categoryGroup struct {
	category engine.Category
	results  []engine.Result
}

func groupByCategory(results []engine.Result) []categoryGroup {
	bucket := map[engine.Category][]engine.Result{}
	for _, r := range results {
		bucket[r.Check.Category()] = append(bucket[r.Check.Category()], r)
	}
	cats := make([]engine.Category, 0, len(bucket))
	for c := range bucket {
		cats = append(cats, c)
	}
	slices.Sort(cats)

	groups := make([]categoryGroup, 0, len(cats))
	for _, c := range cats {
		rs := bucket[c]
		sort.Slice(rs, func(i, j int) bool { return rs[i].Check.ID() < rs[j].Check.ID() })
		groups = append(groups, categoryGroup{category: c, results: rs})
	}
	return groups
}

func writeResult(w io.Writer, r engine.Result, opts Options) {
	wcag := ""
	if opts.ShowWCAG && len(r.Check.WCAG()) > 0 {
		wcag = "  WCAG " + strings.Join(r.Check.WCAG(), ", ")
	}
	fmt.Fprintf(w, "  %s  %s  %s%s\n", r.State(), r.Check.ID(), r.Check.Title(), wcag)
	for _, f := range r.Findings {
		// N/A findings are mode-information, not violations -- print
		// them inline as a short note and skip the hint/location
		// machinery, which would just be empty.
		if f.Severity == engine.SeverityNotApplicable {
			fmt.Fprintf(w, "        ↳ n/a: %s\n", f.Message)
			continue
		}
		fmt.Fprintf(w, "        ↳ %s: %s\n", f.Severity, f.Message)
		if loc := formatLocation(f.Location); loc != "" {
			fmt.Fprintf(w, "          at:   %s\n", loc)
		}
		if f.Hint != "" {
			fmt.Fprintf(w, "          hint: %s\n", f.Hint)
		}
	}
}

// formatLocation returns a single-line human-readable rendering of
// loc, or the empty string when no fields are populated. Used by the
// terminal reporter; the HTML reporter has its own template formatting.
func formatLocation(loc *engine.Location) string {
	if loc == nil {
		return ""
	}
	var parts []string
	if loc.Page > 0 {
		parts = append(parts, fmt.Sprintf("page %d", loc.Page))
	}
	if loc.StructPath != "" {
		parts = append(parts, loc.StructPath)
	}
	if loc.ObjectNumber > 0 {
		parts = append(parts, fmt.Sprintf("obj %d", loc.ObjectNumber))
	}
	return strings.Join(parts, ", ")
}
