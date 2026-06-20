package headings

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// NoUntypedH fails when a PDF/UA-2 document uses the untyped `H`
// structure type. ISO 14289-2 §8.2.5.12 is explicit: "Conforming
// files shall use the explicitly numbered heading structure types
// (H1-Hn) and shall not use the H structure type." Reason given in
// the same clause: H "requires processors to track section depth,
// which adds an unnecessary burden on processors and can cause
// ambiguity."
//
// Companion to UA-14-006, which catches the *mixing* of H and Hn
// in the same document (a PDF/UA-1 concern). UA-14-007 enforces
// the hard ban on H under UA-2 regardless of whether the document
// also uses Hn elements.
//
// PDF/UA-2 only, gated via pdfua.DetectPart. H elements in
// non-default namespaces that are role-mapped to permitted types
// remain allowed (the same namespace logic the other audit-derived
// checks use).
type NoUntypedH struct{}

func (NoUntypedH) ID() string                { return "UA-14-007" }
func (NoUntypedH) Title() string             { return "PDF/UA-2 documents use explicit H1-Hn, not the untyped H" }
func (NoUntypedH) Category() engine.Category { return engine.CategoryHeadings }
func (NoUntypedH) Severity() engine.Severity { return engine.SeverityError }
func (NoUntypedH) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (NoUntypedH) WCAG() []string            { return []string{"1.3.1", "2.4.6"} }
func (NoUntypedH) Description() string {
	return "ISO 14289-2 §8.2.5.12 requires conforming files to use the explicitly numbered heading types (H1-Hn) and forbids the untyped H structure type. H requires processors to track section depth from context, which is ambiguous and unnecessarily burdens AT. Use H1, H2, ... at the level appropriate to the document's heading hierarchy."
}

func (c NoUntypedH) Run(doc model.Document) []engine.Finding {
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

	// Collect all H occurrences in the default PDF namespace.
	// Report at most a few locations (the first three) and aggregate
	// the rest into a single trailing finding so a heading-heavy
	// document does not flood the report.
	var hits []hOccurrence
	walkH(root, "/"+root.Type(), &hits)
	if len(hits) == 0 {
		return nil
	}

	// Stable order: by path so two runs against the same file
	// produce identical reports.
	sort.Slice(hits, func(i, j int) bool { return hits[i].path < hits[j].path })

	var findings []engine.Finding
	const maxPerFile = 3
	for i, h := range hits {
		if i >= maxPerFile {
			break
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "untyped H structure element is not permitted in PDF/UA-2",
			Hint:     "Replace H with H1, H2, ... at the level matching its position in the heading hierarchy.",
			Location: &engine.Location{Page: h.page, StructPath: h.path},
		})
	}
	if len(hits) > maxPerFile {
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("%d further H occurrences elided", len(hits)-maxPerFile),
		})
	}
	return findings
}

// hOccurrence is one location an untyped H element was found at,
// used as the carrier between walkH and the check body.
type hOccurrence struct {
	path string
	page int
}

func walkH(elem model.StructElement, path string, hits *[]hOccurrence) {
	if elem.Type() == "H" && inStandardPDFNamespace(elem) {
		*hits = append(*hits, hOccurrence{path: path, page: elem.Page()})
	}
	for _, child := range elem.Children() {
		walkH(child, path+"/"+child.Type(), hits)
	}
}

// inStandardPDFNamespace is the same namespace predicate used by
// UA-14-009 and UA-31-008: the element lives in the default PDF
// type system (no namespace, or one of the two ISO 32000-2
// registered standard structure namespace URIs).
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

func init() { engine.Register(NoUntypedH{}) }
