// Package html renders check results as a single standalone HTML page
// suitable for sharing as an audit artefact. The template embeds its
// own CSS and the speedata wordmark/logo so the output file has no
// external dependencies.
package html

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
)

//go:embed logo.svg
var logoSVGRaw string

// logoSVG is the speedata logo with the XML declaration stripped, ready
// to inline into HTML. Sizing happens via CSS in the template so the
// 66x66 source scales to whatever footer height we choose.
var logoSVG = template.HTML(stripXMLDecl(logoSVGRaw))

func stripXMLDecl(s string) string {
	if strings.HasPrefix(s, "<?xml") {
		if i := strings.Index(s, "?>"); i > 0 {
			return strings.TrimLeft(s[i+2:], "\n\r\t ")
		}
	}
	return s
}

// formatLocation produces a single-line "page N, /Document/.../Figure"
// string from an engine.Location, or "" if no fields are populated.
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

// Build converts engine results for one document into the view shape
// the template expects. The full path is preserved only for the JSON
// counterpart; the HTML report shows the basename to avoid leaking
// directory structure when reports get shared.
func Build(path string, results []engine.Result) Document {
	sum := engine.Summarize(results)
	doc := Document{
		Filename: filepath.Base(path),
		Verdict:  sum.Verdict().String(),
		Summary: Summary{
			Passed:        sum.Passed,
			NotApplicable: sum.NotApplicable,
			Failed:        sum.Failed,
			Total:         sum.Total,
			Errors:        sum.Errors,
			Warnings:      sum.Warnings,
		},
	}
	doc.Categories = groupByCategory(results)
	return doc
}

// Write renders one or more documents as a standalone HTML page.
func Write(w io.Writer, docs []Document) error {
	return tmpl.Execute(w, pageData{Documents: docs, LogoSVG: logoSVG})
}

type pageData struct {
	Documents []Document
	LogoSVG   template.HTML
}

// Document is the per-PDF view passed to the template. Filename is
// rendered; the originating Path is intentionally not surfaced so
// shared reports do not leak directory structure.
type Document struct {
	Filename   string
	Verdict    string // "PASS" / "WARN" / "FAIL"
	Summary    Summary
	Categories []Category
}

// VerdictClass returns the lowercase verdict for CSS class composition.
func (d Document) VerdictClass() string { return strings.ToLower(d.Verdict) }

// Summary mirrors engine.Summary with template-friendly field types.
type Summary struct {
	Passed, NotApplicable, Failed, Total, Errors, Warnings int
}

// Category groups Result views under one Matterhorn heading.
type Category struct {
	Name    string
	Results []Result
}

// Result is the per-check view passed to the template.
type Result struct {
	ID          string
	Title       string
	Description string
	State       string // "PASS" / "WARN" / "FAIL"
	WCAG        string // comma-joined; empty when no WCAG mapping
	Findings    []Finding
}

// StateClass returns the lowercase state for CSS class composition.
// "N/A" needs the slash stripped to be a valid CSS class.
func (r Result) StateClass() string {
	switch r.State {
	case "N/A":
		return "na"
	}
	return strings.ToLower(r.State)
}

// OpenByDefault returns true for results whose <details> should be
// expanded on page load. Only FAIL is expanded; WARN and PASS stay
// collapsed so the page is calm at first glance and the reader can
// click into details when curious.
func (r Result) OpenByDefault() bool { return r.State == "FAIL" }

// Finding is the per-finding view passed to the template.
type Finding struct {
	Severity string
	Message  string
	Hint     string
	// Location is a pre-formatted human-readable location string, or
	// "" when the finding has no useful location info (e.g. catalog-
	// level checks).
	Location string
}

func groupByCategory(results []engine.Result) []Category {
	bucket := map[engine.Category][]engine.Result{}
	for _, r := range results {
		bucket[r.Check.Category()] = append(bucket[r.Check.Category()], r)
	}
	cats := make([]engine.Category, 0, len(bucket))
	for c := range bucket {
		cats = append(cats, c)
	}
	slices.Sort(cats)

	out := make([]Category, 0, len(cats))
	for _, c := range cats {
		rs := bucket[c]
		sort.Slice(rs, func(i, j int) bool { return rs[i].Check.ID() < rs[j].Check.ID() })
		view := Category{Name: string(c), Results: make([]Result, 0, len(rs))}
		for _, r := range rs {
			view.Results = append(view.Results, buildResult(r))
		}
		out = append(out, view)
	}
	return out
}

