package engine_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/engine"
)

// TestResultState exercises the per-check verdict precedence:
// Error > Warning > (Info treated as PASS) ; N/A only when nothing
// else (not even an advisory) is present.
func TestResultState(t *testing.T) {
	sev := func(s engine.Severity) engine.Finding { return engine.Finding{Severity: s} }
	tests := []struct {
		name     string
		findings []engine.Finding
		want     engine.Verdict
	}{
		{"no findings is PASS", nil, engine.VerdictPass},
		{"error is FAIL", []engine.Finding{sev(engine.SeverityError)}, engine.VerdictFail},
		{"warning is WARN", []engine.Finding{sev(engine.SeverityWarning)}, engine.VerdictWarn},
		{"info-only is PASS", []engine.Finding{sev(engine.SeverityInfo)}, engine.VerdictPass},
		{"n/a is N/A", []engine.Finding{sev(engine.SeverityNotApplicable)}, engine.VerdictNA},
		{
			"warning outranks info",
			[]engine.Finding{sev(engine.SeverityInfo), sev(engine.SeverityWarning)},
			engine.VerdictWarn,
		},
		{
			// An advisory means the check applied, so it PASSES rather
			// than reporting N/A.
			"info outranks n/a (PASS, not N/A)",
			[]engine.Finding{sev(engine.SeverityNotApplicable), sev(engine.SeverityInfo)},
			engine.VerdictPass,
		},
		{
			"error outranks everything",
			[]engine.Finding{sev(engine.SeverityInfo), sev(engine.SeverityWarning), sev(engine.SeverityError)},
			engine.VerdictFail,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := engine.Result{Findings: tc.findings}
			if got := r.State(); got != tc.want {
				t.Errorf("State() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSummaryVerdict checks the document-level rollup: an info-only
// document still PASSES (advisories never reduce the verdict), but the
// suggestion count is preserved on the summary.
func TestSummaryVerdict(t *testing.T) {
	tests := []struct {
		name         string
		results      []engine.Result
		wantVerdict  engine.Verdict
		wantInfoSeen int
	}{
		{"clean is PASS", []engine.Result{{}}, engine.VerdictPass, 0},
		{
			"info-only is PASS but counted",
			[]engine.Result{{Findings: []engine.Finding{{Severity: engine.SeverityInfo}}}},
			engine.VerdictPass,
			1,
		},
		{
			"warning beats info at document level",
			[]engine.Result{
				{Findings: []engine.Finding{{Severity: engine.SeverityInfo}}},
				{Findings: []engine.Finding{{Severity: engine.SeverityWarning}}},
			},
			engine.VerdictWarn,
			1,
		},
		{
			"error is FAIL",
			[]engine.Result{{Findings: []engine.Finding{{Severity: engine.SeverityError}}}},
			engine.VerdictFail,
			0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sum := engine.Summarize(tc.results)
			if got := sum.Verdict(); got != tc.wantVerdict {
				t.Errorf("Verdict() = %v, want %v", got, tc.wantVerdict)
			}
			if sum.Infos != tc.wantInfoSeen {
				t.Errorf("Infos = %d, want %d", sum.Infos, tc.wantInfoSeen)
			}
		})
	}
}

// TestVerdictStringRoundTrip guards the JSON-facing labels.
func TestVerdictStringRoundTrip(t *testing.T) {
	for _, v := range []engine.Verdict{
		engine.VerdictPass, engine.VerdictWarn, engine.VerdictFail, engine.VerdictNA,
	} {
		b, err := v.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON(%v): %v", v, err)
		}
		var got engine.Verdict
		if err := got.UnmarshalJSON(b); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", b, err)
		}
		if got != v {
			t.Errorf("round trip %v -> %s -> %v", v, b, got)
		}
	}
}
