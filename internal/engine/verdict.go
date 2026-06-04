package engine

import (
	"encoding/json"
	"fmt"
)

// Verdict is the per-check / per-document outcome shown to users:
// PASS, WARN, FAIL, or N/A. N/A means the check did not apply (e.g.
// a structure-walking check on a document without a structure tree)
// and is treated like PASS for conformance purposes -- it does not
// reduce the verdict -- but rendered distinctly so users know the
// check was not actually exercised.
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

// State returns the per-check verdict. N/A wins over the otherwise-
// vacuous PASS when the check explicitly declared itself inapplicable
// via a SeverityNotApplicable finding.
func (r Result) State() Verdict {
	hasError := false
	hasOther := false
	hasNA := false
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			hasError = true
		case SeverityNotApplicable:
			hasNA = true
		default:
			hasOther = true
		}
	}
	switch {
	case hasError:
		return VerdictFail
	case hasNA && !hasOther:
		return VerdictNA
	case hasOther:
		return VerdictWarn
	}
	return VerdictPass
}

// Verdict returns the per-document verdict derived from the summary.
// N/A counts as PASS for the verdict; it never breaks conformance.
func (s Summary) Verdict() Verdict {
	switch {
	case !s.Conforming():
		return VerdictFail
	case s.Warnings > 0 || s.Infos > 0:
		return VerdictWarn
	}
	return VerdictPass
}
