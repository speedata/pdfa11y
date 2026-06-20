// Package navigation groups checks that inspect document-level
// navigation aids: outlines (bookmarks), page labels, tab order.
package navigation

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// outlineThreshold is the number of pages above which the document
// is expected to carry an /Outlines tree. PDF/UA-1 §7.17 / Matterhorn
// 27-001 leave the threshold deliberately fuzzy ("documents that
// benefit from navigation aids"); 21 pages is a widely used
// convention -- short enough that any real reading document hits it,
// long enough that single-page forms and reports are not falsely
// flagged.
const outlineThreshold = 21

// Outlines warns when a document with more than outlineThreshold
// pages has no /Outlines entry in its catalog (or an empty outline
// tree). Outlines provide the bookmark navigation that PDF readers
// expose in a sidebar; without them users on assistive technology
// have no quick jump targets and must read the document linearly.
//
// Severity Warning rather than Error: the threshold is conventional,
// not spec-mandated, and very long technical documents may
// legitimately defer outline building. The warning makes the
// omission visible without blocking conformance.
type Outlines struct{}

func (Outlines) ID() string                { return "UA-27-001" }
func (Outlines) Title() string             { return "Multi-page documents declare /Outlines" }
func (Outlines) Category() engine.Category { return engine.CategoryNavigation }
func (Outlines) Severity() engine.Severity { return engine.SeverityWarning }
func (Outlines) Spec() engine.Spec         { return engine.SpecBoth }
func (Outlines) WCAG() []string            { return []string{"2.4.5"} }
func (Outlines) Description() string {
	return "PDF/UA-1 §7.17 calls for navigation aids on documents that benefit from them. Outlines (bookmarks) are the canonical aid: they give AT users jump targets without linear reading. pdfa11y flags documents over 21 pages that lack an /Outlines entry on the catalog or whose outline tree is empty."
}

func (c Outlines) Run(doc model.Document) []engine.Finding {
	pages := doc.PageCount()
	if pages <= outlineThreshold {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  fmt.Sprintf("document has %d page(s); the outline check applies above %d", pages, outlineThreshold),
		}}
	}
	cat, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	outlinesObj, ok := cat.Find("Outlines")
	if !ok {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityWarning,
			Message:  fmt.Sprintf("%d-page document has no /Outlines entry in the catalog", pages),
			Hint:     "Add a bookmark / outline tree so users (especially AT users) can navigate between sections without linear reading.",
		}}
	}
	outlines, err := doc.DereferenceDict(outlinesObj)
	if err != nil || outlines == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityWarning,
			Message:  fmt.Sprintf("%d-page document declares /Outlines but it could not be resolved", pages),
			Hint:     "Ensure the /Outlines indirect reference points at a valid outline dictionary with a /First entry.",
		}}
	}
	if _, hasFirst := outlines.Find("First"); !hasFirst {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityWarning,
			Message:  fmt.Sprintf("%d-page document has an empty /Outlines (no /First entry)", pages),
			Hint:     "An outline dictionary without /First contains no bookmarks. Populate it with at least the top-level section headings.",
		}}
	}
	return nil
}

func init() { engine.Register(Outlines{}) }
