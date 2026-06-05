// Package taborder groups checks that inspect the /Tabs key on
// page objects (annotation tab-order policy).
package taborder

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// PageTabs fails for every page that does not resolve /Tabs to
// "S" (Structure order). PDF/UA-1 §7.18 and Matterhorn 08-001
// require the structure tree to drive annotation tab order; "R"
// (Row) and "C" (Column) are legal PDF but disallowed by PDF/UA.
// /Tabs is inheritable from /Pages ancestors per ISO 32000-1
// §7.7.3.4, so the walker resolves the effective value before the
// check sees the page.
//
// Severity Error: this is one of the most common PDF/UA defects
// in real-world tagged PDFs because authoring tools default /Tabs
// to "R" (the legacy reading-order policy) and producers forget
// to override it.
type PageTabs struct{}

func (PageTabs) ID() string                { return "MH-08-001" }
func (PageTabs) Title() string             { return "Pages declare /Tabs = S (structure order)" }
func (PageTabs) Category() engine.Category { return engine.CategoryTabOrder }
func (PageTabs) Severity() engine.Severity { return engine.SeverityError }
func (PageTabs) Spec() engine.Spec         { return engine.SpecBoth }
func (PageTabs) WCAG() []string            { return []string{"2.4.3"} }
func (PageTabs) Description() string {
	return "PDF/UA-1 §7.18 / Matterhorn 08-001 require every page to set /Tabs to \"S\" so annotation tab order follows the structure tree. \"R\" (row) and \"C\" (column) are valid PDF but disallowed by PDF/UA: they bypass the structure tree and produce reading orders that AT cannot rely on. /Tabs is inheritable, so the effective value (own /Tabs, else the closest /Pages ancestor's /Tabs) is what counts."
}

func (c PageTabs) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no pages -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, p := range pages {
		switch p.Tabs {
		case "S":
			// OK
		case "":
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d has no /Tabs entry (inherited or own)", p.Number),
				Hint:     "Set /Tabs /S on the page (or on its /Pages ancestor for whole-document coverage) so annotation tab order follows the structure tree.",
				Location: &engine.Location{Page: p.Number},
			})
		default:
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d has /Tabs = %q, PDF/UA requires %q", p.Number, p.Tabs, "S"),
				Hint:     "Change /Tabs to /S. Row (R) and Column (C) tab policies bypass the structure tree and are not PDF/UA-conformant.",
				Location: &engine.Location{Page: p.Number},
			})
		}
	}
	return findings
}

func init() { engine.Register(PageTabs{}) }
