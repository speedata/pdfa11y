package headings

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// StyleMix fails when a document uses both the unnumbered /H tag and
// numbered /H1.../H6 tags. PDF/UA-1 §7.4.4 forces a single
// heading-style choice per document ("documents shall be either
// strongly or weakly structured, but not both"): either /H throughout
// (a depth-
// less outline whose nesting is implied by the structure tree) or
// /H<n> throughout (an explicit numeric outline). Mixing the two
// leaves assistive technology unable to interpret either style
// consistently -- the unnumbered /H has no place in the H<n>
// numbering, and the H<n> ladder cannot include an unlevelled item.
//
// Spec gating: PDF/UA-1 only. The strongly-vs-weakly-structured
// distinction (ISO 14289-1 §7.4.4, Matterhorn 14-007) exists only in
// UA-1. ISO 14289-2 §8.2.5.12 forbids the /H tag outright and has no
// "mixing" concept -- that case is already covered by UA-14-007
// (NoUntypedH, PDF/UA-2 only).
type StyleMix struct{}

func (StyleMix) ID() string                { return "UA-14-006" }
func (StyleMix) Title() string             { return "Heading style is consistent (H or H<n>, not both)" }
func (StyleMix) Category() engine.Category { return engine.CategoryHeadings }
func (StyleMix) Severity() engine.Severity { return engine.SeverityError }
func (StyleMix) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (StyleMix) WCAG() []string            { return []string{"1.3.1", "2.4.6"} }
func (StyleMix) Description() string {
	return "PDF/UA-1 §7.4.4 requires a single heading style per document (strongly or weakly structured, not both). /H is the unnumbered form whose level is implied by structure nesting; /H1.../H6 is the explicit-level form. Mixing them within one document confuses assistive technology -- the unlevelled /H cannot be placed on the H<n> outline. PDF/UA-2 forbids /H outright, so this applies to PDF/UA-1 only."
}

func (c StyleMix) Run(doc model.Document) []engine.Finding {
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
	hasH, hasHn := false, false
	var firstH, firstHn occurrence
	collectStyles(root, "/"+root.Type(), &hasH, &hasHn, &firstH, &firstHn)
	if !hasH && !hasHn {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no heading elements -- nothing to inspect",
		}}
	}
	if hasH && hasHn {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "document mixes /H (unnumbered) and /H<n> (numbered) heading tags; pick one style",
			Hint:     "Convert every /H to the appropriate /H<n>, or convert every /H<n> to /H. The /H style requires the structure tree's nesting to encode the heading level instead.",
			Location: &engine.Location{Page: firstH.page, StructPath: firstH.path},
		}, {
			CheckID:  c.ID(),
			Severity: engine.SeverityInfo,
			Message:  "first /H<n> heading found here (for reference)",
			Location: &engine.Location{Page: firstHn.page, StructPath: firstHn.path},
		}}
	}
	return nil
}

func collectStyles(elem model.StructElement, path string, hasH, hasHn *bool, firstH, firstHn *occurrence) {
	t := elem.Type()
	if t == "H" && !*hasH {
		*hasH = true
		*firstH = occurrence{path: path, page: elem.Page()}
	}
	if level := headingLevel(t); level > 0 && !*hasHn {
		*hasHn = true
		*firstHn = occurrence{level: level, path: path, page: elem.Page()}
	}
	for _, child := range elem.Children() {
		collectStyles(child, path+"/"+child.Type(), hasH, hasHn, firstH, firstHn)
	}
}

func init() { engine.Register(StyleMix{}) }
