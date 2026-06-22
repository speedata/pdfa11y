// Package files groups checks that inspect embedded and associated
// file specifications (Matterhorn schedule 12).
package files

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// AFRelationship requires every filespec reachable through an /AF
// (Associated Files) array to declare /AFRelationship. PDF 2.0
// introduces the /AF mechanism: a file associated with the document,
// a page, an annotation or an XObject is connected to it via /AF, and
// the spec mandates a Name in /AFRelationship that says what kind of
// connection it is (Source, Data, Supplement, Alternative,
// EncryptedPayload, FormData, Schema, Unspecified). PDF/UA leans on
// this for AT-driven enumeration of supplementary material.
//
// Spec gating: Associated Files are an ISO 32000-2 (PDF 2.0) feature.
// ISO 14289-1 / PDF 1.7 has no /AF mechanism, so the requirement lives
// only in ISO 14289-2 (§8.9.2.4.10, §7.6.3) -- this check is PDF/UA-2
// only.
//
// Limitation: only catalog- and page-level /AF arrays are walked
// today. Annotation- and XObject-level /AF arrays will produce false
// negatives until the backend exposes them. The walker reports the
// surface ("Catalog" or "Page N") so future surfaces fit the same
// finding shape.
type AFRelationship struct{}

func (AFRelationship) ID() string                { return "UA-12-001" }
func (AFRelationship) Title() string             { return "Associated Files declare /AFRelationship" }
func (AFRelationship) Category() engine.Category { return engine.CategoryEmbeddedFiles }
func (AFRelationship) Severity() engine.Severity { return engine.SeverityError }
func (AFRelationship) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (AFRelationship) WCAG() []string            { return []string{"1.3.1"} }
func (AFRelationship) Description() string {
	return "ISO 32000-2 §14.13 (Associated Files) requires every filespec referenced from an /AF array to declare /AFRelationship -- a Name that classifies the file's role (Source, Data, Supplement, ...). PDF/UA leans on this entry for AT-driven enumeration of supplementary material; without it consumers cannot describe the file's purpose."
}

func (c AFRelationship) Run(doc model.Document) []engine.Finding {
	files, err := doc.AssociatedFiles()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate associated files: " + err.Error(),
		}}
	}
	if len(files) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no /AF arrays on catalog or pages -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, f := range files {
		if f.Relationship != "" {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("%s /AF entry %q has no /AFRelationship", f.SourcePath, displayFilename(f.Filename)),
			Hint:     "Set /AFRelationship on the filespec to one of Source, Data, Alternative, Supplement, EncryptedPayload, FormData, Schema or Unspecified.",
		})
	}
	return findings
}

// displayFilename falls back to a placeholder so the finding message
// stays readable when the filespec carried neither /F nor /UF.
func displayFilename(s string) string {
	if s == "" {
		return "(unnamed)"
	}
	return s
}

func init() { engine.Register(AFRelationship{}) }
