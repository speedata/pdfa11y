// Package json renders check results as JSON. Two flavours are
// supported: a JSON array of documents (for tooling that parses a full
// report at once) and JSON Lines, one document per line, for streaming
// pipelines.
package json

import (
	stdjson "encoding/json"
	"io"

	"github.com/speedata/pdfa11y/internal/engine"
)

// Document is the JSON shape emitted per input file.
type Document struct {
	Path    string         `json:"path"`
	Verdict engine.Verdict `json:"verdict"`
	Summary Summary        `json:"summary"`
	Results []Result       `json:"results"`
}

// Summary mirrors engine.Summary but with explicit JSON tags and the
// derived `conforming` flag exposed.
type Summary struct {
	Total      int  `json:"total"`
	Passed     int  `json:"passed"`
	Failed     int  `json:"failed"`
	Errors     int  `json:"errors"`
	Warnings   int  `json:"warnings"`
	Infos      int  `json:"infos"`
	Conforming bool `json:"conforming"`
}

// Result is the per-check JSON shape.
type Result struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Spec        engine.Spec     `json:"spec"`
	Severity    engine.Severity `json:"severity"`
	WCAG        []string        `json:"wcag,omitempty"`
	State       engine.Verdict  `json:"state"`
	Findings    []Finding       `json:"findings"`
}

// Finding is the per-finding JSON shape.
type Finding struct {
	Severity engine.Severity `json:"severity"`
	Message  string          `json:"message"`
	Hint     string          `json:"hint,omitempty"`
	Location *Location       `json:"location,omitempty"`
}

// Location optionally pinpoints where a finding occurs.
type Location struct {
	Page         int    `json:"page,omitempty"`
	ObjectNumber int    `json:"object_number,omitempty"`
	StructPath   string `json:"struct_path,omitempty"`
}

// Build converts engine results into the JSON-friendly Document shape.
// It is intentionally a pure function so reporters can collect or stream
// documents as they see fit.
func Build(path string, results []engine.Result) Document {
	sum := engine.Summarize(results)
	doc := Document{
		Path:    path,
		Verdict: sum.Verdict(),
		Summary: Summary{
			Total:      sum.Total,
			Passed:     sum.Passed,
			Failed:     sum.Failed,
			Errors:     sum.Errors,
			Warnings:   sum.Warnings,
			Infos:      sum.Infos,
			Conforming: sum.Conforming(),
		},
		Results: make([]Result, 0, len(results)),
	}
	for _, r := range results {
		doc.Results = append(doc.Results, buildResult(r))
	}
	return doc
}

func buildResult(r engine.Result) Result {
	out := Result{
		ID:          r.Check.ID(),
		Title:       r.Check.Title(),
		Description: r.Check.Description(),
		Category:    string(r.Check.Category()),
		Spec:        r.Check.Spec(),
		Severity:    r.Check.Severity(),
		WCAG:        r.Check.WCAG(),
		State:       r.State(),
		Findings:    make([]Finding, 0, len(r.Findings)),
	}
	for _, f := range r.Findings {
		out.Findings = append(out.Findings, buildFinding(f))
	}
	return out
}

func buildFinding(f engine.Finding) Finding {
	out := Finding{
		Severity: f.Severity,
		Message:  f.Message,
		Hint:     f.Hint,
	}
	if f.Location != nil {
		out.Location = &Location{
			Page:         f.Location.Page,
			ObjectNumber: f.Location.ObjectNumber,
			StructPath:   f.Location.StructPath,
		}
	}
	return out
}

// WriteArray emits docs as a JSON array (a single top-level value). Suited
// for tooling that consumes the full report as one parse.
func WriteArray(w io.Writer, docs []Document) error {
	enc := stdjson.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(docs)
}

// WriteLine emits a single document as one compact JSON line followed by
// a newline -- the JSON Lines format. Multiple calls produce a streamable
// log suitable for pipelines (jq, log shippers, etc.).
func WriteLine(w io.Writer, doc Document) error {
	enc := stdjson.NewEncoder(w)
	enc.SetEscapeHTML(false)
	// stdjson.Encoder always appends a newline; no indent for JSONL.
	return enc.Encode(doc)
}
