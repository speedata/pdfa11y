// Command verasync keeps pdfa11y's veraPDF coverage manifest in sync with the
// veraPDF validation profiles. It is a maintenance tool, not part of the
// runtime checker. See VERAPDF_SYNC_PLAN.md.
//
// Modes:
//
//	verasync -update   merge upstream rules into coverage.yaml (preserves
//	                   hand-maintained status/checks/note)
//	verasync -report   regenerate VERAPDF_COVERAGE.md from coverage.yaml
//	verasync -check    fail if the manifest has drifted from upstream or is
//	                   internally inconsistent (for CI)
//
// The veraPDF-validation-profiles checkout is located via $PDFA11Y_VERAPDF or
// the -profiles flag.
package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/speedata/pdfa11y/internal/checks"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/verasync"
)

const (
	defaultManifest = "internal/verasync/coverage.yaml"
	defaultReport   = "VERAPDF_COVERAGE.md"
)

func main() {
	var (
		update   = flag.Bool("update", false, "merge upstream veraPDF rules into the manifest")
		report   = flag.Bool("report", false, "regenerate the Markdown coverage report")
		check    = flag.Bool("check", false, "verify the manifest against upstream and exit non-zero on drift")
		profiles = flag.String("profiles", os.Getenv(verasync.EnvProfiles), "path to the veraPDF-validation-profiles checkout ($"+verasync.EnvProfiles+")")
		manifest = flag.String("manifest", defaultManifest, "path to the coverage manifest (YAML)")
		out      = flag.String("out", defaultReport, "path to the generated Markdown report")
	)
	flag.Parse()

	if !*update && !*report && !*check {
		fmt.Fprintln(os.Stderr, "verasync: pick a mode: -update, -report, or -check")
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*update, *report, *check, *profiles, *manifest, *out); err != nil {
		fmt.Fprintln(os.Stderr, "verasync:", err)
		os.Exit(1)
	}
}

func run(update, report, check bool, profilesDir, manifestPath, outPath string) error {
	// Modes that touch upstream need the profiles checkout.
	needUpstream := update || check
	var rules []verasync.Rule
	if needUpstream {
		if profilesDir == "" {
			return fmt.Errorf("no veraPDF profiles path (set $%s or -profiles)", verasync.EnvProfiles)
		}
		var err error
		rules, err = verasync.LoadProfiles(profilesDir)
		if err != nil {
			return err
		}
	}

	m, err := verasync.LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	if update {
		added, gone := m.Merge(rules)
		if err := m.Save(manifestPath); err != nil {
			return err
		}
		fmt.Printf("updated %s: %d rules, +%d new, %d gone\n", manifestPath, len(m.Rules), len(added), len(gone))
		for _, v := range added {
			fmt.Printf("  + %s\n", v)
		}
		for _, v := range gone {
			fmt.Printf("  ⚠ gone: %s\n", v)
		}
	}

	if check {
		problems := m.Check(rules, registeredCheckSpecs())
		if len(problems) > 0 {
			fmt.Fprintf(os.Stderr, "%d manifest problem(s):\n", len(problems))
			for _, p := range problems {
				fmt.Fprintf(os.Stderr, "  %s\n", p)
			}
			return fmt.Errorf("manifest is out of sync")
		}
		fmt.Printf("manifest ok: %d rules, no drift\n", len(m.Rules))
	}

	if report {
		if err := os.WriteFile(outPath, []byte(m.Report()), 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", outPath)
	}
	return nil
}

// registeredCheckSpecs maps every registered pdfa11y check ID to its Spec
// string, so the drift check can verify spec consistency.
func registeredCheckSpecs() map[string]string {
	specs := map[string]string{}
	for _, c := range engine.All() {
		specs[c.ID()] = c.Spec().String()
	}
	return specs
}
