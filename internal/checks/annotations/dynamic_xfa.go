package annotations

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// DynamicXFA fails when the document uses a dynamic XFA form. ISO 14289-1 §7.15
// (veraPDF UA1:7.15-1): dynamic XFA forms shall not be used. A dynamic XFA form
// re-renders its layout at open time from the XFA template, bypassing the
// tagged page content entirely, so assistive technology has nothing to read.
// The form is dynamic when its XFA config packet sets <dynamicRender>required.
//
// PDF/UA-1 only: PDF 2.0 deprecates XFA outright and ISO 14289-2 §8.10.1
// forbids any XFA (static or dynamic) -- that stricter prohibition is
// UA-28-010. A static XFA form is permitted under PDF/UA-1, so this check
// passes it.
//
// N/A when the document has no XFA form.
type DynamicXFA struct{}

func (DynamicXFA) ID() string                { return "UA-28-028" }
func (DynamicXFA) Title() string             { return "Document does not use a dynamic XFA form" }
func (DynamicXFA) Category() engine.Category { return engine.CategoryInteractive }
func (DynamicXFA) Severity() engine.Severity { return engine.SeverityError }
func (DynamicXFA) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (DynamicXFA) WCAG() []string            { return []string{"1.3.1"} }
func (DynamicXFA) Description() string {
	return "ISO 14289-1 §7.15 forbids dynamic XFA forms. A dynamic XFA form re-renders its layout at open time from the XFA template, bypassing the tagged page content so assistive technology has nothing to read. The form is dynamic when its XFA config packet declares <dynamicRender>required. Static XFA is permitted under PDF/UA-1; PDF/UA-2 forbids XFA entirely (UA-28-010)."
}

func (c DynamicXFA) Run(doc model.Document) []engine.Finding {
	xfa := doc.XFA()
	if !xfa.Present {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no XFA form -- nothing to inspect",
		}}
	}
	if xfa.DynamicRender == "required" {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "document uses a dynamic XFA form (XFA config declares <dynamicRender>required)",
			Hint:     "Remove the dynamic XFA form and rely on the tagged page content. Dynamic XFA re-renders at open time and cannot be read by assistive technology.",
		}}
	}
	return nil
}

func init() { engine.Register(DynamicXFA{}) }
