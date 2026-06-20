package metadata

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// PDFUAIdentifier verifies that the catalog points to an XMP metadata
// stream and that the stream declares a pdfuaid:part value of 1 or 2.
// ISO 14289 requires this identifier to be present so that consumers can
// recognise the document as PDF/UA-conforming.
//
// Limitation: this MVP implementation searches the raw XMP bytes with a
// regex. It handles both the element form (<pdfuaid:part>1</pdfuaid:part>)
// and the attribute form (pdfuaid:part="1") with the conventional
// "pdfuaid" namespace prefix. XMP files using a non-conventional prefix
// for the http://www.aiim.org/pdfua/ns/id/ namespace will yield a false
// negative; a proper XML/namespace-aware parser will replace this once we
// have more XMP-based checks.
type PDFUAIdentifier struct{}

func (PDFUAIdentifier) ID() string    { return "UA-06-003" }
func (PDFUAIdentifier) Title() string { return "XMP metadata declares PDF/UA identifier" }
func (PDFUAIdentifier) Description() string {
	return "ISO 14289 requires the XMP metadata stream to declare pdfuaid:part = 1 (PDF/UA-1) or 2 (PDF/UA-2). Conforming consumers rely on this identifier to recognise the document as PDF/UA-compliant."
}
func (PDFUAIdentifier) Category() engine.Category { return engine.CategoryMetadata }
func (PDFUAIdentifier) Severity() engine.Severity { return engine.SeverityError }
func (PDFUAIdentifier) Spec() engine.Spec         { return engine.SpecBoth }
func (PDFUAIdentifier) WCAG() []string            { return nil }

func (c PDFUAIdentifier) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}
	if _, found := catalog.Find("Metadata"); !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "document has no /Metadata stream — PDF/UA identifier cannot be declared",
			Hint:     "Embed an XMP metadata stream containing pdfuaid:part = 1 (or 2 for PDF/UA-2).",
		}}
	}
	_, found, err := pdfua.DetectPart(doc)
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
			Message:  "XMP metadata is present but contains no pdfuaid:part identifier",
			Hint:     "Add <pdfuaid:part>1</pdfuaid:part> (or 2 for PDF/UA-2) inside an rdf:Description with the namespace http://www.aiim.org/pdfua/ns/id/.",
		}}
	}
	return nil
}

func init() { engine.Register(PDFUAIdentifier{}) }