func buildResult(r engine.Result) Result {
	rv := Result{
		ID:          r.Check.ID(),
		Title:       r.Check.Title(),
		Description: r.Check.Description(),
		State:       r.State().String(),
		WCAG:        strings.Join(r.Check.WCAG(), ", "),
	}
	for _, f := range r.Findings {
		rv.Findings = append(rv.Findings, Finding{
			Severity: f.Severity.String(),
			Message:  f.Message,
			Hint:     f.Hint,
			Location: formatLocation(f.Location),
		})
	}
	return rv
}

var tmpl = template.Must(template.New("report").Parse(reportTemplate))

const reportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>PDF/UA Accessibility Report</title>
<style>
  :root {
    --pass: #1f7a3e;
    --warn: #a36500;
    --fail: #b3261e;
    --info: #1664c0;
    --muted: #6a6a6a;
    --rule: #e5e5e5;
    --surface: #fafafa;
    --text: #1f2328;
  }
  * { box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, system-ui, sans-serif;
    max-width: 64rem;
    margin: 2rem auto;
    padding: 0 1.25rem;
    color: var(--text);
    line-height: 1.55;
  }
  h1 { font-size: 1.6rem; margin: 0 0 .25rem; letter-spacing: -.01em; }
  .lede { color: var(--muted); margin-bottom: 2rem; }
  h2 { font-size: 1.25rem; margin: 2.5rem 0 .5rem; word-break: break-all; letter-spacing: -.005em; }
  h3 { font-size: .9rem; text-transform: uppercase; letter-spacing: .08em; color: var(--muted); margin: 1.75rem 0 .5rem; font-weight: 600; }

  .summary-card {
    display: flex;
    align-items: stretch;
    background: var(--surface);
    border-radius: 6px;
    overflow: hidden;
    margin: .5rem 0 1.25rem;
    border: 1px solid var(--rule);
  }
  .summary-verdict {
    flex: 0 0 auto;
    padding: .6rem 1.1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1rem;
    font-weight: 700;
    letter-spacing: .06em;
    color: #fff;
    min-width: 5rem;
  }
  .summary-verdict.verdict-pass { background: var(--pass); }
  .summary-verdict.verdict-warn { background: var(--warn); }
  .summary-verdict.verdict-fail { background: var(--fail); }
  .summary-stats {
    display: flex;
    flex-wrap: wrap;
    gap: 0 2rem;
    align-items: center;
    padding: .35rem 1.1rem;
    flex: 1 1 auto;
  }
  .summary-stats .stat { display: flex; flex-direction: column; padding: .2rem 0; }
  .summary-stats .stat-value { font-size: 1.05rem; font-weight: 600; color: var(--text); line-height: 1.2; }
  .summary-stats .stat-label { font-size: .75rem; color: var(--muted); letter-spacing: .02em; }

  details.check {
    margin: .15rem 0;
    border-bottom: 1px solid var(--rule);
  }
  details.check:last-of-type { border-bottom: none; }
  details.check summary {
    padding: .45rem .25rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: .7rem;
    list-style: none;
    flex-wrap: wrap;
  }
  details.check summary::-webkit-details-marker { display: none; }
  details.check summary::marker { content: ""; }
  details.check summary:hover { background: rgba(0,0,0,.015); }

  /* Three-column state indicator: a single coloured dot's horizontal
     position (left/middle/right) signals FAIL/WARN/PASS at a glance.
     The two empty cells reserve grid space so the dots line up
     vertically across rows. */
  .state-grid {
    display: grid;
    grid-template-columns: 18px 18px 18px;
    align-items: center;
    flex: 0 0 auto;
  }
  .state-grid .state-cell { width: 12px; height: 12px; border-radius: 50%; justify-self: center; }
  .state-fail .state-cell.fail { background: var(--fail); }
  .state-warn .state-cell.warn { background: var(--warn); }
  .state-pass .state-cell.pass { background: var(--pass); opacity: .55; }
  .state-na   .state-cell      { background: #c8c8c8; opacity: .4; }
  details.check.check-na .check-title { color: #888; font-style: italic; }

  .check-id { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--muted); font-size: .8rem; flex: 0 0 auto; }
  .check-title { font-weight: 500; flex: 1 1 auto; color: var(--text); }
  details.check.check-pass .check-title { color: #444; font-weight: 400; }
  .wcag { color: var(--muted); font-size: .75rem; flex: 0 0 auto; }

  .check-body {
    padding: .35rem 0 .75rem 4.7rem;
  }
  .description { color: var(--muted); font-size: .9rem; margin: 0 0 .4rem; line-height: 1.5; }
  .finding { margin: .45rem 0 0; padding: .15rem 0 .15rem .75rem; border-left: 3px solid; }
  .finding.severity-error   { border-color: var(--fail); }
  .finding.severity-warning { border-color: var(--warn); }
  .finding.severity-info    { border-color: var(--info); }
  .finding-label { font-size: .7rem; font-weight: 700; letter-spacing: .05em; text-transform: uppercase; margin-right: .35rem; }
  .severity-error .finding-label   { color: var(--fail); }
  .severity-warning .finding-label { color: var(--warn); }
  .severity-info .finding-label    { color: var(--info); }
  .finding-hint { color: var(--muted); font-size: .88rem; font-style: italic; margin-top: .25rem; }

  footer { margin-top: 4rem; padding-top: 1rem; border-top: 1px solid var(--rule); color: var(--muted); font-size: .85rem; text-align: center; }
  footer a { color: #444; text-decoration: none; border-bottom: 1px solid #ccc; }
  footer a:hover { color: var(--text); border-bottom-color: #888; }
  footer .brand { display: inline-flex; align-items: center; gap: .35rem; font-weight: 600; letter-spacing: .02em; color: #444; vertical-align: middle; }
  footer .brand svg { width: 1.3em; height: 1.3em; }
</style>
</head>
<body>
<header>
  <h1>PDF/UA Accessibility Report</h1>
  <div class="lede">Matterhorn-protocol check results, per document. Click any check to see the full explanation.</div>
</header>
{{range .Documents}}
<article>
  <h2>{{.Filename}}</h2>
  <div class="summary-card">
    <div class="summary-verdict verdict-{{.VerdictClass}}">{{.Verdict}}</div>
    <div class="summary-stats">
      <div class="stat"><span class="stat-value">{{.Summary.Passed}}/{{.Summary.Total}}</span><span class="stat-label">checks passed</span></div>
      {{if .Summary.NotApplicable}}<div class="stat"><span class="stat-value">{{.Summary.NotApplicable}}</span><span class="stat-label">not applicable</span></div>{{end}}
      <div class="stat"><span class="stat-value">{{.Summary.Errors}}</span><span class="stat-label">error{{if ne .Summary.Errors 1}}s{{end}}</span></div>
      <div class="stat"><span class="stat-value">{{.Summary.Warnings}}</span><span class="stat-label">warning{{if ne .Summary.Warnings 1}}s{{end}}</span></div>
    </div>
  </div>
  {{range .Categories}}
    <h3>{{.Name}}</h3>
    {{range .Results}}
      <details class="check check-{{.StateClass}}"{{if .OpenByDefault}} open{{end}}>
        <summary>
          <span class="state-grid state-{{.StateClass}}" aria-label="{{.State}}">
            <span class="state-cell fail"></span>
            <span class="state-cell warn"></span>
            <span class="state-cell pass"></span>
          </span>
          <span class="check-id">{{.ID}}</span>
          <span class="check-title">{{.Title}}</span>
          {{if .WCAG}}<span class="wcag">WCAG {{.WCAG}}</span>{{end}}
        </summary>
        <div class="check-body">
          {{if .Description}}<p class="description">{{.Description}}</p>{{end}}
          {{range .Findings}}
            <div class="finding severity-{{.Severity}}">
              <div><span class="finding-label">{{.Severity}}</span>{{.Message}}</div>
              {{if .Location}}<div class="finding-loc">at: {{.Location}}</div>{{end}}
              {{if .Hint}}<div class="finding-hint">{{.Hint}}</div>{{end}}
            </div>
          {{end}}
        </div>
      </details>
    {{end}}
  {{end}}
</article>
{{end}}
<footer>
  Generated by <a href="https://github.com/speedata/pdfa11y">pdfa11y</a>
  &middot; made by <a href="https://www.speedata.de" class="brand">{{.LogoSVG}}<span>speedata</span></a>
</footer>
</body>
</html>
`
