package notes

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// NoteNotInUA2 fails when a PDF/UA-2 document contains a Note
// structure element in the default PDF namespace.
//
// ISO 14289-2 §8.2.5.14 introduces FENote as the replacement for
// the legacy Note structure type: "FENote effectively replaces
// the Note structure type [...]. The Note standard structure
// type shall not be present in conforming files."
//
// Note in a non-default namespace that is role-mapped to an
// allowed PDF 1.7 / PDF 2.0 type remains permitted (§8.2.5.14
// Note 1), so the check ignores 'Note' elements that declare a
// non-PDF namespace via /NS.
//
// PDF/UA-2 only: gated via pdfua.DetectPart.
type NoteNotInUA2 struct{}

func (NoteNotInUA2) ID() string                { return "MH-14-009" }
func (NoteNotInUA2) Title() string             { return "PDF/UA-2 documents do not use the Note structure type" }
func (NoteNotInUA2) Category() engine.Category { return engine.CategoryNotes }
func (NoteNotInUA2) Severity() engine.Severity { return engine.SeverityError }
func (NoteNotInUA2) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (NoteNotInUA2) WCAG() []string            { return nil }
func (NoteNotInUA2) Description() string {
	return "ISO 14289-2 §8.2.5.14 introduces FENote (Footnote/Endnote) as the replacement for the legacy Note structure type. In conforming PDF/UA-2 files the Note standard structure type shall not be present. Note in a non-default namespace that is role-mapped to an allowed PDF 1.7 / PDF 2.0 type remains permitted (Note 1 of the same clause); this check inspects only elements in the default PDF structure namespace."
}

func (c NoteNotInUA2) Run(doc model.Document) []engine.Finding {
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

	var findings []engine.Finding
	c.walk(root, "/"+root.Type(), &findings)
	return findings
}

func (c NoteNotInUA2) walk(elem model.StructElement, path string, out *[]engine.Finding) {
	if elem.Type() == "Note" && inStandardPDFNamespace(elem) {
		*out = append(*out, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("Note structure element at %s is not permitted in PDF/UA-2", path),
			Hint:     "Replace the Note structure element with FENote (set /NoteType to Footnote or Endnote as appropriate). §8.2.5.14 specifies the replacement.",
			Location: &engine.Location{Page: elem.Page(), StructPath: path},
		})
	}
	for _, child := range elem.Children() {
		c.walk(child, path+"/"+child.Type(), out)
	}
}

// inStandardPDFNamespace mirrors the predicate used by MH-31-008:
// elements with no namespace or with one of the registered PDF
// standard structure namespace URIs (ISO 32000-2 §14.8.6.3 /
// ISO/TS 32005) live in the default PDF type system.
func inStandardPDFNamespace(elem model.StructElement) bool {
	ns := elem.Namespace()
	if ns == "" {
		return true
	}
	switch ns {
	case "http://iso.org/pdf/ssn", "http://iso.org/pdf2/ssn":
		return true
	}
	return false
}

func init() { engine.Register(NoteNotInUA2{}) }
