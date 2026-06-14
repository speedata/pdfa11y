package structure

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// DocumentNamespace fails when a PDF/UA-2 document's root
// `Document` structure element is not declared in the PDF 2.0
// standard structure namespace.
//
// ISO 14289-2 §8.2.5.2: "The namespace for that element shall be
// specified as the PDF 2.0 namespace, in accordance with
// ISO 32000-2:2020, 14.8.6." NOTE 1 of the same clause spells out
// the consequence: "PDF/UA-2-conforming files cannot solely
// conform to PDF 1.7 because this subclause requires that the
// Document structure element is in the PDF 2.0 namespace."
//
// The PDF 2.0 namespace URI is http://iso.org/pdf2/ssn
// (ISO/TS 32005:2023). A Document in the PDF 1.7 namespace
// (http://iso.org/pdf/ssn) or with no namespace at all signals a
// document that was tagged for PDF/UA-1 and not migrated to UA-2.
//
// PDF/UA-2 only, gated via pdfua.DetectPart.
type DocumentNamespace struct{}

const pdf2Namespace = "http://iso.org/pdf2/ssn"

func (DocumentNamespace) ID() string                { return "MH-01-008" }
func (DocumentNamespace) Title() string             { return "PDF/UA-2 Document structure element is in the PDF 2.0 namespace" }
func (DocumentNamespace) Category() engine.Category { return engine.CategoryStructure }
func (DocumentNamespace) Severity() engine.Severity { return engine.SeverityError }
func (DocumentNamespace) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (DocumentNamespace) WCAG() []string            { return []string{"1.3.1"} }
func (DocumentNamespace) Description() string {
	return "ISO 14289-2 §8.2.5.2 requires the root Document structure element of a PDF/UA-2 file to be declared in the PDF 2.0 standard structure namespace (http://iso.org/pdf2/ssn, registered by ISO/TS 32005). A Document declared in the PDF 1.7 namespace, or with no /NS at all, indicates a structure tree authored against PDF/UA-1 conventions; the spec excludes such files from PDF/UA-2 conformance."
}

func (c DocumentNamespace) Run(doc model.Document) []engine.Finding {
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
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}
	if root.Type() != "Document" {
		// StructTreeRoot's only child should be Document, but if
		// it is not, MH-31-008 (custom-types) or the structural
		// containment rules will fire elsewhere. We do not
		// duplicate that diagnosis.
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "structure-tree root is not a Document element -- skipping namespace check",
		}}
	}
	ns := root.Namespace()
	if ns == pdf2Namespace {
		return nil
	}
	msg := "Document structure element declares no namespace; expected " + pdf2Namespace
	if ns != "" {
		msg = "Document structure element is in namespace " + quote(ns) + ", expected " + quote(pdf2Namespace)
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  msg,
		Hint:     "Attach a /NS reference on the Document element pointing to a Namespace dictionary whose /NS string is " + pdf2Namespace + ". Add the same dictionary to /StructTreeRoot/Namespaces.",
	}}
}

func quote(s string) string {
	return `"` + s + `"`
}

func init() { engine.Register(DocumentNamespace{}) }
