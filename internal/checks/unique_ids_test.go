package checks_test

import (
	"testing"

	_ "github.com/speedata/pdfa11y/internal/checks"
	"github.com/speedata/pdfa11y/internal/engine"
)

// TestRegisteredIDsUnique guards against two checks registering the same ID.
// A collision silently breaks the verasync oracle and any ID-keyed lookup,
// because byID maps resolve to whichever check registered last.
func TestRegisteredIDsUnique(t *testing.T) {
	seen := map[string]string{}
	for _, c := range engine.All() {
		id := c.ID()
		if prev, ok := seen[id]; ok {
			t.Errorf("duplicate check ID %q: registered by both %s and %T", id, prev, c)
			continue
		}
		seen[id] = idType(c)
	}
}

func idType(c engine.Check) string {
	return c.Title()
}
