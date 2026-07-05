package files

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// EmbeddedNames requires the file specification dictionary of an embedded file
// to carry non-empty /F and /UF keys. ISO 14289-1 §7.11 (ISO 32000-1
// §7.11.3): a filespec that embeds a file (via /EF) shall name it in both the
// legacy /F and the Unicode /UF form so assistive technology and viewers can
// present a reliable, portable file name.
//
// Scope: file specifications reached through the catalog's /Names
// /EmbeddedFiles name tree. Filespecs embedded only via /AF are not walked
// here (a false negative, noted for a later extension).
//
// One finding per offending embedded filespec. N/A when the document has no
// embedded files.
type EmbeddedNames struct{}

func (EmbeddedNames) ID() string                { return "UA-12-003" }
func (EmbeddedNames) Title() string             { return "Embedded file specifications have non-empty /F and /UF" }
func (EmbeddedNames) Category() engine.Category { return engine.CategoryEmbeddedFiles }
func (EmbeddedNames) Severity() engine.Severity { return engine.SeverityError }
func (EmbeddedNames) Spec() engine.Spec         { return engine.SpecBoth }
func (EmbeddedNames) WCAG() []string            { return []string{"1.3.1"} }
func (EmbeddedNames) Description() string {
	return "ISO 14289-1 §7.11 requires the file specification dictionary of an embedded file to contain non-empty /F and /UF keys (the legacy and the Unicode file name). A missing or empty name leaves the attachment without a portable, AT-presentable identifier."
}

func (c EmbeddedNames) Run(doc model.Document) []engine.Finding {
	specs := doc.EmbeddedFileSpecs()
	if len(specs) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no embedded files -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	considered := 0
	for _, s := range specs {
		if !s.HasEF {
			continue // §7.11 only constrains filespecs that embed a file
		}
		considered++
		if s.HasNonEmptyF && s.HasNonEmptyUF {
			continue
		}
		var missing string
		switch {
		case !s.HasNonEmptyF && !s.HasNonEmptyUF:
			missing = "both /F and /UF"
		case !s.HasNonEmptyF:
			missing = "/F"
		default:
			missing = "/UF"
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "embedded file specification is missing a non-empty " + missing,
			Hint:     "Set both /F and /UF on the file specification dictionary to the (non-empty) file name.",
		})
	}
	if considered == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no embedded-file specifications (/EF) -- nothing to inspect",
		}}
	}
	return findings
}

func init() { engine.Register(EmbeddedNames{}) }
