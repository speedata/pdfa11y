package verasync

import (
	"strings"
	"testing"
)

func TestParseVera(t *testing.T) {
	cases := []struct {
		in         string
		spec       string
		clause     string
		test       int
	}{
		{"UA2:8.2.5.26-5", "UA2", "8.2.5.26", 5},
		{"UA1:7.1-1", "UA1", "7.1", 1},
		{"UA1:5-3", "UA1", "5", 3},
	}
	for _, c := range cases {
		s, cl, tn := parseVera(c.in)
		if s != c.spec || cl != c.clause || tn != c.test {
			t.Errorf("parseVera(%q) = (%q,%q,%d), want (%q,%q,%d)", c.in, s, cl, tn, c.spec, c.clause, c.test)
		}
	}
}

func TestCompareClause(t *testing.T) {
	// "8.2.5.2" must sort before "8.2.5.26" (numeric, not lexical).
	if compareClause("8.2.5.2", "8.2.5.26") >= 0 {
		t.Error("8.2.5.2 should sort before 8.2.5.26")
	}
	if compareClause("7.1", "7.1") != 0 {
		t.Error("equal clauses should compare equal")
	}
	if compareClause("10", "9") <= 0 {
		t.Error("clause 10 should sort after clause 9 numerically")
	}
}

func TestMergePreservesManualFields(t *testing.T) {
	m := &Manifest{Rules: []Entry{{
		Vera:   "UA2:8.2.5.26-5",
		Object: "OLD",
		Desc:   "old desc",
		Status: StatusCovered,
		Checks: []string{"UA-15-006"},
		Note:   "keep me",
	}}}
	rules := []Rule{{
		Vera:        "UA2:8.2.5.26-5",
		Spec:        "UA2",
		Object:      "SETD",
		Description: "new desc",
	}}
	added, gone := m.Merge(rules)
	if len(added) != 0 || len(gone) != 0 {
		t.Fatalf("unexpected added=%v gone=%v", added, gone)
	}
	e := m.Rules[0]
	if e.Object != "SETD" || e.Desc != "new desc" {
		t.Errorf("upstream fields not refreshed: %+v", e)
	}
	if e.Status != StatusCovered || len(e.Checks) != 1 || e.Note != "keep me" {
		t.Errorf("manual fields clobbered: %+v", e)
	}
}

func TestMergeMarksGone(t *testing.T) {
	m := &Manifest{Rules: []Entry{{Vera: "UA2:9.9-9", Status: StatusTodo}}}
	_, gone := m.Merge([]Rule{{Vera: "UA2:8.2-1", Spec: "UA2"}})
	if len(gone) != 1 || gone[0] != "UA2:9.9-9" {
		t.Fatalf("expected UA2:9.9-9 marked gone, got %v", gone)
	}
	for _, e := range m.Rules {
		if e.Vera == "UA2:9.9-9" && !e.Gone {
			t.Error("entry should be marked Gone")
		}
	}
}

func TestCheckDetectsInconsistency(t *testing.T) {
	m := &Manifest{Rules: []Entry{
		{Vera: "UA2:1-1", Status: StatusCovered, Checks: nil},        // covered w/o checks
		{Vera: "UA2:1-2", Status: StatusTodo, Checks: []string{"X"}}, // checks w/ todo
		{Vera: "UA2:1-3", Status: "bogus"},                           // unknown status
	}}
	rules := []Rule{{Vera: "UA2:1-1", Spec: "UA2"}, {Vera: "UA2:1-2", Spec: "UA2"}, {Vera: "UA2:1-3", Spec: "UA2"}}
	problems := m.Check(rules, map[string]string{})
	if len(problems) < 4 {
		// covered-no-checks, checks-with-todo, unregistered "X", unknown status
		t.Errorf("expected >=4 problems, got %d: %v", len(problems), problems)
	}
}

func TestCheckDetectsSpecMismatch(t *testing.T) {
	m := &Manifest{Rules: []Entry{
		{Vera: "UA1:1-1", Status: StatusCovered, Checks: []string{"UA-X"}},
	}}
	rules := []Rule{{Vera: "UA1:1-1", Spec: "UA1"}}
	// UA-X applies to PDF/UA-2 only, but the rule is UA1 → mismatch.
	problems := m.Check(rules, map[string]string{"UA-X": "PDF/UA-2"})
	found := false
	for _, p := range problems {
		if p.Vera == "UA1:1-1" && strings.Contains(p.Msg, "applies to PDF/UA-2") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected spec-mismatch problem, got %v", problems)
	}
}
