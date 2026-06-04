package engine

import "sort"

var registry []Check

// Register adds a check to the global registry. Intended to be called from
// each check package's init().
func Register(c Check) {
	registry = append(registry, c)
}

// All returns all registered checks sorted by ID.
func All() []Check {
	out := make([]Check, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// ForSpec returns the registered checks that apply to the given spec.
func ForSpec(s Spec) []Check {
	var out []Check
	for _, c := range All() {
		if c.Spec()&s != 0 {
			out = append(out, c)
		}
	}
	return out
}
