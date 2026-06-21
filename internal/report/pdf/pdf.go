// Package pdf renders check results as a PDF/UA-1 report card via the
// boxesandglue/bagme typesetter. The output is structurally tagged
// (heading hierarchy, lists, tables) so screen readers can navigate
// the report itself.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bagmedoc "github.com/boxesandglue/bagme/document"
	"github.com/boxesandglue/boxesandglue/backend/document"
	"github.com/boxesandglue/boxesandglue/frontend"
	"github.com/boxesandglue/csshtml"

	"github.com/speedata/pdfa11y/internal/engine"
)

//go:embed style.css
var stylesheetCSS string

//go:embed logo.svg
var logoSVG []byte

// Build converts engine results for one document into the view shape the
// template expects. Mirrors html.Build so the two reports stay in lockstep.
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
			Suggestions:   sum.Infos,
		},
	}
	doc.Groups = groupByVerdict(results)
	return doc
}

// Option configures Write. Use the With* constructors.
type Option func(*options)

type options struct {
	version string
}

// WithVersion adds the pdfa11y version string to the "Generated …" line on
// the title page. Empty string (the default) suppresses the version suffix.
func WithVersion(v string) Option {
	return func(o *options) { o.version = v }
}

// Write renders one or more documents as a single PDF/UA-1 file. The
// caller is responsible for choosing the destination (stdout, file, …).
func Write(w io.Writer, docs []Document, opts ...Option) error {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	fe, err := frontend.NewForWriter(w)
	if err != nil {
		return fmt.Errorf("pdf: init frontend: %w", err)
	}
	fe.Doc.Format = document.FormatPDFUA

	cs := csshtml.NewCSSParserWithDefaults()
	d, err := bagmedoc.NewWithFrontend(fe, cs)
	if err != nil {
		return fmt.Errorf("pdf: init bagme: %w", err)
	}
	d.Title = reportTitle(docs)
	d.Language = "en"
	d.Creator = "pdfa11y"
	d.Subject = "PDF/UA Accessibility Report"

	// The speedata logo ships embedded; write it to a tempfile so
	// csshtml's `content: url(...)` in the @page margin box can find
	// it via an absolute path. The file lives for the duration of
	// Write only.
	logoPath, err := writeTempLogo()
	if err != nil {
		return fmt.Errorf("pdf: stage logo: %w", err)
	}
	defer os.Remove(logoPath)

	css := strings.ReplaceAll(stylesheetCSS, "__LOGO_PATH__", logoPath)
	if err := d.AddCSS(css); err != nil {
		return fmt.Errorf("pdf: add css: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pageData{
		Documents:   docs,
		GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		Version:     o.version,
	}); err != nil {
		return fmt.Errorf("pdf: render template: %w", err)
	}

	if err := d.RenderPages(buf.String()); err != nil {
		return fmt.Errorf("pdf: render pages: %w", err)
	}
	return d.Finish()
}

