package structure

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// CaptionPosition fails when a Caption structure element is neither the
// first nor the last child of its parent. PDF/UA-2 §8.2.5.27 requires a
// Caption -- whether on a Table, a List or a Figure -- to sit at one end of
// its parent's children so assistive technology can announce it as an
// introduction or a summary rather than losing it mid-content.
//
// One finding per misplaced Caption. N/A when the document has no Caption
// elements at all.
type CaptionPosition struct{}

func (CaptionPosition) ID() string                { return "UA-01-011" }
func (CaptionPosition) Title() string             { return "Caption is the first or last child of its parent" }
func (CaptionPosition) Category() engine.Category { return engine.CategoryStructure }
func (CaptionPosition) Severity() engine.Severity { return engine.SeverityError }
func (CaptionPosition) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (CaptionPosition) WCAG() []string            { return []string{"1.3.1"} }
func (CaptionPosition) Description() string {
	return "PDF/UA-2 §8.2.5.27 requires a Caption structure element, when present, to be the first or the last child of its parent (e.g. a Table or List). A Caption buried between other children is read out of place and loses its role as the caption for the surrounding structure."
}

func (c CaptionPosition) Run(doc model.Document) []engine.Finding {
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
	seen := c.walk(root, "/"+root.Type(), &findings)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Caption elements -- nothing to inspect",
		}}
	}
	return findings
}

// walk inspects elem's children for misplaced Captions and recurses,
// returning the number of Caption children encountered.
func (c CaptionPosition) walk(elem model.StructElement, path string, findings *[]engine.Finding) int {
	kids := elem.Children()
	seen := 0
	for i, k := range kids {
		if k.Type() != "Caption" {
			continue
		}
		seen++
		if i != 0 && i != len(kids)-1 {
			*findings = append(*findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("Caption is child %d of %d under %s -- it must be the first or last child", i+1, len(kids), elem.Type()),
				Hint:     "Move the Caption structure element to the beginning or the end of its parent's children.",
				Location: &engine.Location{Page: k.Page(), StructPath: path + "/Caption"},
			})
		}
	}
	for _, k := range kids {
		seen += c.walk(k, path+"/"+k.Type(), findings)
	}
	return seen
}

func init() { engine.Register(CaptionPosition{}) }
