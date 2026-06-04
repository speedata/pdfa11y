// pdfa11ywasm is the WebAssembly entry point for pdfa11y. It exposes a
// single function, window.pdfa11y.check(uint8Array, filename), that
// runs all registered checks on the given PDF bytes and returns the
// rendered HTML report card. Designed for the speedata.de drop-and-
// inspect surface: PDFs never leave the browser sandbox.
//
// Build with: GOOS=js GOARCH=wasm go build -o web/pdfa11y.wasm ./cmd/pdfa11ywasm

//go:build js && wasm

package main

import (
	"bytes"
	"syscall/js"

	_ "github.com/speedata/pdfa11y/internal/checks" // register every check
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
	"github.com/speedata/pdfa11y/internal/report/html"
)

func main() {
	js.Global().Set("pdfa11y", js.ValueOf(map[string]any{
		"check":   js.FuncOf(check),
		"version": js.ValueOf("dev"),
	}))

	// syscall/js requires the program to stay alive so JS can invoke
	// the registered callbacks; blocking on an empty channel is the
	// canonical Go-WASM idiom.
	select {}
}

// check is the JS-facing entry point.
//
//	pdfa11y.check(bytes Uint8Array, filename string) => {
//	    verdict: "PASS"|"WARN"|"FAIL",
//	    passed:  number,
//	    total:   number,
//	    errors:  number,
//	    warnings: number,
//	    html:    string,  // the standalone HTML report card
//	    error:   string,  // present only on failure to parse the PDF
//	}
func check(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return result(map[string]any{"error": "missing PDF bytes argument"})
	}
	uarr := args[0]
	filename := "document.pdf"
	if len(args) >= 2 && args[1].Type() == js.TypeString {
		filename = args[1].String()
	}

	// Copy bytes from JS into Go. CopyBytesToGo is the idiomatic
	// bridge; the cost is a single memcpy.
	n := uarr.Get("length").Int()
	data := make([]byte, n)
	js.CopyBytesToGo(data, uarr)

	doc, err := pdf.Load(bytes.NewReader(data))
	if err != nil {
		return result(map[string]any{"error": err.Error()})
	}

	results := engine.Run(doc, engine.All())
	sum := engine.Summarize(results)

	var buf bytes.Buffer
	if err := html.Write(&buf, []html.Document{html.Build(filename, results)}); err != nil {
		return result(map[string]any{"error": err.Error()})
	}

	return result(map[string]any{
		"verdict":  sum.Verdict().String(),
		"passed":   sum.Passed,
		"total":    sum.Total,
		"errors":   sum.Errors,
		"warnings": sum.Warnings,
		"html":     buf.String(),
	})
}

// result wraps a map literal in js.ValueOf so the returned shape is
// always a plain JS object, even for the error path.
func result(m map[string]any) js.Value { return js.ValueOf(m) }
