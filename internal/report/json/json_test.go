package json_test

import (
	"bytes"
	stdjson "encoding/json"
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	jsonrep "github.com/speedata/pdfa11y/internal/report/json"
)

func TestBuildAndWriteArray(t *testing.T) {
	check := structure.StructTreeRoot{}
	results := []engine.Result{{
		Check: check,
		Findings: []engine.Finding{
			{
				Severity: engine.SeverityError,
				Message:  "no structure tree",
				Hint:     "tag the document",
			},
		},
	}}

	doc := jsonrep.Build("sample.pdf", results)

	if doc.Verdict != engine.VerdictFail {
		t.Errorf("verdict = %v, want FAIL", doc.Verdict)
	}
	if doc.Summary.Errors != 1 {
		t.Errorf("summary.errors = %d, want 1", doc.Summary.Errors)
	}
	if doc.Summary.Conforming {
		t.Errorf("summary.conforming = true, want false")
	}
	if len(doc.Results) != 1 || doc.Results[0].ID != check.ID() {
		t.Fatalf("results: got %+v", doc.Results)
	}
	if doc.Results[0].State != engine.VerdictFail {
		t.Errorf("results[0].state = %v, want FAIL", doc.Results[0].State)
	}

	// Round-trip through JSON to verify nothing breaks during encoding.
	var buf bytes.Buffer
	if err := jsonrep.WriteArray(&buf, []jsonrep.Document{doc}); err != nil {
		t.Fatalf("WriteArray: %v", err)
	}
	if !strings.Contains(buf.String(), `"verdict": "FAIL"`) {
		t.Errorf("output missing verdict field: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"conforming": false`) {
		t.Errorf("output missing conforming flag: %s", buf.String())
	}

	// And re-decode to confirm the structure is well-formed.
	var decoded []jsonrep.Document
	if err := stdjson.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("re-decode failed: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("decoded %d documents, want 1", len(decoded))
	}
}

func TestWriteLine(t *testing.T) {
	doc := jsonrep.Document{Path: "x.pdf", Verdict: engine.VerdictPass}
	var buf bytes.Buffer
	if err := jsonrep.WriteLine(&buf, doc); err != nil {
		t.Fatalf("WriteLine: %v", err)
	}
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output does not end with newline: %q", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("output should contain exactly one newline: %q", out)
	}
}
