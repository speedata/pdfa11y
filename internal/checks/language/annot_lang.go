package language

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// AnnotLang fails when an annotation carries a /Contents entry but no natural
// language can be determined for it. ISO 14289-1 §7.2 (via ISO 32000-1
// §14.9.2) requires that language to be resolvable from:
//
//   - the annotation's own /Lang (inherited up the /Parent chain), or
//   - the /Lang of the structure element that owns the annotation (reached
//     through /StructParent and the /ParentTree, with ancestor inheritance), or
//   - the document catalog's /Lang.
//
// Covers veraPDF UA1:7.2-24 (annotation /Contents). The sibling form-field
// /TU rule (UA1:7.2-25) is intentionally not covered here: veraPDF's
// containsLang for a form field does not follow the same annotation/structure
// resolution, and the corpus (7.2-t25-pass-b vs -fail-a) cannot yet be
// distinguished by this model -- tracked as open.
//
// One finding per offending annotation. N/A when no annotation carries
// /Contents.
type AnnotLang struct{}

func (AnnotLang) ID() string                { return "UA-11-005" }
func (AnnotLang) Title() string             { return "Annotation /Contents has a determinable language" }
func (AnnotLang) Category() engine.Category { return engine.CategoryNaturalLanguage }
func (AnnotLang) Severity() engine.Severity { return engine.SeverityError }
func (AnnotLang) Spec() engine.Spec         { return engine.SpecBoth }
func (AnnotLang) WCAG() []string            { return []string{"3.1.2"} }
func (AnnotLang) Description() string {
	return "ISO 14289-1 §7.2 requires the natural language of an annotation's /Contents string to be determinable (ISO 32000-1 §14.9.2): via the annotation's /Lang, the /Lang of the structure element that owns it, or the catalog /Lang. Annotation text with no resolvable language cannot be voiced correctly by assistive technology."
}

func (c AnnotLang) Run(doc model.Document) []engine.Finding {
	catalogLang := false
	if cat, err := doc.Catalog(); err == nil {
		if lang, ok := cat.String("Lang"); ok && lang != "" {
			catalogLang = true
		}
	}

	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}

	var findings []engine.Finding
	considered := 0
	for _, a := range annots {
		if a.Contents == "" {
			continue
		}
		considered++
		// An annotation's /Contents is content anchored in the structure
		// tree, so the owning structure element's /Lang (StructLang)
		// determines its language, alongside the annotation's own /Lang and
		// the catalog /Lang.
		if catalogLang || a.Lang != "" || a.StructLang != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("annotation /Contents on page %d has no determinable language", a.Page),
			Hint:     "Set /Lang on the annotation, on the structure element that references it, or declare a document-wide /Lang on the catalog.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no annotation carries /Contents -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(AnnotLang{}) }