// writeTempLogo dumps the embedded speedata logo into a temp file and
// returns the absolute path. Caller owns the file (must os.Remove it).
func writeTempLogo() (string, error) {
	f, err := os.CreateTemp("", "pdfa11y-logo-*.svg")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(logoSVG); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// reportTitle picks a Title for the PDF metadata. PDF/UA requires a
// non-empty document Title; we use a generic one for multi-doc batches
// and include the filename when reporting on a single PDF.
func reportTitle(docs []Document) string {
	if len(docs) == 1 {
		return "PDF/UA Accessibility Report — " + docs[0].Filename
	}
	return "PDF/UA Accessibility Report"
}

type pageData struct {
	Documents   []Document
	GeneratedAt string
	Version     string // "" if unknown — suppresses the version suffix
}

// Document is the per-PDF view passed to the template.
type Document struct {
	Filename string
	Verdict  string
	Summary  Summary
	Groups   []Group
}

func (d Document) VerdictClass() string { return strings.ToLower(d.Verdict) }

// HasGroup reports whether a verdict group with the given slug is
// present (non-empty), so the summary can link a stat to its section
// only when that section actually exists.
func (d Document) HasGroup(slug string) bool {
	for _, g := range d.Groups {
		if g.Slug == slug {
			return true
		}
	}
	return false
}

type Summary struct {
	Passed, NotApplicable, Failed, Total, Errors, Warnings, Suggestions int
}

// Group bundles checks sharing a verdict bucket, in display order.
type Group struct {
	Label   string
	Slug    string
	Count   int
	Results []Result
}

type Result struct {
	ID          string
	Title       string
	Category    string
	Description string
	State       string
	DotClass    string // verdict-bucket slug for the status dot (suggestion != pass)
	WCAG        string
	Findings    []Finding
}

func (r Result) StateClass() string {
	if r.State == "N/A" {
		return "na"
	}
	return strings.ToLower(r.State)
}

type Finding struct {
	Severity string
	Message  string
	Hint     string
	Location string
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

// verdictBuckets lists the verdict groups in display order; the index
// is what bucketIndex returns.
var verdictBuckets = []struct{ label, slug string }{
	{"Failed", "fail"},
	{"Warnings", "warn"},
	{"Passed", "pass"},
	{"Suggestions", "suggestion"},
	{"Not applicable", "na"},
}

// bucketIndex maps a result to its verdict bucket. A passing check with
// advisory (Info) findings lands in "Suggestions" though its badge stays
// PASS.
func bucketIndex(r engine.Result) int {
	switch r.State() {
	case engine.VerdictFail:
		return 0
	case engine.VerdictWarn:
		return 1
	case engine.VerdictNA:
		return 4
	default: // VerdictPass
		if hasInfoFinding(r) {
			return 3
		}
		return 2
	}
}

func hasInfoFinding(r engine.Result) bool {
	for _, f := range r.Findings {
		if f.Severity == engine.SeverityInfo {
			return true
		}
	}
	return false
}

// groupByVerdict buckets results by verdict in display order (Failed,
// Warnings, Passed, Suggestions, Not applicable), omitting empty
// buckets; within a bucket, by category then ID.
func groupByVerdict(results []engine.Result) []Group {
	buckets := make([][]engine.Result, len(verdictBuckets))
	for _, r := range results {
		buckets[bucketIndex(r)] = append(buckets[bucketIndex(r)], r)
	}
	out := make([]Group, 0, len(verdictBuckets))
	for i, b := range verdictBuckets {
		rs := buckets[i]
		if len(rs) == 0 {
			continue
		}
		sort.Slice(rs, func(a, c int) bool {
			if rs[a].Check.Category() != rs[c].Check.Category() {
				return rs[a].Check.Category() < rs[c].Check.Category()
			}
			return rs[a].Check.ID() < rs[c].Check.ID()
		})
		g := Group{Label: b.label, Slug: b.slug, Count: len(rs), Results: make([]Result, 0, len(rs))}
		for _, r := range rs {
			res := buildResult(r)
			res.DotClass = b.slug
			g.Results = append(g.Results, res)
		}
		out = append(out, g)
	}
	return out
}

func buildResult(r engine.Result) Result {
	rv := Result{
		ID:          r.Check.ID(),
		Title:       r.Check.Title(),
		Category:    string(r.Check.Category()),
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

const reportTemplate = `{{range $i, $doc := .Documents}}
<section class="title-page">
  <p class="eyebrow">pdfa11y · PDF/UA accessibility report</p>
  <h1>Accessibility Report</h1>
  <p class="subtitle">PDF/UA conformance check results</p>
  <p class="filename">{{$doc.Filename}}</p>

  <div class="verdict-block verdict-{{$doc.VerdictClass}}">
    <p class="verdict-label">Overall verdict</p>
    <p class="verdict-value">{{$doc.Verdict}}</p>
  </div>

  <table class="stats">
    {{if $doc.HasGroup "pass"}}<tr><td class="num"><a href="#g-pass">{{$doc.Summary.Passed}} / {{$doc.Summary.Total}}</a></td><td class="label"><a href="#g-pass">checks passed</a></td></tr>{{else}}<tr><td class="num">{{$doc.Summary.Passed}} / {{$doc.Summary.Total}}</td><td class="label">checks passed</td></tr>{{end}}
    {{if $doc.HasGroup "fail"}}<tr><td class="num"><a href="#g-fail">{{$doc.Summary.Errors}}</a></td><td class="label"><a href="#g-fail">error{{if ne $doc.Summary.Errors 1}}s{{end}}</a></td></tr>{{else}}<tr><td class="num">{{$doc.Summary.Errors}}</td><td class="label">error{{if ne $doc.Summary.Errors 1}}s{{end}}</td></tr>{{end}}
    {{if $doc.HasGroup "warn"}}<tr><td class="num"><a href="#g-warn">{{$doc.Summary.Warnings}}</a></td><td class="label"><a href="#g-warn">warning{{if ne $doc.Summary.Warnings 1}}s{{end}}</a></td></tr>{{else}}<tr><td class="num">{{$doc.Summary.Warnings}}</td><td class="label">warning{{if ne $doc.Summary.Warnings 1}}s{{end}}</td></tr>{{end}}
    {{if $doc.Summary.Suggestions}}<tr><td class="num"><a href="#g-suggestion">{{$doc.Summary.Suggestions}}</a></td><td class="label"><a href="#g-suggestion">suggestion{{if ne $doc.Summary.Suggestions 1}}s{{end}}</a></td></tr>{{end}}
    {{if $doc.Summary.NotApplicable}}<tr><td class="num"><a href="#g-na">{{$doc.Summary.NotApplicable}}</a></td><td class="label"><a href="#g-na">not applicable</a></td></tr>{{end}}
  </table>

  <p class="meta">Generated {{$.GeneratedAt}} by pdfa11y{{with $.Version}} v{{.}}{{end}}</p>
</section>

<section class="report-page">
  <h2>Findings</h2>
  <p class="filename-line">{{$doc.Filename}}</p>
  {{range $doc.Groups}}
    <p class="group group-{{.Slug}}" id="g-{{.Slug}}">{{.Label}} ({{.Count}})</p>
    {{range .Results}}
      <article class="check check-{{.StateClass}}">
        <p class="check-head">
          <span class="state state-{{.DotClass}}">&#8226;</span>
          <span class="check-id">{{.ID}}</span>
          <span class="check-title">{{.Title}}</span>
          <span class="check-cat">{{.Category}}</span>
        </p>
        {{if .Description}}<p class="description">{{.Description}}</p>{{end}}
        {{range .Findings}}
          <div class="finding severity-{{.Severity}}">
            <p class="finding-msg"><span class="severity-label">{{.Severity}}</span> {{.Message}}</p>
            {{if .Location}}<p class="finding-loc">at: {{.Location}}</p>{{end}}
            {{if .Hint}}<p class="finding-hint">{{.Hint}}</p>{{end}}
          </div>
        {{end}}
      </article>
    {{end}}
  {{end}}
</section>
{{end}}`
