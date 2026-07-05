package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// TrapNet fails when a PDF/UA-1 document carries a TrapNet annotation that is
// actually presented to the reader. ISO 14289-1 §7.18.2 forbids TrapNet
// annotations; veraPDF UA1:7.18.2-1 relaxes this to the visible case only
// (the annotation is acceptable when hidden via /F or placed outside the crop
// box), because a trap network that is never shown carries no accessibility
// surface.
//
// PDF/UA-2 forbids TrapNet outright (no visibility exception) via UA-28-009,
// so this check is PDF/UA-1 only to avoid double-reporting.
//
// One finding per offending annotation. N/A when the document has no TrapNet
// annotations at all.
type TrapNet struct{}

func (TrapNet) ID() string                { return "UA-28-012" }
func (TrapNet) Title() string             { return "No visible TrapNet annotations" }
func (TrapNet) Category() engine.Category { return engine.CategoryInteractive }
func (TrapNet) Severity() engine.Severity { return engine.SeverityError }
func (TrapNet) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (TrapNet) WCAG() []string            { return []string{"1.3.1"} }
func (TrapNet) Description() string {
	return "ISO 14289-1 §7.18.2 forbids TrapNet (trap network) annotations. A TrapNet describes colour-trapping for print production and carries no meaning for assistive technology; when it is presented on the page it becomes a phantom, undescribable annotation. A TrapNet that is hidden (/F flag) or lies outside the crop box is tolerated because it is not shown."
}

func (c TrapNet) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	seen := 0
	for _, a := range annots {
		if a.Subtype != "TrapNet" {
			continue
		}
		seen++
		// Hidden or off-page TrapNets are tolerated: they are not shown.
		if a.Hidden || a.OffPage {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("TrapNet annotation on page %d is presented to the reader", a.Page),
			Hint:     "Remove the TrapNet annotation (trap networks belong in print production, not in a conforming PDF/UA file), or hide it via the /F flag if it must remain.",
			Location: &engine.Location{Page: a.Page},
		})
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no TrapNet annotations -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(TrapNet{}) }
