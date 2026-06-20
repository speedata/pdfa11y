package metadata

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// PDFUARevision fails when the XMP metadata stream does not declare
// pdfuaid:rev. ISO 14289-2 §5 marks both pdfuaid:part and pdfuaid:rev
// as Required: rev is the four-digit year of publication or
// revision of the PDF/UA part the document claims to conform to.
//
// Companion to UA-06-003 (pdfuaid:part). UA-06-003 already catches
// the case where the XMP stream itself is missing or contains no
// pdfuaid identifier at all, so this check fires only when part is
// present but rev is not.
//
// Spec gating: ISO 14289-2 §5 spells out both properties as
// Required. The ISO 14289-1 schema also defines pdfuaid:rev, so
// the check applies to both UA-1 and UA-2 files.
type PDFUARevision struct{}

func (PDFUARevision) ID() string    { return "UA-06-006" }
func (PDFUARevision) Title() string { return "XMP metadata declares pdfuaid:rev" }
func (PDFUARevision) Description() string {
	return "ISO 14289-2 §5 specifies the pdfuaid:rev property as Required: a four-digit year identifying the publication or revision year of the PDF/UA part the file conforms to. Together with pdfuaid:part (UA-06-003) it forms the PDF/UA identification schema. Without rev the identifier set is incomplete; consumers cannot tell which revision of the standard was targeted."
}
func (PDFUARevision) Category() engine.Category { return engine.CategoryMetadata }
func (PDFUARevision) Severity() engine.Severity { return engine.SeverityError }
func (PDFUARevision) Spec() engine.Spec         { return engine.SpecBoth }
func (PDFUARevision) WCAG() []string            { return nil }

func (c PDFUARevision) Run(doc model.Document) []engine.Finding {
	// Guard: if pdfuaid:part is absent the document is not declaring
	// PDF/UA conformance at all, and UA-06-003 already reports that.
	// Asking for rev there would just double up the same finding.
	if _, found, err := pdfua.DetectPart(doc); err != nil || !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "pdfuaid:part is absent -- rev is meaningful only alongside a declared part",
		}}
	}
	_, found, err := pdfua.DetectRev(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata stream: " + err.Error(),
		}}
	}
	if !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "XMP metadata declares pdfuaid:part but no pdfuaid:rev",
			Hint:     "Add <pdfuaid:rev>2024</pdfuaid:rev> (or the four-digit year of the PDF/UA-part revision you target) inside the same rdf:Description that carries pdfuaid:part.",
		}}
	}
	return nil
}

func init() { engine.Register(PDFUARevision{}) }
