// Package optionalcontent groups checks that inspect optional
// content (PDF layers, /OCProperties).
package optionalcontent

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// OCGName fails for every OCG (Optional Content Group, i.e. PDF
// layer) that does not declare a /Name. PDF/UA-1 §7.13 and ISO
// 32000-1 §8.11.2 require OCGs to carry a human-readable name so
// the layer panel in AT-aware viewers can present the user a
// meaningful choice ("Drawing geometry", "Annotations", "Background
// map"). Unnamed layers either appear blank in the panel or are
// labelled with a synthetic identifier that is not meaningful.
type OCGName struct{}

func (OCGName) ID() string                { return "MH-20-001" }
func (OCGName) Title() string             { return "Optional Content Groups declare /Name" }
func (OCGName) Category() engine.Category { return engine.CategoryOptionalContent }
func (OCGName) Severity() engine.Severity { return engine.SeverityError }
func (OCGName) Spec() engine.Spec         { return engine.SpecBoth }
func (OCGName) WCAG() []string            { return []string{"1.3.1"} }
func (OCGName) Description() string {
	return "PDF/UA-1 §7.13 / ISO 32000-1 §8.11.2 require every Optional Content Group to expose a /Name string so the user can identify the layer in the viewer's layer panel. Without /Name AT presents unnamed layers without a meaningful label."
}

func (c OCGName) Run(doc model.Document) []engine.Finding {
	groups, err := doc.OptionalContentGroups()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate optional content groups: " + err.Error(),
		}}
	}
	if len(groups) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no Optional Content Groups -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for i, g := range groups {
		if g.Name == "" {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("Optional Content Group #%d has no /Name", i+1),
				Hint:     "Set /Name on the OCG dictionary to a short human-readable label (e.g. 'Drawing geometry', 'Background map').",
			})
		}
	}
	return findings
}

func init() { engine.Register(OCGName{}) }
