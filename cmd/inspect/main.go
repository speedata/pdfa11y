// inspect is an ad-hoc debug helper used during development to print
// what pdfa11y "sees" in a real-world PDF: DocInfo, fonts, and the
// shape of the structure tree. Intentionally simple -- not part of the
// shipped CLI surface.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: inspect FILE.pdf")
		os.Exit(2)
	}
	doc, err := pdf.LoadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	fmt.Println("--- DocInfo ---")
	fmt.Printf("Title=%q\n", doc.DocInfo().Title)

	fmt.Println("--- Fonts ---")
	fonts, _ := doc.Fonts()
	for _, f := range fonts {
		fmt.Printf("  %-32s %-9s embedded=%v ToU=%v\n", f.BaseFont, f.Subtype, f.Embedded, f.HasToUnicode)
	}

	fmt.Println("--- Struct Tree ---")
	root, err := doc.StructTreeRoot()
	if err != nil {
		panic(err)
	}
	if root == nil {
		fmt.Println("(no structure tree)")
		return
	}
	counts := map[string]int{}
	walk(root, 0, counts)

	fmt.Println("--- Type counts ---")
	for typ, n := range counts {
		fmt.Printf("  %-15s %d\n", typ, n)
	}
}

func walk(e model.StructElement, depth int, counts map[string]int) {
	counts[e.Type()]++
	if depth < 4 {
		fmt.Printf("%s%s\n", strings.Repeat("  ", depth), e.Type())
	}
	for _, c := range e.Children() {
		walk(c, depth+1, counts)
	}
}
