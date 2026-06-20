// Package taborder groups checks that inspect the /Tabs key on
// page objects (annotation tab-order policy).
package taborder

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// PageTabs fails for every page whose effective /Tabs value is
// disallowed by the document's declared PDF/UA part.
//
// PDF/UA-1 §7.18 / Matterhorn 08-001 require /Tabs = "S"
// (Structure order).
// PDF/UA-2 §8.9.3.3 broadened the rule: pages that include
// annotations must declare /Tabs as "A" (Annotation), "W"
// (Widget) or "S". The legacy "R" (Row) and "C" (Column) values
// are still disallowed everywhere.
//
// /Tabs is inheritable from /Pages ancestors per ISO 32000-1
// §7.7.3.4, so the walker resolves the effective value before
// the check sees the page.
//
// Severity Error: this is one of the most common PDF/UA defects
// in real-world tagged PDFs because authoring tools default
// /Tabs to "R" (the legacy reading-order policy) and producers
// forget to override it.
type PageTabs struct{}

func (PageTabs) ID() string                { return "UA-08-001" }
func (PageTabs) Title() string             { return "Pages declare an accessible /Tabs value" }
func (PageTabs) Category() engine.Category { return engine.CategoryTabOrder }
func (PageTabs) Severity() engine.Severity { return engine.SeverityError }
func (PageTabs) Spec() engine.Spec         { return engine.SpecBoth }
func (PageTabs) WCAG() []string            { return []string{"2.4.3"} }
func (PageTabs) Description() string {
	return "PDF/UA-1 §7.18 / Matterhorn 08-001 require /Tabs = S so annotation tab order follows the structure tree. PDF/UA-2 §8.9.3.3 widens the allowed set to S, A (Annotation order) or W (Widget order). The legacy R (row) and C (column) values are disallowed everywhere: they bypass the structure tree and produce reading orders that AT cannot rely on. /Tabs is inheritable, so the effective value (own /Tabs, else the closest /Pages ancestor's /Tabs) is what counts."
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

	// Spec gating: under PDF/UA-2 the allowed set is {S, A, W}.
	// Without an explicit pdfuaid:part = 2 we keep the strict
	// UA-1 rule (only S) so legacy documents continue to behave
	// the way they always did.
	allowed := map[string]bool{"S": true}
	expected := `"S"`
	if part, found, err := pdfua.DetectPart(doc); err == nil && found && part == 2 {
		allowed["A"] = true
		allowed["W"] = true
		expected = `one of "S", "A", "W"`
	}

	var findings []engine.Finding
	for _, p := range pages {
		switch {
		case allowed[p.Tabs]:
			// OK
		case p.Tabs == "":
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d has no /Tabs entry (inherited or own)", p.Number),
				Hint:     "Set /Tabs /S on the page (or on its /Pages ancestor for whole-document coverage) so annotation tab order is well defined.",
				Location: &engine.Location{Page: p.Number},
			})
		default:
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("page %d has /Tabs = %q, PDF/UA requires %s", p.Number, p.Tabs, expected),
				Hint:     "Change /Tabs to /S (or, under PDF/UA-2, /A or /W). Row (R) and Column (C) tab policies bypass the structure tree and are not PDF/UA-conformant.",
				Location: &engine.Location{Page: p.Number},
			})
		}
	}
	return findings
}

func init() { engine.Register(PageTabs{}) }
