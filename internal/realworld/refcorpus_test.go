package realworld_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/speedata/pdfa11y/internal/checks" // register every check
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// (no default corpus path -- PDFA11Y_REFCORPUS is the sole entry
// point. Tests skip silently when it is unset, so a fresh checkout
// runs `go test ./...` cleanly.)

// noiseCheckID is the MH check whose failure is systemic to the
// pdfa.org reference set: every file in that corpus declares XMP but
// omits the pdfuaid:part identifier. Counting that failure in the
// verdict would mask every per-file failing condition the corpus is
// actually testing. We exclude it for the comparison.
const noiseCheckID = "MH-06-003"

// failureFilenamePattern picks out failure-demonstration PDFs from the
// pdfa.org naming convention: `UA1_Tpdf-<group>_F<NN>.pdf` is a
// failure; `UA1_Tpdf-<group>_<NN>.pdf` (no "F") is a passing example.
var failureFilenamePattern = regexp.MustCompile(`_F\d+\.pdf$`)

// TestReferenceCorpus walks the pdfa.org reference set and tabulates
// pdfa11y's verdict against each file's filename-encoded expectation.
// The test never fails on a mismatch -- divergences from pdfa.org are
// expected (some failure conditions are PAC human checks; others are
// covered indirectly by a different MH ID). The point is to surface
// them in a report we can diff between iterations.
//
// Output: a Markdown report at the repo root (`CROSS_VALIDATION.md`,
// gitignored).
func TestReferenceCorpus(t *testing.T) {
	corpus := os.Getenv("PDFA11Y_REFCORPUS")
	if corpus == "" {
		t.Skip("PDFA11Y_REFCORPUS is unset; point it at a fresh download of https://pdfa.org/techniques-for-accessible-pdf/ (CC-BY-4.0)")
	}
	if _, err := os.Stat(corpus); errors.Is(err, fs.ErrNotExist) {
		t.Skipf("PDFA11Y_REFCORPUS=%s does not exist", corpus)
	}

	files, err := collectPDFs(corpus)
	if err != nil {
		t.Fatalf("walk corpus: %v", err)
	}
	if len(files) == 0 {
		t.Skipf("no .pdf files under %s", corpus)
	}

	results := make([]corpusResult, 0, len(files))
	for _, path := range files {
		results = append(results, evaluate(path, corpus))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].RelPath < results[j].RelPath })

	report := buildReport(corpus, results)
	out := repoRoot(t) + "/CROSS_VALIDATION.md"
	if err := os.WriteFile(out, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	t.Logf("wrote %s (%d files, %d mismatches)", out, len(results), countMismatches(results))
}

type corpusResult struct {
	AbsPath       string
	RelPath       string // relative to corpus root
	FileName      string // basename minus .pdf
	Group         string // first path segment under corpus root
	Expected      string // "pass" or "fail"
	Got           string // "pass" or "fail"; "load-error" if pdfa11y could not parse the file
	FailingChecks []string
	AllowedSkips  []string // checks listed in allowedFailures that fired (excluded from FailingChecks)
	LoadErr       string
}

func (r corpusResult) Match() bool { return r.Expected == r.Got }

func collectPDFs(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".pdf") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func evaluate(absPath, corpusRoot string) corpusResult {
	rel, _ := filepath.Rel(corpusRoot, absPath)
	r := corpusResult{
		AbsPath:  absPath,
		RelPath:  rel,
		FileName: strings.TrimSuffix(filepath.Base(absPath), ".pdf"),
		Group:    firstSegment(rel),
		Expected: expectationFromFilename(filepath.Base(absPath)),
	}
	doc, err := pdf.LoadFile(absPath)
	if err != nil {
		r.Got = "load-error"
		r.LoadErr = err.Error()
		return r
	}
	results := engine.Run(doc, engine.All())
	allowed := allowedSet(r.RelPath)
	var failing, skipped []string
	for _, res := range results {
		if res.State() != engine.VerdictFail {
			continue
		}
		id := res.Check.ID()
		if id == noiseCheckID {
			continue
		}
		if allowed[id] {
			skipped = append(skipped, id)
			continue
		}
		failing = append(failing, id)
	}
	sort.Strings(failing)
	sort.Strings(skipped)
	r.FailingChecks = failing
	r.AllowedSkips = skipped
	if len(failing) > 0 {
		r.Got = "fail"
	} else {
		r.Got = "pass"
	}
	return r
}

