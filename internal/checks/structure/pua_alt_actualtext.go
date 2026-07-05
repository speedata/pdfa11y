package structure

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// containsPUA reports whether s contains a Unicode Private Use Area code
// point: U+E000-U+F8FF (BMP), U+F0000-U+FFFFD (plane 15) or
// U+100000-U+10FFFD (plane 16). PUA code points have no assigned meaning, so
// assistive technology cannot read them.
func containsPUA(s string) bool {
	for _, r := range s {
		switch {
		case r >= 0xE000 && r <= 0xF8FF,
			r >= 0xF0000 && r <= 0xFFFFD,
			r >= 0x100000 && r <= 0x10FFFD:
			return true
		}
	}
	return false
}

// PUAInAltActualText fails when an /Alt or /ActualText entry contains a
// Unicode Private Use Area code point. PDF/UA-2 §8.4.3-2/-3 forbid PUA in these
// alternate-text entries: their whole purpose is to give assistive technology
// readable text, which PUA code points are not.
//
// One finding per offending attribute. N/A when the document has no structure
// tree.
type PUAInAltActualText struct{}

func (PUAInAltActualText) ID() string                { return "UA-01-013" }
func (PUAInAltActualText) Title() string             { return "Alt and ActualText do not contain Private Use Area characters" }
func (PUAInAltActualText) Category() engine.Category { return engine.CategoryRealContent }
func (PUAInAltActualText) Severity() engine.Severity { return engine.SeverityError }
func (PUAInAltActualText) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (PUAInAltActualText) WCAG() []string            { return []string{"1.1.1"} }
func (PUAInAltActualText) Description() string {
	return "PDF/UA-2 §8.4.3 forbids Unicode Private Use Area (PUA) code points in /Alt and /ActualText. These entries exist to provide readable alternate text, and PUA code points carry no assigned meaning that assistive technology could voice."
}

func (c PUAInAltActualText) Run(doc model.Document) []engine.Finding {
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
	checked := 0
	c.walk(root, "/"+root.Type(), &checked, &findings)
	if checked == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no /Alt or /ActualText entries -- nothing to inspect",
		}}
	}
	return findings
}

func (c PUAInAltActualText) walk(elem model.StructElement, path string, checked *int, findings *[]engine.Finding) {
	for _, attr := range []string{"Alt", "ActualText"} {
		v := elem.Attr(attr)
		if v == "" {
			continue
		}
		*checked++
		if containsPUA(v) {
			*findings = append(*findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("/%s at %s contains a Unicode Private Use Area character", attr, path),
				Hint:     "Replace the Private Use Area code point(s) with readable Unicode text.",
				Location: &engine.Location{Page: elem.Page(), StructPath: path},
			})
		}
	}
	for _, ch := range elem.Children() {
		c.walk(ch, path+"/"+ch.Type(), checked, findings)
	}
}

func init() { engine.Register(PUAInAltActualText{}) }
