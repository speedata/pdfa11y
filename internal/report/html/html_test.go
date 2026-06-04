package html_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/report/html"
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

	// Path includes a directory prefix so the basename-only rendering
	// can be asserted; the prefix must not appear in the output.
	doc := html.Build("/tmp/private/sample.pdf", results)
	if doc.Filename != "sample.pdf" {
		t.Errorf("Filename = %q, want %q", doc.Filename, "sample.pdf")
	}
	if doc.Verdict != "FAIL" {
		t.Errorf("verdict = %q, want FAIL", doc.Verdict)
	}

	var buf bytes.Buffer
	if err := html.Write(&buf, []html.Document{doc}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"<!DOCTYPE html>",
		`<title>PDF/UA Accessibility Report</title>`,
		`<h1>PDF/UA Accessibility Report</h1>`,
		"<h2>sample.pdf</h2>",
		`class="summary-verdict verdict-fail"`,
		`class="state-grid state-fail"`,
		`<details class="check check-fail" open>`,
		"MH-01-005",
		"Document has a structure tree",
		"A PDF/UA-conforming document", // beginning of Description()
		"no structure tree",
		"tag the document",
		`href="https://www.speedata.de"`,
		"speedata",
		`<svg`,
		`viewBox="0 0 66 66"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}

	// The directory prefix must NOT leak into the output (privacy).
	if strings.Contains(out, "/tmp/private/") {
		t.Errorf("output unexpectedly contains the full path -- basename rendering broken")
	}
	// Inline SVG inside HTML5 must not carry the XML declaration.
	if strings.Contains(out, "<?xml") {
		t.Errorf("output unexpectedly contains an XML declaration")
	}
}

func TestOpenByDefault(t *testing.T) {
	cases := []struct {
		state    string
		wantOpen bool
	}{
		{"PASS", false},
		{"WARN", false}, // collapsed to keep the page calm; user expands on demand
		{"FAIL", true},
	}
	for _, tc := range cases {
		r := html.Result{State: tc.state}
		if got := r.OpenByDefault(); got != tc.wantOpen {
			t.Errorf("OpenByDefault for %s = %v, want %v", tc.state, got, tc.wantOpen)
		}
	}
}

// TestWrite_MultipleDocs confirms that multiple input PDFs render as
// stacked articles, each with its own verdict and basename heading.
func TestWrite_MultipleDocs(t *testing.T) {
	docA := html.Build("/x/a.pdf", nil)
	docB := html.Build("/y/b.pdf", nil)

	var buf bytes.Buffer
	if err := html.Write(&buf, []html.Document{docA, docB}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()
	if strings.Count(out, "<article>") != 2 {
		t.Errorf("got %d articles, want 2", strings.Count(out, "<article>"))
	}
	for _, want := range []string{"<h2>a.pdf</h2>", "<h2>b.pdf</h2>"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	// Only one top-level H1 even with multiple documents.
	if strings.Count(out, "<h1>PDF/UA Accessibility Report</h1>") != 1 {
		t.Errorf("expected exactly one top-level H1, got %d", strings.Count(out, "<h1>"))
	}
}
