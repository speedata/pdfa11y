package verasync_test

import (
	"os"
	"testing"

	_ "github.com/speedata/pdfa11y/internal/checks"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/verasync"
)

// TestManifestNoDrift guards the checked-in coverage manifest against the
// pinned veraPDF profiles: it fails when upstream has added, removed, or
// renumbered a rule, or when the manifest is internally inconsistent (a
// covered rule with no checks, a checks reference to an unregistered check ID,
// etc.). Skipped when $PDFA11Y_VERAPDF is unset, so `go test ./...` stays green
// without the veraPDF checkout — same convention as the refcorpus test.
func TestManifestNoDrift(t *testing.T) {
	dir := os.Getenv(verasync.EnvProfiles)
	if dir == "" {
		t.Skipf("%s not set; skipping veraPDF drift check", verasync.EnvProfiles)
	}

	rules, err := verasync.LoadProfiles(dir)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	m, err := verasync.LoadManifest("coverage.yaml")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	registered := map[string]string{}
	for _, c := range engine.All() {
		registered[c.ID()] = c.Spec().String()
	}

	problems := m.Check(rules, registered)
	for _, p := range problems {
		t.Errorf("manifest drift: %s", p)
	}
	if len(problems) > 0 {
		t.Logf("run `verasync -update` then triage, and regenerate the report with `verasync -report`")
	}
}