// allowedSet builds a lookup of MH-IDs whose failure is acceptable on
// the given file. Falls back to an empty set when the file is not in
// the expectations table.
func allowedSet(rel string) map[string]bool {
	out := map[string]bool{}
	for _, id := range allowedFailures[rel] {
		out[id] = true
	}
	return out
}

func expectationFromFilename(name string) string {
	if failureFilenamePattern.MatchString(name) {
		return "fail"
	}
	return "pass"
}

func firstSegment(rel string) string {
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func countMismatches(rs []corpusResult) int {
	n := 0
	for _, r := range rs {
		if !r.Match() {
			n++
		}
	}
	return n
}

func buildReport(corpus string, rs []corpusResult) string {
	var matches, mismatches int
	var expPass, expFail int
	var agreePass, agreeFail int
	var falseNeg, falsePos, loadErr int
	for _, r := range rs {
		if r.Expected == "pass" {
			expPass++
		} else {
			expFail++
		}
		switch {
		case r.Match() && r.Expected == "pass":
			agreePass++
			matches++
		case r.Match() && r.Expected == "fail":
			agreeFail++
			matches++
		case r.Got == "load-error":
			loadErr++
			mismatches++
		case r.Expected == "fail" && r.Got == "pass":
			falseNeg++
			mismatches++
		case r.Expected == "pass" && r.Got == "fail":
			falsePos++
			mismatches++
		}
	}

	rate := 0.0
	if len(rs) > 0 {
		rate = 100.0 * float64(matches) / float64(len(rs))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Cross-validation against pdfa.org reference corpus\n\n")
	fmt.Fprintf(&b, "Generated %s.\n\n", time.Now().Format("2006-01-02 15:04 MST"))
	fmt.Fprintf(&b, "Corpus: `%s` (%d PDFs).\n\n", corpus, len(rs))
	fmt.Fprintf(&b, "Reference PDFs © pdfa.org, CC-BY-4.0. pdfa.org asks consumers\n")
	fmt.Fprintf(&b, "to re-fetch the files from <https://pdfa.org/techniques-for-accessible-pdf/>\n")
	fmt.Fprintf(&b, "rather than caching them, since the samples may change during the\n")
	fmt.Fprintf(&b, "standard's development phase.\n\n")
	fmt.Fprintf(&b, "`%s` (PDF/UA identifier missing in XMP) is excluded from the\n", noiseCheckID)
	fmt.Fprintf(&b, "verdict because the pdfa.org reference set systematically omits it\n")
	fmt.Fprintf(&b, "across all files; counting it would mask the per-failure assessment.\n\n")

	fmt.Fprintf(&b, "pdfa.org's `_NN` PASS examples demonstrate one technique correctly\n")
	fmt.Fprintf(&b, "but are not full-conformance fixtures -- they may carry unrelated\n")
	fmt.Fprintf(&b, "defects that pdfa11y rightly flags. Such per-file orthogonal\n")
	fmt.Fprintf(&b, "findings are listed in `refcorpus_expectations_test.go` and excluded\n")
	fmt.Fprintf(&b, "from the false-positive count. %d file(s) carry an entry there.\n\n", len(allowedFailures))

	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "| | Expected pass | Expected fail | Total |\n")
	fmt.Fprintf(&b, "|---|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| pdfa11y agrees | %d | %d | %d |\n", agreePass, agreeFail, matches)
	fmt.Fprintf(&b, "| pdfa11y disagrees (false neg: PASS instead of FAIL) | — | %d | %d |\n", falseNeg, falseNeg)
	fmt.Fprintf(&b, "| pdfa11y disagrees (false pos: FAIL instead of PASS) | %d | — | %d |\n", falsePos, falsePos)
	if loadErr > 0 {
		fmt.Fprintf(&b, "| load errors | — | — | %d |\n", loadErr)
	}
	fmt.Fprintf(&b, "| **Total** | **%d** | **%d** | **%d** |\n\n", expPass, expFail, len(rs))
	fmt.Fprintf(&b, "Match rate: **%.1f %%**.\n\n", rate)

	fmt.Fprintf(&b, "## Mismatches\n\n")

	if falseNeg > 0 {
		fmt.Fprintf(&b, "### False negatives (pdfa11y PASS where pdfa.org expects FAIL)\n\n")
		fmt.Fprintf(&b, "These are failure conditions that pdfa11y does not yet detect, or that map to a Matterhorn *human check* outside the automatable subset.\n\n")
		fmt.Fprintf(&b, "| Path | File | Description (from directory name) |\n")
		fmt.Fprintf(&b, "|---|---|---|\n")
		for _, r := range rs {
			if r.Match() || r.Got == "load-error" {
				continue
			}
			if !(r.Expected == "fail" && r.Got == "pass") {
				continue
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s |\n", r.Group, r.FileName, descriptionFromPath(r.RelPath))
		}
		fmt.Fprintln(&b)
	}

	if falsePos > 0 {
		fmt.Fprintf(&b, "### False positives (pdfa11y FAIL where pdfa.org expects PASS)\n\n")
		fmt.Fprintf(&b, "These are stricter-than-PAC findings. Document each one as either a real spec violation that PAC misses, or a heuristic over-reach that should be relaxed.\n\n")
		fmt.Fprintf(&b, "| Path | File | Spurious failing checks |\n")
		fmt.Fprintf(&b, "|---|---|---|\n")
		for _, r := range rs {
			if r.Match() || r.Got == "load-error" {
				continue
			}
			if !(r.Expected == "pass" && r.Got == "fail") {
				continue
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s |\n", r.Group, r.FileName, strings.Join(r.FailingChecks, ", "))
		}
		fmt.Fprintln(&b)
	}

	if loadErr > 0 {
		fmt.Fprintf(&b, "### Load errors\n\n")
		fmt.Fprintf(&b, "Files that pdfa11y could not parse at all.\n\n")
		fmt.Fprintf(&b, "| Path | File | Error |\n")
		fmt.Fprintf(&b, "|---|---|---|\n")
		for _, r := range rs {
			if r.Got != "load-error" {
				continue
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s |\n", r.Group, r.FileName, r.LoadErr)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## Per-file results\n\n")
	fmt.Fprintf(&b, "| Group | File | Expected | pdfa11y | Match | Failing checks | Allowed (skipped) |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|---|\n")
	for _, r := range rs {
		marker := "✓"
		if !r.Match() {
			marker = "✗"
		}
		failing := strings.Join(r.FailingChecks, ", ")
		if failing == "" {
			failing = "—"
		}
		allowed := strings.Join(r.AllowedSkips, ", ")
		if allowed == "" {
			allowed = "—"
		}
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s | %s | %s | %s |\n",
			r.Group, r.FileName, r.Expected, r.Got, marker, failing, allowed)
	}
	return b.String()
}

// descriptionFromPath turns the parent-directory name into a human
// description. pdfa.org's naming uses dashes to separate words and a
// `G2_F03-` style prefix to identify the case.
func descriptionFromPath(rel string) string {
	parent := filepath.Base(filepath.Dir(rel))
	// Strip the leading "<group>_<case>-" prefix, e.g. "G2_F03-".
	if i := strings.Index(parent, "-"); i >= 0 && i < 10 {
		parent = parent[i+1:]
	}
	return strings.ReplaceAll(parent, "-", " ")
}

// repoRoot finds the project root for writing the report. It walks up
// from the test file's working directory looking for go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}
