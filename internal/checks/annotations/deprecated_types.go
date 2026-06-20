package annotations

import (
	"fmt"
	"sort"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// DeprecatedAnnotTypes fails when a PDF/UA-2 document carries an
// annotation whose subtype has been deprecated in PDF 2.0 and
// explicitly forbidden by ISO 14289-2. The relevant subtypes are
// listed below.
//
// Sources:
//   - §8.9.2.4.11 Sound, movie: "Sound and movie annotations are
//     deprecated in PDF 2.0 and therefore, in accordance with
//     §8.9.1, shall not be present."
//   - §8.9.2.4.15 Trap network: "Trap network annotations are
//     deprecated in PDF 2.0 and shall not be used in files
//     conforming to this document."
//
// One finding per unique forbidden subtype, with an occurrence
// count, so a document with many forbidden annotations does not
// flood the report.
//
// PDF/UA-2 only, gated via pdfua.DetectPart.
type DeprecatedAnnotTypes struct{}

func (DeprecatedAnnotTypes) ID() string                { return "UA-28-009" }
func (DeprecatedAnnotTypes) Title() string             { return "PDF/UA-2 documents do not contain deprecated annotation types" }
func (DeprecatedAnnotTypes) Category() engine.Category { return engine.CategoryInteractive }
func (DeprecatedAnnotTypes) Severity() engine.Severity { return engine.SeverityError }
func (DeprecatedAnnotTypes) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (DeprecatedAnnotTypes) WCAG() []string            { return nil }
func (DeprecatedAnnotTypes) Description() string {
	return "ISO 14289-2 forbids annotation types that PDF 2.0 deprecated: Sound and Movie (§8.9.2.4.11) and TrapNet (§8.9.2.4.15). Their functions were replaced by multimedia annotations (Screen + RichMedia) and by output-intent dictionaries respectively; carrying the deprecated types in a PDF/UA-2 document signals an authoring pipeline that has not been updated for PDF 2.0."
}

// forbiddenSubtypes lists annotation /Subtype values that PDF/UA-2
// forbids. ISO 32000-2 §12.5.6 deprecates each in turn; ISO 14289-2
// turns the deprecation into a hard prohibition.
var forbiddenSubtypes = map[string]string{
	"Sound":   "deprecated in PDF 2.0 (§8.9.2.4.11)",
	"Movie":   "deprecated in PDF 2.0 (§8.9.2.4.11)",
	"TrapNet": "deprecated in PDF 2.0 (§8.9.2.4.15)",
}

func (c DeprecatedAnnotTypes) Run(doc model.Document) []engine.Finding {
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
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	if len(annots) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no annotations -- nothing to inspect",
		}}
	}

	// Group findings: one per offending subtype with a count and
	// the first page on which it appeared.
	type occ struct {
		count int
		page  int
	}
	hits := map[string]*occ{}
	for _, a := range annots {
		reason, bad := forbiddenSubtypes[a.Subtype]
		if !bad {
			continue
		}
		_ = reason
		if o, ok := hits[a.Subtype]; ok {
			o.count++
		} else {
			hits[a.Subtype] = &occ{count: 1, page: a.Page}
		}
	}
	if len(hits) == 0 {
		return nil
	}

	subs := make([]string, 0, len(hits))
	for s := range hits {
		subs = append(subs, s)
	}
	sort.Strings(subs)
	var findings []engine.Finding
	for _, sub := range subs {
		o := hits[sub]
		msg := fmt.Sprintf(
			"%s annotation is %s -- %d occurrence(s); PDF/UA-2 forbids it",
			sub, forbiddenSubtypes[sub], o.count)
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  msg,
			Hint:     hintFor(sub),
			Location: &engine.Location{Page: o.page},
		})
	}
	return findings
}

func hintFor(subtype string) string {
	switch subtype {
	case "Sound", "Movie":
		return "Replace with a Screen or RichMedia annotation per ISO 32000-2 §12.5.6.18 / §13."
	case "TrapNet":
		return "Trap networks are removed in PDF 2.0; rely on output intents (ISO 32000-2 §14.11)."
	}
	return ""
}

func init() { engine.Register(DeprecatedAnnotTypes{}) }
