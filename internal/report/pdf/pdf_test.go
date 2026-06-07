package pdf_test

import (
	"bytes"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/report/pdf"
)

func TestBuildAndWrite(t *testing.T) {
	check := structure.StructTreeRoot{}
	results := []engine.Result{{
		Check: check,
		Findings: []engine.Finding{{
			Severity: engine.SeverityError,
			Message:  "no structure tree",
			Hint:     "tag the document",
		}},
	}}

	doc := pdf.Build("/tmp/private/sample.pdf", results)
	if doc.Filename != "sample.pdf" {
		t.Errorf("Filename = %q, want %q", doc.Filename, "sample.pdf")
	}
	if doc.Verdict != "FAIL" {
		t.Errorf("verdict = %q, want FAIL", doc.Verdict)
	}

	var buf bytes.Buffer
	if err := pdf.Write(&buf, []pdf.Document{doc}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output is not a PDF: starts with %q", out[:min(20, len(out))])
	}
	if !bytes.HasSuffix(bytes.TrimRight(out, "\n"), []byte("%%EOF")) {
		tail := out[max(0, len(out)-30):]
		t.Errorf("PDF does not end with %%EOF: tail = %q", tail)
	}
	// PDF/UA-1 marker in XMP metadata.
	if !bytes.Contains(out, []byte("pdfuaid:part")) {
		t.Errorf("output lacks pdfuaid:part XMP marker — not flagged as PDF/UA-1")
	}
}
