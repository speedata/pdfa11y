package graphics

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// FormulaMathMLSubtype fails when a Formula carries an Associated
// File declared as a MathML supplement (AFRelationship = Supplement)
// but the embedded-file stream's /Subtype is not the BPG-mandated
// application/mathml+xml. ISO 14289-2 §8.2.5.29.1 and the PDF
// Association BPG "Math in PDF" both require that exact MIME type
// for MathML-as-AF on a Formula.
//
// PDF/UA-2 only: gated via pdfua.DetectPart so engine.All()-based
// callers see N/A on PDF/UA-1 documents instead of false positives.
// MH-17-003 is the symmetric check (Subtype right, relationship
// wrong); together they catch the two ways a producer can
// half-declare a MathML AF.
type FormulaMathMLSubtype struct{}

func (FormulaMathMLSubtype) ID() string                { return "MH-17-002" }
func (FormulaMathMLSubtype) Title() string             { return "MathML associated file declares Subtype = application/mathml+xml" }
func (FormulaMathMLSubtype) Category() engine.Category { return engine.CategoryGraphics }
func (FormulaMathMLSubtype) Severity() engine.Severity { return engine.SeverityError }
func (FormulaMathMLSubtype) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (FormulaMathMLSubtype) WCAG() []string            { return []string{"1.1.1"} }
func (FormulaMathMLSubtype) Description() string {
	return "PDF/UA-2 §8.2.5.29.1 (and PDF Association BPG \"Math in PDF\") require every MathML Associated File attached to a Formula to declare /Subtype application/mathml+xml on its embedded-file stream (PDF-encoded as /application#2Fmathml+xml). An AF with /AFRelationship /Supplement but a different Subtype tells AT software it is not a MathML supplement, defeating the math accessibility surface."
}

func (c FormulaMathMLSubtype) Run(doc model.Document) []engine.Finding {
	if !isPDFUA2(doc) {
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
	supplementCount := 0
	walkFormulas(root, "/"+root.Type(), func(formula model.StructElement, path string) {
		for _, af := range formula.AssociatedFiles() {
			if af.Relationship != "Supplement" {
				continue
			}
			supplementCount++
			if af.Subtype == mathMLSubtype {
				continue
			}
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message: fmt.Sprintf(
					"Formula AF %q declares /AFRelationship /Supplement but /Subtype %q (expected %q)",
					displayFilename(af.Filename), displaySubtype(af.Subtype), mathMLSubtype),
				Hint:     "Set the embedded-file stream's /Subtype to /application#2Fmathml+xml. PDF encodes the MIME slash as the hex escape '#2F'.",
				Location: &engine.Location{Page: formula.Page(), StructPath: path},
			})
		}
	})
	if supplementCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Formula with an AFRelationship=/Supplement AF -- nothing to inspect",
		}}
	}
	return findings
}

// FormulaMathMLRelationship is the symmetric counterpart to
// FormulaMathMLSubtype: it fails when an AF whose stream Subtype is
// application/mathml+xml is attached to a Formula with the wrong
// /AFRelationship. The BPG fixes the pair: a MathML AF must be both
// application/mathml+xml AND /Supplement.
type FormulaMathMLRelationship struct{}

func (FormulaMathMLRelationship) ID() string                { return "MH-17-003" }
func (FormulaMathMLRelationship) Title() string             { return "MathML associated file declares AFRelationship = /Supplement" }
func (FormulaMathMLRelationship) Category() engine.Category { return engine.CategoryGraphics }
func (FormulaMathMLRelationship) Severity() engine.Severity { return engine.SeverityError }
func (FormulaMathMLRelationship) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (FormulaMathMLRelationship) WCAG() []string            { return []string{"1.1.1"} }
func (FormulaMathMLRelationship) Description() string {
	return "PDF/UA-2 §8.2.5.29.1 (and PDF Association BPG \"Math in PDF\") require every MathML Associated File attached to a Formula to declare /AFRelationship /Supplement on the filespec. An embedded stream declared as application/mathml+xml but reached through a non-Supplement AF entry is not the BPG MathML pattern; AT software will not pick it up as the canonical math representation."
}

func (c FormulaMathMLRelationship) Run(doc model.Document) []engine.Finding {
	if !isPDFUA2(doc) {
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
	mathmlCount := 0
	walkFormulas(root, "/"+root.Type(), func(formula model.StructElement, path string) {
		for _, af := range formula.AssociatedFiles() {
			if af.Subtype != mathMLSubtype {
				continue
			}
			mathmlCount++
			if af.Relationship == "Supplement" {
				continue
			}
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message: fmt.Sprintf(
					"Formula AF %q declares /Subtype %s but /AFRelationship %q (expected %q)",
					displayFilename(af.Filename), mathMLSubtype, displayRelationship(af.Relationship), "Supplement"),
				Hint:     "Set the filespec's /AFRelationship to /Supplement. The MathML AF on a Formula is the canonical supplement per BPG §\"Use of Associated files\".",
				Location: &engine.Location{Page: formula.Page(), StructPath: path},
			})
		}
	})
	if mathmlCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Formula with a MathML AF -- nothing to inspect",
		}}
	}
	return findings
}

// isPDFUA2 reports whether the document's XMP declares pdfuaid:part = 2.
// Math L2/L3 checks all gate on this so engine.All()-based callers
// (tests, the realworld driver) see N/A on PDF/UA-1 documents
// instead of bogus errors.
func isPDFUA2(doc model.Document) bool {
	part, found, err := pdfua.DetectPart(doc)
	return err == nil && found && part == 2
}

// walkFormulas invokes fn for every Formula structure element under
// elem, threading the slash-separated structure-tag path through.
// Shared by the MH-17 family so each check does not re-implement the
// recursion.
func walkFormulas(elem model.StructElement, path string, fn func(model.StructElement, string)) {
	if elem.Type() == "Formula" {
		fn(elem, path)
	}
	for _, child := range elem.Children() {
		walkFormulas(child, path+"/"+child.Type(), fn)
	}
}

func displayFilename(f string) string {
	if f == "" {
		return "(unnamed)"
	}
	return f
}

func displaySubtype(s string) string {
	if s == "" {
		return "(absent)"
	}
	return s
}

func displayRelationship(r string) string {
	if r == "" {
		return "(absent)"
	}
	return r
}

func init() {
	engine.Register(FormulaMathMLSubtype{})
	engine.Register(FormulaMathMLRelationship{})
}
