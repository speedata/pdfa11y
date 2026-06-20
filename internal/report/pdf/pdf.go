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
	"slices"
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
	doc.Categories = groupByCategory(results)
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
	Filename   string
	Verdict    string
	Summary    Summary
	Categories []Category
}

func (d Document) VerdictClass() string { return strings.ToLower(d.Verdict) }

type Summary struct {
	Passed, NotApplicable, Failed, Total, Errors, Warnings, Suggestions int
}

type Category struct {
	Name    string
	Results []Result
}

type Result struct {
	ID          string
	Title       string
	Description string
	State       string
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
    <tr><td class="num">{{$doc.Summary.Passed}} / {{$doc.Summary.Total}}</td><td class="label">checks passed</td></tr>
    {{if $doc.Summary.NotApplicable}}<tr><td class="num">{{$doc.Summary.NotApplicable}}</td><td class="label">not applicable</td></tr>{{end}}
    <tr><td class="num">{{$doc.Summary.Errors}}</td><td class="label">error{{if ne $doc.Summary.Errors 1}}s{{end}}</td></tr>
    <tr><td class="num">{{$doc.Summary.Warnings}}</td><td class="label">warning{{if ne $doc.Summary.Warnings 1}}s{{end}}</td></tr>
    {{if $doc.Summary.Suggestions}}<tr><td class="num">{{$doc.Summary.Suggestions}}</td><td class="label">suggestion{{if ne $doc.Summary.Suggestions 1}}s{{end}}</td></tr>{{end}}
  </table>

  <p class="meta">Generated {{$.GeneratedAt}} by pdfa11y{{with $.Version}} v{{.}}{{end}}</p>
</section>

<section class="report-page">
  <h2>Findings</h2>
  <p class="filename-line">{{$doc.Filename}}</p>
  {{range $doc.Categories}}
    <p class="category">{{.Name}}</p>
    {{range .Results}}
      <article class="check check-{{.StateClass}}">
        <p class="check-head">
          <span class="state state-{{.StateClass}}">{{.State}}</span>
          <span class="check-id">{{.ID}}</span>
          <span class="check-title">{{.Title}}</span>
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
