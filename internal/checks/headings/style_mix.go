package headings

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// StyleMix fails when a document uses both the unnumbered /H tag and
// numbered /H1.../H6 tags. PDF/UA-1 §7.4.2 forces a single
// heading-style choice per document: either /H throughout (a depth-
// less outline whose nesting is implied by the structure tree) or
// /H<n> throughout (an explicit numeric outline). Mixing the two
// leaves assistive technology unable to interpret either style
// consistently -- the unnumbered /H has no place in the H<n>
// numbering, and the H<n> ladder cannot include an unlevelled item.
type StyleMix struct{}

func (StyleMix) ID() string                { return "UA-14-006" }
func (StyleMix) Title() string             { return "Heading style is consistent (H or H<n>, not both)" }
func (StyleMix) Category() engine.Category { return engine.CategoryHeadings }
func (StyleMix) Severity() engine.Severity { return engine.SeverityError }
func (StyleMix) Spec() engine.Spec         { return engine.SpecBoth }
func (StyleMix) WCAG() []string            { return []string{"1.3.1", "2.4.6"} }
func (StyleMix) Description() string {
	return "PDF/UA-1 §7.4.2 requires a single heading style per document. /H is the unnumbered form whose level is implied by structure nesting; /H1.../H6 is the explicit-level form. Mixing them within one document confuses assistive technology -- the unlevelled /H cannot be placed on the H<n> outline."
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
