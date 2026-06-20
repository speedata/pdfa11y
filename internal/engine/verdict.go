package engine

import (
	"encoding/json"
	"fmt"
)

// Verdict is the per-check / per-document outcome shown to users:
// PASS, WARN, FAIL, or N/A.
//
// Advisory (Info) findings -- spec recommendations ("should"), not
// violations -- do not get their own verdict: a check that produces
// only advisories still PASSES. The advisories are surfaced as findings
// and counted in the summary ("suggestions: N"), but they never reduce
// the verdict and are left untouched by --strict.
//
// N/A means the check did not apply (e.g. a structure-walking check on
// a document without a structure tree) and is treated like PASS for
// conformance purposes -- it does not reduce the verdict -- but
// rendered distinctly so users know the check was not actually
// exercised.
type Verdict uint8

const (
	VerdictPass Verdict = iota
	VerdictWarn
	VerdictFail
	VerdictNA
)

func (v Verdict) String() string {
	switch v {
	case VerdictPass:
		return "PASS"
	case VerdictWarn:
		return "WARN"
	case VerdictFail:
		return "FAIL"
	case VerdictNA:
		return "N/A"
	}
	return "UNKNOWN"
}

func (v Verdict) MarshalJSON() ([]byte, error) { return json.Marshal(v.String()) }

func (v *Verdict) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "PASS":
		*v = VerdictPass
	case "WARN":
		*v = VerdictWarn
	case "FAIL":
		*v = VerdictFail
	case "N/A":
		*v = VerdictNA
	default:
		return fmt.Errorf("unknown verdict %q", s)
	}
	return nil
}

// State returns the per-check verdict, picking the most severe finding
// present: Error -> FAIL, Warning -> WARN, otherwise PASS. Advisory
// (Info) findings do not produce their own verdict -- a check with only
// advisories still PASSES (they are surfaced as findings and counted in
// the summary). N/A wins over PASS only when the check declared itself
// inapplicable and produced nothing else (not even an advisory).
func (r Result) State() Verdict {
	hasError := false
	hasWarning := false
	hasInfo := false
	hasNA := false
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			hasError = true
		case SeverityWarning:
			hasWarning = true
		case SeverityInfo:
			hasInfo = true
		case SeverityNotApplicable:
			hasNA = true
		}
	}
	switch {
	case hasError:
		return VerdictFail
	case hasWarning:
		return VerdictWarn
	case hasNA && !hasInfo:
		return VerdictNA
	}
	return VerdictPass
}

// Verdict returns the per-document verdict derived from the summary.
// N/A counts as PASS for the verdict; it never breaks conformance.
// Advisory (Info) findings do not reduce the verdict -- a document with
// only suggestions still PASSES; the count surfaces as "suggestions: N".
func (s Summary) Verdict() Verdict {
	switch {
	case !s.Conforming():
		return VerdictFail
	case s.Warnings > 0:
		return VerdictWarn
	}
	return VerdictPass
}
