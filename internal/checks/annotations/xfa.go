package annotations

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// XFAForbidden fails when a PDF/UA-2 document carries XFA forms.
//
// ISO 14289-2 §8.10.1 closes with: "XFA forms (see ISO 32000-2,
// Annex K) shall not be present." PDF 2.0 deprecated XFA entirely;
// PDF/UA-2 forbids it outright. XFA forms describe their
// interactivity through a separate XML dialect that PDF/UA's
// tagged-structure model does not extend to.
//
// PDF/UA-2 only, gated via pdfua.DetectPart.
type XFAForbidden struct{}

func (XFAForbidden) ID() string                { return "UA-28-010" }
func (XFAForbidden) Title() string             { return "PDF/UA-2 documents do not contain XFA forms" }
func (XFAForbidden) Category() engine.Category { return engine.CategoryInteractive }
func (XFAForbidden) Severity() engine.Severity { return engine.SeverityError }
func (XFAForbidden) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (XFAForbidden) WCAG() []string            { return nil }
func (XFAForbidden) Description() string {
	return "ISO 14289-2 §8.10.1 forbids XFA forms in PDF/UA-2 (XFA was deprecated in PDF 2.0 / ISO 32000-2 Annex K). XFA forms convey their interactivity through a separate XML dialect that the PDF/UA tagged-structure model does not cover, so AT cannot reliably consume them. Forms in PDF/UA-2 must use AcroForm widget annotations only."
}

func (c XFAForbidden) Run(doc model.Document) []engine.Finding {
	part, found, err := pdfua.DetectPart(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata: " + err.Error(),
		}}
	}
	if !found || part != 2 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "check applies to PDF/UA-2 only; document does not declare pdfuaid:part = 2",
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
	afObj, ok := cat.Find("AcroForm")
	if !ok {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no /AcroForm -- nothing to inspect",
		}}
	}
	af, err := doc.DereferenceDict(afObj)
	if err != nil || af == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document /AcroForm does not resolve to a dictionary",
		}}
	}
	if _, hasXFA := af.Find("XFA"); !hasXFA {
		return nil
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "document /AcroForm carries an /XFA entry; XFA forms are forbidden in PDF/UA-2",
		Hint:     "Remove the /XFA entry from /AcroForm. Express the form's behaviour through the standard /AcroForm/Fields hierarchy (AcroForm) instead.",
	}}
}

func init() { engine.Register(XFAForbidden{}) }
