package language

import (
	"fmt"
	"regexp"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// langTag is the RFC 3066 Language-Tag shape required by ISO 32000
// §14.9.2 (and mirrored verbatim by veraPDF's UA1:7.2-29 / UA2:8.4.4-2
// tests): a primary subtag of 1-8 ASCII letters, followed by any number
// of "-"-separated subtags of 1-8 letters or digits. This is a syntactic
// check only -- it does not verify the tag against the IANA registry.
var langTag = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)

// LangFormat fails when a /Lang entry that is present -- on the Catalog or
// on a structure element -- carries a value that is not a well-formed
// RFC 3066 language tag. An empty value is also a failure: the RFC 3066
// grammar has no empty production (veraPDF's atomic corpus agrees). The
// related "a language must be declared at all" requirement is UA-11-001 /
// UA-11-002; this check is purely about the format of whatever tags are
// declared.
//
// Marked-content property-list /Lang entries are not inspected -- the model
// surfaces language only at the Catalog and structure-element level -- so this
// check only partially covers the veraPDF 7.2-29 / 8.4.4-2 rules.
type LangFormat struct{}

func (LangFormat) ID() string                { return "UA-11-003" }
func (LangFormat) Title() string             { return "Declared /Lang values are well-formed language tags" }
func (LangFormat) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (LangFormat) Severity() engine.Severity { return engine.SeverityError }
func (LangFormat) Spec() engine.Spec         { return engine.SpecBoth }
func (LangFormat) WCAG() []string            { return []string{"3.1.1", "3.1.2"} }
func (LangFormat) Description() string {
	return "PDF/UA-1 §7.2 / PDF/UA-2 §8.4.4 require every /Lang value (on the Catalog or a structure element) to be a Language-Tag as defined in RFC 3066: 1-8 letters optionally followed by \"-\"-separated subtags of 1-8 letters or digits. A malformed tag (e.g. a 9-letter primary subtag or a leading digit) cannot be interpreted by assistive technology."
}

func (c LangFormat) Run(doc model.Document) []engine.Finding {
	var findings []engine.Finding
	checked := 0

	// Catalog /Lang. A present-but-empty value is still validated: the
	// RFC 3066 grammar has no empty production, so an empty /Lang is
	// malformed (veraPDF's 8.4.4-t02-fail-n treats it as a failure).
	if cat, err := doc.Catalog(); err == nil {
		if _, present := cat.Find("Lang"); present {
			v, _ := cat.String("Lang")
			checked++
			if !langTag.MatchString(v) {
				findings = append(findings, c.invalid(v, "Catalog", nil))
			}
		}
	}

	// Structure-element /Lang, document-wide.
	if root, err := doc.StructTreeRoot(); err == nil && root != nil {
		checked += c.walk(root, "/"+root.Type(), &findings)
	}

	if checked == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no /Lang entries to validate",
		}}
	}
	return findings
}

// walk validates /Lang on elem and its descendants, returning how many
// non-empty /Lang values it inspected.
func (c LangFormat) walk(elem model.StructElement, path string, findings *[]engine.Finding) int {
	checked := 0
	if elem.AttrPresent("Lang") {
		v := elem.Attr("Lang")
		checked++
		if !langTag.MatchString(v) {
			*findings = append(*findings, c.invalid(v, path, &engine.Location{Page: elem.Page(), StructPath: path}))
		}
	}
	for _, child := range elem.Children() {
		checked += c.walk(child, path+"/"+child.Type(), findings)
	}
	return checked
}

func (c LangFormat) invalid(value, where string, loc *engine.Location) engine.Finding {
	return engine.Finding{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  fmt.Sprintf("/Lang value %q on %s is not a well-formed RFC 3066 language tag", value, where),
		Hint:     "Use a valid language tag such as \"en\", \"en-US\" or \"de-DE\": a 1-8 letter primary subtag, optional \"-\"-separated subtags of 1-8 letters or digits.",
		Location: loc,
	}
}

func init() { engine.Register(LangFormat{}) }
