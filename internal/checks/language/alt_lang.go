package language

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// AltLang fails when a structure element carries an /Alt, /ActualText or /E
// attribute but no natural language can be determined for that text. ISO
// 14289-1 §7.2 (via ISO 32000-1 §14.9.2) requires the language of these
// alternate-text strings to be determinable: from the element's own /Lang,
// from an ancestor's /Lang, or from the document catalog's /Lang. Without any
// of these, assistive technology cannot pick the right voice/pronunciation for
// the description.
//
// Covers veraPDF UA1:7.2-21 (ActualText), UA1:7.2-22 (Alt) and UA1:7.2-23 (E).
// Marked-content (Span) variants (UA1:7.2-30..-32) are not reachable through
// the model and stay out of scope, as with UA-11-003.
//
// One finding per offending attribute. N/A when no element carries /Alt,
// /ActualText or /E.
type AltLang struct{}

func (AltLang) ID() string                { return "UA-11-004" }
func (AltLang) Title() string             { return "Alt, ActualText and E have a determinable language" }
func (AltLang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (AltLang) Severity() engine.Severity { return engine.SeverityError }
func (AltLang) Spec() engine.Spec         { return engine.SpecBoth }
func (AltLang) WCAG() []string            { return []string{"3.1.2"} }
func (AltLang) Description() string {
	return "ISO 14289-1 §7.2 requires the natural language of an /Alt, /ActualText or /E string to be determinable (ISO 32000-1 §14.9.2): via the element's /Lang, an ancestor's /Lang, or the catalog /Lang. Alternate text with no resolvable language cannot be voiced correctly by assistive technology."
}

// altAttrs are the alternate-text attributes whose language must be
// determinable.
var altAttrs = []string{"Alt", "ActualText", "E"}

func (c AltLang) Run(doc model.Document) []engine.Finding {
	// A catalog /Lang determines the language for the whole document, so it
	// satisfies every element at once.
	catalogLang := false
	if cat, err := doc.Catalog(); err == nil {
		if lang, ok := cat.String("Lang"); ok && lang != "" {
			catalogLang = true
		}
	}

	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	seen := 0
	c.walk(root, "/"+root.Type(), "", catalogLang, &findings, &seen)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no element carries /Alt, /ActualText or /E -- nothing to inspect",
		}}
	}
	return findings
}

// walk descends the tree carrying inheritedLang, the nearest ancestor /Lang
// (empty when none). catalogLang short-circuits every check.
func (c AltLang) walk(elem model.StructElement, path, inheritedLang string, catalogLang bool, out *[]engine.Finding, seen *int) {
	lang := inheritedLang
	if own := elem.Attr("Lang"); own != "" {
		lang = own
	}

	for _, attr := range altAttrs {
		if !elem.AttrPresent(attr) {
			continue
		}
		*seen++
		if catalogLang || lang != "" {
			continue
		}
		*out = append(*out, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("/%s at %s has no determinable language (no /Lang on the element, any ancestor, or the catalog)", attr, path),
			Hint:     "Set /Lang on the element (or an ancestor), or declare a document-wide /Lang on the catalog.",
			Location: &engine.Location{Page: elem.Page(), StructPath: path},
		})
	}

	for _, ch := range elem.Children() {
		c.walk(ch, path+"/"+ch.Type(), lang, catalogLang, out, seen)
	}
}

func init() { engine.Register(AltLang{}) }
