package github_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/report/github"
)

func sampleResults() []engine.Result {
	return []engine.Result{{
		Check: structure.StructTreeRoot{},
		Findings: []engine.Finding{
			{
				Severity: engine.SeverityError,
				Message:  "no structure tree",
				Hint:     "tag the document",
				Location: &engine.Location{Page: 2, StructPath: "/Document/P[1]"},
			},
			{Severity: engine.SeverityWarning, Message: "50% done"},
			{Severity: engine.SeverityInfo, Message: "consider this"},
			{Severity: engine.SeverityNotApplicable, Message: "nothing to inspect"},
		},
	}}
}

func TestWriteAnnotations(t *testing.T) {
	var buf bytes.Buffer
	github.Write(&buf, "out/a,b:c.pdf", sampleResults())
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")

	want := []string{
		"out/a,b:c.pdf: FAIL   errors: 1 · warnings: 1 · suggestions: 1",
		"::error file=out/a%2Cb%3Ac.pdf,title=UA-01-005 Document has a structure tree::no structure tree%0Aat: page 2, /Document/P[1]%0Ahint: tag the document",
		"::warning file=out/a%2Cb%3Ac.pdf,title=UA-01-005 Document has a structure tree::50%25 done",
		"::notice file=out/a%2Cb%3Ac.pdf,title=UA-01-005 Document has a structure tree::consider this",
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(want), buf.String())
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d:\n got %q\nwant %q", i, lines[i], want[i])
		}
	}
}

func TestWriteLoadError(t *testing.T) {
	var buf bytes.Buffer
	github.WriteLoadError(&buf, "x.pdf", errors.New("bad\nxref"))
	want := "::error file=x.pdf,title=pdfa11y::cannot load PDF: bad%0Axref\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteSummary(t *testing.T) {
	var buf bytes.Buffer
	github.WriteSummary(&buf, []github.Document{
		{Path: "a.pdf", Results: sampleResults()},
		{Path: "ok.pdf", Results: []engine.Result{{Check: structure.StructTreeRoot{}}}},
		{Path: "broken.pdf", Err: errors.New("bad xref")},
	})
	out := buf.String()

	for _, s := range []string{
		"| <code>a.pdf</code> | ❌ FAIL | 1 | 1 | 1 |",
		"| <code>ok.pdf</code> | ✅ PASS | 0 | 0 | 0 |",
		"| <code>broken.pdf</code> | ⛔ not loaded | – | – | – |",
		"<summary><code>a.pdf</code>: 1 checks with findings</summary>",
		"| UA-01-005 | Document has a structure tree | FAIL | 3 |",
		"**<code>broken.pdf</code>**: bad xref",
	} {
		if !strings.Contains(out, s) {
			t.Errorf("summary missing %q:\n%s", s, out)
		}
	}
	if strings.Contains(out, "<summary><code>ok.pdf</code>") {
		t.Errorf("clean document should not get a details block:\n%s", out)
	}
}
