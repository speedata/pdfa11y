// Package pdfua holds shared helpers for the PDF/UA identifier in XMP
// metadata. Both the UA-06-003 check and CLI spec autodetection need to
// read pdfuaid:part, so the regex lives here once.
//
// Limitation: same as UA-06-003. The conventional "pdfuaid" namespace
// prefix is assumed; a non-conventional prefix bound to
// http://www.aiim.org/pdfua/ns/id/ would be missed.
package pdfua

import (
	"regexp"
	"strconv"

	"github.com/speedata/pdfa11y/internal/model"
)

var (
	partElem = regexp.MustCompile(`<\s*pdfuaid:part\s*>\s*([12])\s*<\s*/\s*pdfuaid:part\s*>`)
	partAttr = regexp.MustCompile(`\bpdfuaid:part\s*=\s*["']\s*([12])\s*["']`)
	revElem  = regexp.MustCompile(`<\s*pdfuaid:rev\s*>\s*(\d{4})\s*<\s*/\s*pdfuaid:rev\s*>`)
	revAttr  = regexp.MustCompile(`\bpdfuaid:rev\s*=\s*["']\s*(\d{4})\s*["']`)
)

// DetectPart reads the document's XMP metadata stream (if any) and
// returns the declared pdfuaid:part value. Returned values are 1 or 2.
// If the stream is absent or contains no pdfuaid:part, returns 0 and
// found=false. Errors during catalog or stream access propagate.
func DetectPart(doc model.Document) (part int, found bool, err error) {
	catalog, err := doc.Catalog()
	if err != nil {
		return 0, false, err
	}
	mdObj, present := catalog.Find("Metadata")
	if !present {
		return 0, false, nil
	}
	content, err := doc.DecodeStream(mdObj)
	if err != nil {
		return 0, false, err
	}
	if m := partElem.FindSubmatch(content); m != nil {
		n, _ := strconv.Atoi(string(m[1]))
		return n, true, nil
	}
	if m := partAttr.FindSubmatch(content); m != nil {
		n, _ := strconv.Atoi(string(m[1]))
		return n, true, nil
	}
	return 0, false, nil
}

// DetectRev reads the document's XMP metadata stream and returns the
// declared pdfuaid:rev value (a four-digit year per ISO 14289-2 §5).
// Returns (0, false, nil) when the stream is absent or carries no
// pdfuaid:rev. The regex requires exactly four digits, matching the
// spec's "four digits of the year of publication or revision".
// Same XMP-prefix limitation as DetectPart.
func DetectRev(doc model.Document) (rev int, found bool, err error) {
	catalog, err := doc.Catalog()
	if err != nil {
		return 0, false, err
	}
	mdObj, present := catalog.Find("Metadata")
	if !present {
		return 0, false, nil
	}
	content, err := doc.DecodeStream(mdObj)
	if err != nil {
		return 0, false, err
	}
	if m := revElem.FindSubmatch(content); m != nil {
		n, _ := strconv.Atoi(string(m[1]))
		return n, true, nil
	}
	if m := revAttr.FindSubmatch(content); m != nil {
		n, _ := strconv.Atoi(string(m[1]))
		return n, true, nil
	}
	return 0, false, nil
}
