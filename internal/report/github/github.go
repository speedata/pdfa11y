// Package github renders check results for GitHub Actions: one workflow
// command (::error / ::warning / ::notice) per finding, which the runner
// turns into annotations on the workflow run, plus a Markdown job summary
// meant for the file named by $GITHUB_STEP_SUMMARY.
package github

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
)

// Document bundles one input file's outcome for the job summary. Err is
// set when the file could not be loaded; Results is empty then.
type Document struct {
	Path    string
	Results []engine.Result
	Err     error
}

// Write emits a one-line verdict for path followed by one workflow
// command per finding. N/A findings are skipped: they are not problems
// and would only crowd out the real annotations (GitHub shows a limited
// number per step).
func Write(w io.Writer, path string, results []engine.Result) {
	sum := engine.Summarize(results)
	fmt.Fprintf(w, "%s: %s   errors: %d · warnings: %d · suggestions: %d\n",
		path, sum.Verdict(), sum.Errors, sum.Warnings, sum.Infos)

	for _, r := range sortedByID(results) {
		title := r.Check.ID() + " " + r.Check.Title()
		for _, f := range r.Findings {
			cmd := command(f.Severity)
			if cmd == "" {
				continue
			}
			fmt.Fprintf(w, "::%s file=%s,title=%s::%s\n",
				cmd, escapeProperty(path), escapeProperty(title), escapeData(message(f)))
		}
	}
}

// WriteLoadError emits an error annotation for a file that could not be
// parsed at all.
func WriteLoadError(w io.Writer, path string, err error) {
	fmt.Fprintf(w, "::error file=%s,title=pdfa11y::%s\n",
		escapeProperty(path), escapeData("cannot load PDF: "+err.Error()))
}

// WriteSummary renders a Markdown overview of all documents: one table
// row per document, then per document a collapsible list of the checks
// that reported something.
func WriteSummary(w io.Writer, docs []Document) {
	fmt.Fprintln(w, "## pdfa11y")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Document | Verdict | Errors | Warnings | Suggestions |")
	fmt.Fprintln(w, "| --- | --- | ---: | ---: | ---: |")
	for _, d := range docs {
		if d.Err != nil {
			fmt.Fprintf(w, "| %s | ⛔ not loaded | – | – | – |\n", codeCell(d.Path))
			continue
		}
		sum := engine.Summarize(d.Results)
		fmt.Fprintf(w, "| %s | %s | %d | %d | %d |\n",
			codeCell(d.Path), verdictCell(sum.Verdict()), sum.Errors, sum.Warnings, sum.Infos)
	}
	fmt.Fprintln(w)

	for _, d := range docs {
		if d.Err != nil {
			fmt.Fprintf(w, "**%s**: %s\n\n", codeCell(d.Path), cell(d.Err.Error()))
			continue
		}
		var rows []engine.Result
		for _, r := range sortedByID(d.Results) {
			if reported(r) {
				rows = append(rows, r)
			}
		}
		if len(rows) == 0 {
			continue
		}
		fmt.Fprintf(w, "<details><summary><code>%s</code>: %d checks with findings</summary>\n\n",
			htmlEscape(d.Path), len(rows))
		fmt.Fprintln(w, "| Check | Title | State | Findings |")
		fmt.Fprintln(w, "| --- | --- | --- | ---: |")
		for _, r := range rows {
			fmt.Fprintf(w, "| %s | %s | %s | %d |\n",
				r.Check.ID(), cell(r.Check.Title()), stateCell(r), countReported(r))
		}
		fmt.Fprintln(w, "\n</details>")
		fmt.Fprintln(w)
	}
}

// command maps a finding severity to a workflow command name; "" means
// the finding is not annotated.
func command(s engine.Severity) string {
	switch s {
	case engine.SeverityError:
		return "error"
	case engine.SeverityWarning:
		return "warning"
	case engine.SeverityInfo:
		return "notice"
	}
	return ""
}

// message joins a finding's text, location and hint. The annotation
// body may span several lines.
func message(f engine.Finding) string {
	var b strings.Builder
	b.WriteString(f.Message)
	if loc := formatLocation(f.Location); loc != "" {
		b.WriteString("\nat: ")
		b.WriteString(loc)
	}
	if f.Hint != "" {
		b.WriteString("\nhint: ")
		b.WriteString(f.Hint)
	}
	return b.String()
}

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

func sortedByID(results []engine.Result) []engine.Result {
	rs := append([]engine.Result(nil), results...)
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].Check.ID() < rs[j].Check.ID() })
	return rs
}

// reported reports whether r has at least one finding that is annotated.
func reported(r engine.Result) bool { return countReported(r) > 0 }

func countReported(r engine.Result) int {
	n := 0
	for _, f := range r.Findings {
		if command(f.Severity) != "" {
			n++
		}
	}
	return n
}

func verdictCell(v engine.Verdict) string {
	switch v {
	case engine.VerdictFail:
		return "❌ FAIL"
	case engine.VerdictWarn:
		return "⚠️ WARN"
	}
	return "✅ PASS"
}

// stateCell labels a check row. A passing check only shows up in the
// table because of advisories, so it is called a suggestion there.
func stateCell(r engine.Result) string {
	switch r.State() {
	case engine.VerdictFail:
		return "FAIL"
	case engine.VerdictWarn:
		return "WARN"
	}
	return "suggestion"
}

// escapeData escapes the message part of a workflow command, following
// @actions/core's escapeData.
func escapeData(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	return strings.ReplaceAll(s, "\n", "%0A")
}

// escapeProperty escapes a key=value property of a workflow command,
// following @actions/core's escapeProperty.
func escapeProperty(s string) string {
	s = escapeData(s)
	s = strings.ReplaceAll(s, ":", "%3A")
	return strings.ReplaceAll(s, ",", "%2C")
}

// cell makes s safe for a single Markdown table cell.
func cell(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "|", `\|`)
}

// codeCell renders s as inline code inside a table cell. HTML <code>
// rather than backticks so that backticks in file names cannot break
// out of the span.
func codeCell(s string) string {
	return "<code>" + cell(htmlEscape(s)) + "</code>"
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
