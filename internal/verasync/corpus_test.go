package verasync_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	_ "github.com/speedata/pdfa11y/internal/checks"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
	"github.com/speedata/pdfa11y/internal/verasync"
)

// corpusFile is a single atomic veraPDF test PDF.
type corpusFile struct {
	vera string // canonical rule ID the file exercises, e.g. "UA2:8.7-2"
	pass bool   // true = conforming (no error expected), false = seeded defect
	path string
}

// veraPDFs whose clause/test the filename encodes, e.g.
//
//	PDF_UA-2/8.7 Optional content/8.7-t02-fail-a.pdf -> UA2:8.7-2, fail
var corpusNameRE = regexp.MustCompile(`^(.+)-t(\d+)-(pass|fail)-[^.]+\.pdf$`)

// TestCorpusOracle checks every `covered` manifest rule against veraPDF's
// atomic corpus: each matching pass-PDF must produce no error from the mapped
// checks, each fail-PDF must produce at least one. It is the quality gate for
// the coverage claims (VERAPDF_SYNC_PLAN.md phase 3). Skipped when
// $PDFA11Y_VERAPDF_CORPUS is unset, so `go test ./...` stays green without the
// corpus checkout.
func TestCorpusOracle(t *testing.T) {
	root := os.Getenv(verasync.EnvCorpus)
	if root == "" {
		t.Skipf("%s not set; skipping veraPDF corpus oracle", verasync.EnvCorpus)
	}

	m, err := verasync.LoadManifest("coverage.yaml")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	byID := map[string]engine.Check{}
	for _, c := range engine.All() {
		byID[c.ID()] = c
	}

	files, err := collectCorpus(root)
	if err != nil {
		t.Fatalf("scan corpus: %v", err)
	}
	byVera := map[string][]corpusFile{}
	for _, f := range files {
		byVera[f.vera] = append(byVera[f.vera], f)
	}

	var covered, exercised, passOK, failOK, loadSkip int
	for _, e := range m.Rules {
		if e.Status != verasync.StatusCovered {
			continue
		}
		covered++
		matches := byVera[e.Vera]
		if len(matches) == 0 {
			continue // no atomic PDF for this rule
		}
		checks := make([]engine.Check, 0, len(e.Checks))
		for _, id := range e.Checks {
			if c := byID[id]; c != nil {
				checks = append(checks, c)
			}
		}
		if len(checks) == 0 {
			continue
		}
		exercised++

		for _, f := range matches {
			doc, err := pdf.LoadFile(f.path)
			if err != nil {
				loadSkip++
				t.Logf("SKIP (load error) %s [%s]: %v", rel(root, f.path), e.Vera, err)
				continue
			}
			gotError := false
			for _, r := range engine.Run(doc, checks) {
				if !r.Passed() {
					gotError = true
					break
				}
			}
			switch {
			case f.pass && gotError:
				t.Errorf("FALSE POSITIVE: %s [%s via %v] is a pass-PDF but a mapped check flagged an error",
					rel(root, f.path), e.Vera, e.Checks)
			case !f.pass && !gotError:
				t.Errorf("MISS: %s [%s via %v] is a fail-PDF but no mapped check produced an error",
					rel(root, f.path), e.Vera, e.Checks)
			case f.pass:
				passOK++
			default:
				failOK++
			}
		}
	}

	t.Logf("corpus oracle: %d covered rules, %d exercised by the corpus; %d pass-PDFs ok, %d fail-PDFs ok, %d load-skipped",
		covered, exercised, passOK, failOK, loadSkip)
}

// collectCorpus walks the veraPDF-corpus PDF_UA-1/PDF_UA-2 trees and returns
// every atomic test PDF whose name encodes a clause/test and pass/fail verdict.
func collectCorpus(root string) ([]corpusFile, error) {
	var out []corpusFile
	for spec, sub := range map[string]string{"UA1": "PDF_UA-1", "UA2": "PDF_UA-2"} {
		base := filepath.Join(root, sub)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			mm := corpusNameRE.FindStringSubmatch(info.Name())
			if mm == nil {
				return nil
			}
			tn, err := strconv.Atoi(mm[2])
			if err != nil {
				return nil
			}
			out = append(out, corpusFile{
				vera: spec + ":" + mm[1] + "-" + strconv.Itoa(tn),
				pass: mm[3] == "pass",
				path: path,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, nil
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return strings.TrimPrefix(path, root)
}
