// pdfa11y is a command-line accessibility checker for PDF documents.
// It runs the machine-checkable subset of PDF/UA conformance checks
// (ISO 14289-1 / -2; the Matterhorn protocol is the PDF/UA-1 basis) and
// prints a report card per document.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/speedata/optionparser"
	_ "github.com/speedata/pdfa11y/internal/checks"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdf"
	"github.com/speedata/pdfa11y/internal/pdfua"
	"github.com/speedata/pdfa11y/internal/report/html"
	jsonrep "github.com/speedata/pdfa11y/internal/report/json"
	pdfrep "github.com/speedata/pdfa11y/internal/report/pdf"
	"github.com/speedata/pdfa11y/internal/report/terminal"
)

// Version is set at build time via -ldflags "-X main.Version=...".
// The default "dev" applies to `go run` / `go build` without ldflags.
var Version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	var (
		format      = "terminal"
		specFlag    = "auto"
		showWCAG    bool
		showNA      bool
		listOnly    bool
		strict      bool
		showVersion bool
	)

	op := optionparser.NewOptionParser()
	op.Banner = "Usage: pdfa11y [options] FILE [FILE ...]"
	op.On("--format FORMAT", "output format: terminal (default), json, jsonl, html, pdf", &format)
	op.On("--spec SPEC", "PDF/UA spec: pdfua1, pdfua2, auto (default)", &specFlag)
	op.On("--wcag", "show WCAG mapping in the report", &showWCAG)
	op.On("--show-na", "show not-applicable checks in the terminal report (hidden by default)", &showNA)
	op.On("--strict", "treat warnings as errors (affects verdict and exit code)", &strict)
	op.On("--list-rules", "list registered checks and exit", &listOnly)
	op.On("--version", "print version and exit", &showVersion)

	if err := op.Parse(); err != nil {
		if errors.Is(err, optionparser.ErrHelp) {
			return 0
		}
		fmt.Fprintln(os.Stderr, "pdfa11y:", err)
		return 2
	}

	if showVersion {
		fmt.Println("pdfa11y", Version)
		return 0
	}

	if listOnly {
		listRules(os.Stdout)
		return 0
	}

	if len(op.Extra) == 0 {
		fmt.Fprintln(os.Stderr, "pdfa11y: no input files (try --help)")
		return 2
	}

	mode, err := parseSpecMode(specFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pdfa11y:", err)
		return 2
	}

	switch format {
	case "terminal", "json", "jsonl", "html", "pdf":
	default:
		fmt.Fprintf(os.Stderr, "pdfa11y: unknown --format %q (use terminal, json, jsonl, html, pdf)\n", format)
		return 2
	}

	anyFail := false
	anyError := false
	var jsonDocs []jsonrep.Document // accumulated only for --format json
	var htmlDocs []html.Document    // accumulated only for --format html
	var pdfDocs []pdfrep.Document   // accumulated only for --format pdf

	for _, path := range op.Extra {
		doc, err := pdf.LoadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pdfa11y: %s: %v\n", path, err)
			anyError = true
			continue
		}
		spec := resolveSpec(doc, mode)
		checks := engine.ForSpec(spec)
		results := engine.Run(doc, checks)
		if strict {
			engine.PromoteWarnings(results)
		}

		switch format {
		case "terminal":
			terminal.Write(os.Stdout, path, results, terminal.Options{ShowWCAG: showWCAG, ShowNA: showNA})
		case "jsonl":
			if err := jsonrep.WriteLine(os.Stdout, jsonrep.Build(path, results)); err != nil {
				fmt.Fprintln(os.Stderr, "pdfa11y:", err)
				return 2
			}
		case "json":
			jsonDocs = append(jsonDocs, jsonrep.Build(path, results))
		case "html":
			htmlDocs = append(htmlDocs, html.Build(path, results))
		case "pdf":
			pdfDocs = append(pdfDocs, pdfrep.Build(path, results))
		}

		if !engine.Summarize(results).Conforming() {
			anyFail = true
		}
	}

	if format == "json" {
		if err := jsonrep.WriteArray(os.Stdout, jsonDocs); err != nil {
			fmt.Fprintln(os.Stderr, "pdfa11y:", err)
			return 2
		}
	}
	if format == "html" {
		if err := html.Write(os.Stdout, htmlDocs); err != nil {
			fmt.Fprintln(os.Stderr, "pdfa11y:", err)
			return 2
		}
	}
	if format == "pdf" {
		if err := pdfrep.Write(os.Stdout, pdfDocs, pdfrep.WithVersion(Version)); err != nil {
			fmt.Fprintln(os.Stderr, "pdfa11y:", err)
			return 2
		}
	}

	switch {
	case anyError:
		return 2
	case anyFail:
		return 1
	default:
		return 0
	}
}

// specMode reflects what the user asked for via --spec; the actual
// engine.Spec is derived per document so autodetection can inspect each
// file's XMP independently.
type specMode int

const (
	specAuto specMode = iota
	specForcedUA1
	specForcedUA2
)

func parseSpecMode(s string) (specMode, error) {
	switch s {
	case "pdfua1":
		return specForcedUA1, nil
	case "pdfua2":
		return specForcedUA2, nil
	case "auto", "":
		return specAuto, nil
	}
	return 0, fmt.Errorf("unknown --spec %q (use pdfua1, pdfua2, auto)", s)
}

// resolveSpec returns the spec set to run for doc. With --spec auto we
// look at the XMP pdfuaid:part value; if the document declares neither
// PDF/UA-1 nor PDF/UA-2 we fall back to running both check sets, since
// the user probably wants to know about both kinds of failure.
func resolveSpec(doc model.Document, mode specMode) engine.Spec {
	switch mode {
	case specForcedUA1:
		return engine.SpecPDFUA1
	case specForcedUA2:
		return engine.SpecPDFUA2
	}
	part, found, _ := pdfua.DetectPart(doc)
	if !found {
		return engine.SpecBoth
	}
	switch part {
	case 1:
		return engine.SpecPDFUA1
	case 2:
		return engine.SpecPDFUA2
	}
	return engine.SpecBoth
}

func listRules(w *os.File) {
	for _, c := range engine.All() {
		wcag := ""
		if v := c.WCAG(); len(v) > 0 {
			wcag = " (WCAG "
			for i, s := range v {
				if i > 0 {
					wcag += ", "
				}
				wcag += s
			}
			wcag += ")"
		}
		fmt.Fprintf(w, "%s  [%s]  %s — %s%s\n",
			c.ID(), c.Spec(), c.Category(), c.Title(), wcag)
	}
}
