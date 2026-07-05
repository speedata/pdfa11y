// Package verasync bridges pdfa11y's hand-written checks against veraPDF's
// PDF/UA validation profiles. It is a thin coverage/drift layer, not a runtime
// dependency: the profile XML is used only as a data source for a machine-
// readable gap map (the coverage manifest). See VERAPDF_SYNC_PLAN.md.
//
// The profile XML (veraPDF validation profiles) is licensed CC BY 4.0; rule
// descriptions and clause/test identifiers reproduced in the manifest carry
// the attribution recorded in THIRD_PARTY_LICENSES.md.
package verasync

import (
	"encoding/xml"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// EnvProfiles is the environment variable naming the veraPDF-validation-profiles
// checkout. Analogous to PDFA11Y_REFCORPUS.
const EnvProfiles = "PDFA11Y_VERAPDF"

// EnvCorpus optionally names the veraPDF-corpus checkout (atomic test PDFs).
const EnvCorpus = "PDFA11Y_VERAPDF_CORPUS"

// Rule is a single veraPDF validation rule, distilled from one <rule> element.
type Rule struct {
	Vera        string // canonical ID, e.g. "UA2:8.2.5.26-5"
	Spec        string // "UA1" or "UA2"
	Object      string // veraPDF model object the rule tests, e.g. "PDStructElem"
	Tags        string // rule tags, e.g. "metadata"
	Clause      string // ISO clause, e.g. "8.2.5.26"
	TestNumber  int    // test number within the clause
	Description string // human-readable requirement text (CC BY 4.0)
	Test        string // veraPDF test expression (reference only, not evaluated)
	Message     string // error message template
}

// profileXML mirrors the veraPDF validation profile format just enough to
// extract the fields we care about.
type profileXML struct {
	Flavour string    `xml:"flavour,attr"`
	Rules   []ruleXML `xml:"rules>rule"`
}

type ruleXML struct {
	Object      string  `xml:"object,attr"`
	Tags        string  `xml:"tags,attr"`
	ID          idXML   `xml:"id"`
	Description string  `xml:"description"`
	Test        string  `xml:"test"`
	Message     string  `xml:"error>message"`
}

type idXML struct {
	Spec       string `xml:"specification,attr"`
	Clause     string `xml:"clause,attr"`
	TestNumber string `xml:"testNumber,attr"`
}

// specToken maps a veraPDF <id specification=...> value to our canonical spec
// token. Only the ISO 14289 parts are meaningful here; other specifications
// referenced by a rule appear under <references>, not <id>.
func specToken(idSpec string) (string, error) {
	switch idSpec {
	case "ISO_14289_1":
		return "UA1", nil
	case "ISO_14289_2":
		return "UA2", nil
	default:
		return "", fmt.Errorf("unexpected id specification %q", idSpec)
	}
}

// ParseProfile reads a veraPDF PDF/UA profile XML and returns its rules with
// canonical IDs, sorted by (spec, clause, testNumber).
func ParseProfile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p profileXML
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	rules := make([]Rule, 0, len(p.Rules))
	for _, r := range p.Rules {
		spec, err := specToken(r.ID.Spec)
		if err != nil {
			return nil, fmt.Errorf("%s: rule clause %s test %s: %w", path, r.ID.Clause, r.ID.TestNumber, err)
		}
		tn, err := strconv.Atoi(r.ID.TestNumber)
		if err != nil {
			return nil, fmt.Errorf("%s: clause %s: non-numeric testNumber %q", path, r.ID.Clause, r.ID.TestNumber)
		}
		rules = append(rules, Rule{
			Vera:        fmt.Sprintf("%s:%s-%d", spec, r.ID.Clause, tn),
			Spec:        spec,
			Object:      r.Object,
			Tags:        r.Tags,
			Clause:      r.ID.Clause,
			TestNumber:  tn,
			Description: normalizeSpace(r.Description),
			Test:        strings.TrimSpace(r.Test),
			Message:     normalizeSpace(r.Message),
		})
	}
	SortRules(rules)
	return rules, nil
}

// LoadProfiles reads the PDFUA-1.xml and PDFUA-2.xml profiles from the
// veraPDF-validation-profiles checkout rooted at dir and returns all rules.
func LoadProfiles(dir string) ([]Rule, error) {
	var all []Rule
	for _, name := range []string{"PDF_UA/PDFUA-1.xml", "PDF_UA/PDFUA-2.xml"} {
		rules, err := ParseProfile(dir + "/" + name)
		if err != nil {
			return nil, err
		}
		all = append(all, rules...)
	}
	SortRules(all)
	return all, nil
}

// SortRules orders rules by spec (UA1 before UA2), then clause numerically,
// then test number — a stable order for diffable manifests.
func SortRules(rules []Rule) {
	sort.SliceStable(rules, func(i, j int) bool {
		return lessVera(rules[i].Spec, rules[i].Clause, rules[i].TestNumber,
			rules[j].Spec, rules[j].Clause, rules[j].TestNumber)
	})
}

func lessVera(specA, clauseA string, testA int, specB, clauseB string, testB int) bool {
	if specA != specB {
		return specA < specB // "UA1" < "UA2"
	}
	if c := compareClause(clauseA, clauseB); c != 0 {
		return c < 0
	}
	return testA < testB
}

// compareClause compares dotted clause numbers component-wise and numerically
// (so "8.2.5.26" sorts after "8.2.5.2", not before it).
func compareClause(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		if ea == nil && eb == nil {
			if na != nb {
				return na - nb
			}
			continue
		}
		if c := strings.Compare(pa[i], pb[i]); c != 0 {
			return c
		}
	}
	return len(pa) - len(pb)
}

// parseVera splits a canonical vera ID ("UA2:8.2.5.26-5") back into its spec,
// clause, and test number. Malformed IDs sort last (test number 0, clause "").
func parseVera(vera string) (spec, clause string, test int) {
	spec, rest, ok := strings.Cut(vera, ":")
	if !ok {
		return vera, "", 0
	}
	if dash := strings.LastIndexByte(rest, '-'); dash >= 0 {
		clause = rest[:dash]
		test, _ = strconv.Atoi(rest[dash+1:])
		return spec, clause, test
	}
	return spec, rest, 0
}

// normalizeSpace collapses runs of whitespace (including newlines from wrapped
// XML text) into single spaces and trims the result.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
