package engine

import "github.com/speedata/pdfa11y/internal/model"

// Run executes every check against the document and returns one Result per check.
func Run(doc model.Document, checks []Check) []Result {
	results := make([]Result, len(checks))
	for i, c := range checks {
		results[i] = Result{Check: c, Findings: c.Run(doc)}
	}
	return results
}

// Summary aggregates pass/fail counts across results. NotApplicable
// results are tracked separately so reporters can distinguish "13/13
// checks ran cleanly" from "13/13 but four of them were not actually
// exercised".
type Summary struct {
	Total         int
	Passed        int
	Failed        int
	NotApplicable int
	Errors        int
	Warnings      int
	Infos         int
}

// Summarize counts pass/fail/n-a and severities over a slice of results.
// Passed + NotApplicable + Failed == Total.
func Summarize(results []Result) Summary {
	var s Summary
	s.Total = len(results)
	for _, r := range results {
		switch r.State() {
		case VerdictNA:
			s.NotApplicable++
		case VerdictFail:
			s.Failed++
		default:
			s.Passed++
		}
		for _, f := range r.Findings {
			switch f.Severity {
			case SeverityError:
				s.Errors++
			case SeverityWarning:
				s.Warnings++
			case SeverityInfo:
				s.Infos++
			}
		}
	}
	return s
}

// Conforming reports whether the document passes all error-level
// checks. N/A findings do not break conformance.
func (s Summary) Conforming() bool { return s.Errors == 0 }
