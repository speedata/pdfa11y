package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// LinkDistinctTargets fails when a single Link or Reference structure element
// encloses link annotations that target different locations. ISO 14289-2
// §8.2.5.20 (veraPDF UA2:8.2.5.20-2): link annotations targeting different
// locations shall be in separate Link or Reference structure elements. When one
// tagged link groups several annotations that jump to different destinations,
// assistive technology announces one link but activating it is ambiguous.
//
// The target of each enclosed link is compared by its structure destination
// (/SD, preferred), explicit destination, or URI. One finding per offending
// element. N/A when the document has no Link or Reference structure elements.
// PDF/UA-2 only.
type LinkDistinctTargets struct{}

func (LinkDistinctTargets) ID() string { return "UA-28-030" }
func (LinkDistinctTargets) Title() string {
	return "Links to different locations are in separate Link/Reference elements"
}
func (LinkDistinctTargets) Category() engine.Category { return engine.CategoryInteractive }
func (LinkDistinctTargets) Severity() engine.Severity { return engine.SeverityError }
func (LinkDistinctTargets) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (LinkDistinctTargets) WCAG() []string            { return []string{"1.3.1", "2.4.4"} }
func (LinkDistinctTargets) Description() string {
	return "ISO 14289-2 §8.2.5.20 requires link annotations that target different locations to sit in separate Link or Reference structure elements. A single tagged link enclosing annotations that jump to different destinations is ambiguous: assistive technology presents one link but activation is unpredictable. Targets are compared by structure destination, explicit destination, or URI."
}

// linkTypes is the set of standard structure types the rule governs.
var linkTypes = map[string]bool{"Link": true, "Reference": true}

func (c LinkDistinctTargets) Run(doc model.Document) []engine.Finding {
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
	seen := 0
	c.walk(root, &findings, &seen)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no Link or Reference structure elements -- nothing to inspect",
		}}
	}
	return findings
}

func (c LinkDistinctTargets) walk(elem model.StructElement, out *[]engine.Finding, seen *int) {
	if linkTypes[elem.Type()] {
		*seen++
		distinct := map[string]bool{}
		for _, key := range elem.EnclosedLinkTargets() {
			distinct[key] = true
		}
		if len(distinct) > 1 {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("%s structure element on page %d encloses link annotations targeting %d different locations", elem.Type(), elem.Page(), len(distinct)),
				Hint:     "Split the link annotations into separate Link (or Reference) structure elements, one per distinct target.",
				Location: &engine.Location{Page: elem.Page()},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, out, seen)
	}
}

func init() { engine.Register(LinkDistinctTargets{}) }
