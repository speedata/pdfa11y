package engine_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/engine"
)

func TestPromoteWarnings(t *testing.T) {
	results := []engine.Result{
		{Findings: []engine.Finding{
			{Severity: engine.SeverityWarning, Message: "w1"},
			{Severity: engine.SeverityError, Message: "e1"},
			{Severity: engine.SeverityInfo, Message: "i1"},
		}},
		{Findings: []engine.Finding{
			{Severity: engine.SeverityWarning, Message: "w2"},
		}},
	}

	engine.PromoteWarnings(results)

	got := []engine.Severity{
		results[0].Findings[0].Severity,
		results[0].Findings[1].Severity,
		results[0].Findings[2].Severity,
		results[1].Findings[0].Severity,
	}
	want := []engine.Severity{
		engine.SeverityError, // promoted from Warning
		engine.SeverityError, // already Error
		engine.SeverityInfo,  // Info left alone
		engine.SeverityError, // promoted from Warning
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("findings[%d].Severity = %v, want %v", i, g, want[i])
		}
	}
}
