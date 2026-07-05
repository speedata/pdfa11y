package verasync

import (
	"fmt"
	"sort"
)

// Problem is a single drift/consistency issue found by Check.
type Problem struct {
	Vera string
	Msg  string
}

func (p Problem) String() string { return fmt.Sprintf("%s: %s", p.Vera, p.Msg) }

// specApplies reports whether a check with the given engine Spec string
// ("PDF/UA-1", "PDF/UA-2", "PDF/UA-1+2") applies to a rule whose spec token is
// "UA1" or "UA2".
func specApplies(checkSpec, ruleSpec string) bool {
	switch checkSpec {
	case "PDF/UA-1+2":
		return true
	case "PDF/UA-1":
		return ruleSpec == "UA1"
	case "PDF/UA-2":
		return ruleSpec == "UA2"
	default:
		return false
	}
}

// Check validates the manifest against the current upstream rules and the set
// of registered pdfa11y check IDs. It reports, in canonical order:
//   - a veraPDF rule with no manifest entry ("new gap");
//   - a manifest entry with no matching upstream rule ("upstream removed");
//   - a `checks:` reference to an unregistered check ID;
//   - status/checks inconsistencies (covered ⇒ checks non-empty; checks set ⇒
//     status ∈ {covered, partial}; unknown status value).
//
//   - a `checks:` reference whose check does not apply to the rule's spec (a
//     UA1 rule mapped to a UA2-only check, or vice versa).
//
// checkSpecs maps each registered pdfa11y check ID to its Spec string
// ("PDF/UA-1", "PDF/UA-2", or "PDF/UA-1+2"), as reported by engine. Pass nil to
// skip the check-ID cross-reference entirely.
func (m *Manifest) Check(rules []Rule, checkSpecs map[string]string) []Problem {
	var problems []Problem

	upstream := make(map[string]bool, len(rules))
	for _, r := range rules {
		upstream[r.Vera] = true
	}
	entries := make(map[string]bool, len(m.Rules))
	for _, e := range m.Rules {
		entries[e.Vera] = true
	}

	// New upstream rules with no manifest entry.
	for _, r := range rules {
		if !entries[r.Vera] {
			problems = append(problems, Problem{r.Vera,
				fmt.Sprintf("new veraPDF rule (%s) has no manifest entry — run `verasync -update`", r.Object)})
		}
	}

	for _, e := range m.Rules {
		// Entry whose upstream rule vanished.
		if !upstream[e.Vera] {
			problems = append(problems, Problem{e.Vera,
				"manifest entry has no matching veraPDF rule (upstream removed or renumbered)"})
		}
		// Unknown status.
		if !validStatus(e.Status) {
			problems = append(problems, Problem{e.Vera,
				fmt.Sprintf("unknown status %q", e.Status)})
		}
		// Status/checks consistency.
		if e.Status == StatusCovered && len(e.Checks) == 0 {
			problems = append(problems, Problem{e.Vera,
				"status is covered but no checks are listed"})
		}
		if len(e.Checks) > 0 && e.Status != StatusCovered && e.Status != StatusPartial {
			problems = append(problems, Problem{e.Vera,
				fmt.Sprintf("checks are listed but status is %q (expected covered or partial)", e.Status)})
		}
		// Check-ID references must resolve and apply to the rule's spec.
		if checkSpecs != nil {
			ruleSpec, _, _ := parseVera(e.Vera)
			for _, id := range e.Checks {
				cs, ok := checkSpecs[id]
				if !ok {
					problems = append(problems, Problem{e.Vera,
						fmt.Sprintf("references unregistered check ID %q", id)})
					continue
				}
				if !specApplies(cs, ruleSpec) {
					problems = append(problems, Problem{e.Vera,
						fmt.Sprintf("check %s applies to %s but the rule is %s", id, cs, ruleSpec)})
				}
			}
		}
	}

	sort.SliceStable(problems, func(i, j int) bool {
		si, ci, ti := parseVera(problems[i].Vera)
		sj, cj, tj := parseVera(problems[j].Vera)
		if si != sj || ci != cj || ti != tj {
			return lessVera(si, ci, ti, sj, cj, tj)
		}
		return problems[i].Msg < problems[j].Msg
	})
	return problems
}
