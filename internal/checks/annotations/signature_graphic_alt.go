package annotations

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// SignatureGraphicAlt fails when a graphic that is part of a signature's
// appearance carries no alternative text. ISO 14289-2 §8.10.3.5 (veraPDF
// UA2:8.10.3.5-1): if a portion of the appearance of a signature is
// represented by a graphic, alternative text shall be provided for that
// graphic, so assistive technology can describe what the signature shows.
//
// A graphic is a Figure structure element inside the Form structure element
// that encloses a signature-field widget. One finding per such Figure without
// /Alt. N/A when no signature graphic is present. PDF/UA-2 only.
type SignatureGraphicAlt struct{}

func (SignatureGraphicAlt) ID() string { return "UA-28-036" }
func (SignatureGraphicAlt) Title() string {
	return "Signature appearance graphics have alternative text"
}
func (SignatureGraphicAlt) Category() engine.Category { return engine.CategoryInteractive }
func (SignatureGraphicAlt) Severity() engine.Severity { return engine.SeverityError }
func (SignatureGraphicAlt) WCAG() []string            { return []string{"1.1.1"} }
func (SignatureGraphicAlt) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (SignatureGraphicAlt) Description() string {
	return "ISO 14289-2 §8.10.3.5 requires alternative text for any graphic that represents a portion of a signature's appearance. A Figure inside the Form structure element of a signature field with no /Alt leaves assistive technology unable to describe what the signature graphic shows."
}

func (c SignatureGraphicAlt) Run(doc model.Document) []engine.Finding {
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
	c.walk(root, false, &findings, &seen)
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no signature appearance graphics -- nothing to inspect",
		}}
	}
	return findings
}

// walk descends the structure tree. inSignature becomes true once an ancestor
// element encloses a signature widget; from there every Figure descendant is a
// signature appearance graphic that must carry /Alt.
func (c SignatureGraphicAlt) walk(elem model.StructElement, inSignature bool, out *[]engine.Finding, seen *int) {
	if !inSignature && elem.EnclosesSignatureWidget() {
		inSignature = true
	}
	if inSignature && elem.Type() == "Figure" {
		*seen++
		if elem.Attr("Alt") == "" {
			*out = append(*out, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "signature appearance graphic (Figure) has no /Alt alternative text",
				Hint:     "Add an /Alt entry to the Figure describing the graphic portion of the signature's appearance.",
				Location: &engine.Location{Page: elem.Page()},
			})
		}
	}
	for _, child := range elem.Children() {
		c.walk(child, inSignature, out, seen)
	}
}

func init() { engine.Register(SignatureGraphicAlt{}) }
