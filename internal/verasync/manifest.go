package verasync

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Status classifies how a veraPDF rule is handled by pdfa11y.
type Status string

const (
	StatusCovered Status = "covered" // a registered check enforces the rule
	StatusPartial Status = "partial" // a check covers part of the rule
	StatusTodo    Status = "todo"    // not yet implemented
	StatusNA      Status = "n/a"     // deliberately out of scope (e.g. spec-mismatch)
	StatusWontfix Status = "wontfix" // consciously not implemented
)

// validStatus reports whether s is a known status value.
func validStatus(s Status) bool {
	switch s {
	case StatusCovered, StatusPartial, StatusTodo, StatusNA, StatusWontfix:
		return true
	}
	return false
}

// Entry is one manifest record: a veraPDF rule plus pdfa11y's coverage of it.
// The upstream-owned fields (Object, Desc) are refreshed by `verasync -update`;
// the hand-maintained fields (Status, Checks, Note) are preserved across
// updates.
type Entry struct {
	Vera   string   `yaml:"vera"`             // canonical veraPDF rule ID, e.g. "UA2:8.2.5.26-5"
	Object string   `yaml:"object"`           // veraPDF model object (upstream)
	Desc   string   `yaml:"desc"`             // requirement text (upstream, CC BY 4.0)
	Status Status   `yaml:"status"`           // coverage status (hand-maintained)
	Checks []string `yaml:"checks"`           // pdfa11y check IDs, e.g. [UA-15-006] (hand-maintained)
	Note   string   `yaml:"note,omitempty"`   // rationale / cross-reference (hand-maintained)
	Gone   bool     `yaml:"gone,omitempty"`   // set by -update when the upstream rule vanished
}

// Manifest is the full coverage map, one entry per veraPDF rule, kept in
// canonical (spec, clause, test) order for diffable output.
type Manifest struct {
	// Source records the pinned veraPDF profile version this manifest was last
	// merged against (informational; the authoritative pin lives in
	// PINNED_VERSION).
	Source string  `yaml:"source,omitempty"`
	Rules  []Entry `yaml:"rules"`
}

// LoadManifest reads a coverage manifest. A missing file yields an empty
// manifest (so the first `-update` bootstraps it).
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Manifest{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &m, nil
}

// Save writes the manifest as YAML, sorted canonically.
func (m *Manifest) Save(path string) error {
	m.sort()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(m); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// index maps vera ID → position in Rules.
func (m *Manifest) index() map[string]int {
	idx := make(map[string]int, len(m.Rules))
	for i, e := range m.Rules {
		idx[e.Vera] = i
	}
	return idx
}

// Merge folds the given veraPDF rules into the manifest:
//   - a rule with no entry is added with status "todo";
//   - an existing entry has its upstream fields (Object, Desc) refreshed while
//     its hand-maintained fields (Status, Checks, Note) are left untouched;
//   - an entry whose rule no longer exists upstream is marked Gone (kept, not
//     dropped, so notes survive and `-check` can flag the drift).
//
// It returns the vera IDs that were newly added and those newly marked gone.
func (m *Manifest) Merge(rules []Rule) (added, gone []string) {
	idx := m.index()
	live := make(map[string]bool, len(rules))
	for _, r := range rules {
		live[r.Vera] = true
		if i, ok := idx[r.Vera]; ok {
			e := &m.Rules[i]
			e.Object = r.Object
			e.Desc = r.Description
			if e.Gone {
				e.Gone = false // resurfaced upstream
			}
			continue
		}
		m.Rules = append(m.Rules, Entry{
			Vera:   r.Vera,
			Object: r.Object,
			Desc:   r.Description,
			Status: StatusTodo,
			Checks: []string{},
		})
		added = append(added, r.Vera)
	}
	for i := range m.Rules {
		if e := &m.Rules[i]; !live[e.Vera] && !e.Gone {
			e.Gone = true
			gone = append(gone, e.Vera)
		}
	}
	m.sort()
	return added, gone
}

// sort orders entries canonically by parsing the vera ID.
func (m *Manifest) sort() {
	sort.SliceStable(m.Rules, func(i, j int) bool {
		si, ci, ti := parseVera(m.Rules[i].Vera)
		sj, cj, tj := parseVera(m.Rules[j].Vera)
		return lessVera(si, ci, ti, sj, cj, tj)
	})
}
