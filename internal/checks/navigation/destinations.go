package navigation

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// StructureDestinations enforces ISO 14289-2 §8.8: every destination whose
// target lies within the document shall be a structure destination -- its
// target a structure element, not a page. Structure destinations survive
// re-pagination and let assistive technology land on the tagged content a
// link or bookmark points at, rather than raw page coordinates.
//
// It flags each intra-document destination whose target is positively a
// page: the /Dest of an outline item or Link annotation, a
// destination-array /OpenAction (veraPDF UA2:8.8-1), or the /D of a /GoTo
// action reached from any of those carriers (veraPDF UA2:8.8-2). A carrier
// that carries an /SD structure destination, or whose target is a
// structure element (or cannot be classified), is not flagged.
//
// PDF/UA-2 only: structure destinations are a PDF 2.0 feature and ISO
// 14289-1 imposes no such requirement.
//
// One finding per offending destination. N/A when the document carries no
// intra-document destinations at all.
type StructureDestinations struct{}

func (StructureDestinations) ID() string                { return "UA-27-003" }
func (StructureDestinations) Title() string             { return "Intra-document destinations are structure destinations" }
func (StructureDestinations) Category() engine.Category { return engine.CategoryNavigation }
func (StructureDestinations) Severity() engine.Severity { return engine.SeverityError }
func (StructureDestinations) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (StructureDestinations) WCAG() []string            { return []string{"1.3.1"} }
func (StructureDestinations) Description() string {
	return "ISO 14289-2 §8.8 requires every destination whose target lies within the document to be a structure destination -- one that targets a structure element rather than a page. Structure destinations survive re-pagination and let assistive technology land on the tagged content a bookmark or link points at. pdfa11y flags outline items, Link annotations, the /OpenAction, and /GoTo actions whose destination targets a page."
}

func (c StructureDestinations) Run(doc model.Document) []engine.Finding {
	dests := doc.IntraDocDestinations()
	if len(dests) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document carries no intra-document destinations -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, d := range dests {
		if !d.TargetsPage {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  d.Source + " uses a page destination, not a structure destination",
			Hint:     "Point the destination at the structure element for the target content (an /SD structure destination, or a destination array whose first element is a structure element) instead of a page.",
		})
	}
	return findings
}

func init() { engine.Register(StructureDestinations{}) }
