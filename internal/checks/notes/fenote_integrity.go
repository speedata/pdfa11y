package notes

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// validNoteTypes is the set of permitted /NoteType values on a FENote
// (ISO 32000-2, PDF/UA-2 §8.2.5.14-4).
var validNoteTypes = map[string]bool{"Footnote": true, "Endnote": true, "None": true}

// FENoteIntegrity checks the footnote/endnote (FENote) structure elements
// introduced by PDF/UA-2 §8.2.5.14:
//
//   - the /Ref link between a citation and its FENote must be bidirectional
//     (§8.2.5.14-2/-3): every element that references a FENote must be
//     referenced back by it, and a FENote must not reference an element that
//     does not reference the FENote; and
//   - a FENote's /NoteType, when present, must be Footnote, Endnote or None
//     (§8.2.5.14-4).
//
// One finding per defect. N/A when the document has no FENote elements.
type FENoteIntegrity struct{}

func (FENoteIntegrity) ID() string                { return "UA-19-002" }
func (FENoteIntegrity) Title() string             { return "FENote references are bidirectional and NoteType is valid" }
func (FENoteIntegrity) Category() engine.Category { return engine.CategoryNotes }
func (FENoteIntegrity) Severity() engine.Severity { return engine.SeverityError }
func (FENoteIntegrity) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (FENoteIntegrity) WCAG() []string            { return []string{"1.3.1"} }
func (FENoteIntegrity) Description() string {
	return "PDF/UA-2 §8.2.5.14 requires the /Ref link between a citation and its FENote (footnote/endnote) to be bidirectional -- the FENote must reference back every citation that references it and no others -- and a FENote's /NoteType, when present, to be Footnote, Endnote or None."
}

type elem struct {
	se   model.StructElement
	path string
}

func (c FENoteIntegrity) Run(doc model.Document) []engine.Finding {
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

	var all []elem
	collect(root, "/"+root.Type(), &all)

	var feNotes []elem
	for _, e := range all {
		if e.se.Type() == "FENote" {
			feNotes = append(feNotes, e)
		}
	}
	if len(feNotes) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no FENote elements -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	for _, f := range feNotes {
		// §8.2.5.14-4: NoteType value.
		if nt := f.se.Attribute("NoteType"); nt != "" && !validNoteTypes[nt] {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("FENote at %s has /NoteType %q -- must be Footnote, Endnote or None", f.path, nt),
				Hint:     "Set /NoteType to Footnote, Endnote or None.",
				Location: &engine.Location{Page: f.se.Page(), StructPath: f.path},
			})
		}

		// §8.2.5.14-2/-3: bidirectional /Ref.
		back := f.se.Refs()
		var referrers []elem
		for _, r := range all {
			if containsSame(r.se.Refs(), f.se) {
				referrers = append(referrers, r)
			}
		}
		// -2: a citation references the FENote but the FENote does not
		// reference it back.
		for _, r := range referrers {
			if !containsSame(back, r.se) {
				findings = append(findings, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("FENote at %s is referenced by %s via /Ref but does not reference it back", f.path, r.path),
					Hint:     "Add the citation to the FENote's /Ref array so the link is bidirectional.",
					Location: &engine.Location{Page: f.se.Page(), StructPath: f.path},
				})
			}
		}
		// -3: the FENote references an element that does not reference the
		// FENote back.
		for _, b := range back {
			if !containsSameElems(referrers, b) {
				findings = append(findings, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("FENote at %s references an element via /Ref that does not reference the FENote back", f.path),
					Hint:     "A FENote's /Ref must list exactly the citations that reference it.",
					Location: &engine.Location{Page: f.se.Page(), StructPath: f.path},
				})
			}
		}
	}
	return findings
}

func collect(e model.StructElement, path string, out *[]elem) {
	*out = append(*out, elem{se: e, path: path})
	for _, ch := range e.Children() {
		collect(ch, path+"/"+ch.Type(), out)
	}
}

func containsSame(list []model.StructElement, x model.StructElement) bool {
	for _, e := range list {
		if e.Same(x) {
			return true
		}
	}
	return false
}

func containsSameElems(list []elem, x model.StructElement) bool {
	for _, e := range list {
		if e.se.Same(x) {
			return true
		}
	}
	return false
}

func init() { engine.Register(FENoteIntegrity{}) }
