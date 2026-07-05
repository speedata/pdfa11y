package files

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// EmbeddedDesc requires every file specification dictionary in the catalog's
// /Names /EmbeddedFiles name tree to carry a /Desc entry. ISO 14289-2 §8.14.1:
// "The Desc entry shall be present on all file specification dictionaries
// present in the EmbeddedFiles name tree of a conforming document." The /Desc
// string is the human-readable description assistive technology announces for
// an embedded file; without it the attachment is an unlabelled blob.
//
// Filespecs reached only through /AF (not listed in the EmbeddedFiles tree)
// are out of scope here -- their requirement is /AFRelationship (UA-12-001).
//
// One finding per embedded filespec without /Desc. N/A when the document has
// no embedded files. PDF/UA-2 only.
type EmbeddedDesc struct{}

func (EmbeddedDesc) ID() string                { return "UA-12-002" }
func (EmbeddedDesc) Title() string             { return "Embedded file specifications declare /Desc" }
func (EmbeddedDesc) Category() engine.Category { return engine.CategoryEmbeddedFiles }
func (EmbeddedDesc) Severity() engine.Severity { return engine.SeverityError }
func (EmbeddedDesc) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (EmbeddedDesc) WCAG() []string            { return []string{"1.3.1"} }
func (EmbeddedDesc) Description() string {
	return "ISO 14289-2 §8.14.1 requires every file specification dictionary in the catalog's /Names /EmbeddedFiles name tree to carry a /Desc entry: the human-readable description an assistive technology announces for the attachment. An embedded file without /Desc is presented as an unlabelled blob."
}

func (c EmbeddedDesc) Run(doc model.Document) []engine.Finding {
	specs := doc.EmbeddedFileSpecs()
	if len(specs) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no embedded files -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, s := range specs {
		if s.HasDesc {
			continue
		}
		name := s.Filename
		if name == "" {
			name = "(unnamed)"
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("embedded file %q has no /Desc entry", name),
			Hint:     "Add a /Desc string to the file specification dictionary describing the attachment for assistive technology.",
		})
	}
	return findings
}

func init() { engine.Register(EmbeddedDesc{}) }
