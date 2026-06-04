// Package metadata groups checks that inspect document-level metadata such
// as Title, Author, Language and the PDF/UA identifier in XMP.
package metadata

import (
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// DocumentTitle fails if the document has no non-empty /Title in its
// DocumentInfo dictionary. PDF/UA-1 §7.1 requires the title to be
// expressed in both DocumentInfo /Title and XMP dc:title -- this check
// covers the DocumentInfo side; the XMP side is the companion check
// MH-06-004. Both must pass for a fully conforming document.
type DocumentTitle struct{}

func (DocumentTitle) ID() string    { return "MH-06-001" }
func (DocumentTitle) Title() string { return "Document has a title in metadata" }
func (DocumentTitle) Description() string {
	return "PDF/UA-1 §7.1 requires the document title to be set in the DocumentInfo dictionary so screen readers can announce the document by name instead of by file name. The same title must also appear in XMP under the Dublin Core namespace (covered by MH-06-004)."
}
func (DocumentTitle) Category() engine.Category { return engine.CategoryMetadata }
func (DocumentTitle) Severity() engine.Severity { return engine.SeverityError }
func (DocumentTitle) Spec() engine.Spec         { return engine.SpecBoth }
func (DocumentTitle) WCAG() []string            { return []string{"2.4.2"} }

func (c DocumentTitle) Run(doc model.Document) []engine.Finding {
	if strings.TrimSpace(doc.DocInfo().Title) == "" {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "document has no /Title in the DocumentInfo dictionary",
			Hint:     "Set the document title in the PDF DocumentInfo before export; an XMP dc:title alone does not satisfy PDF/UA §7.1.",
		}}
	}
	return nil
}

func init() { engine.Register(DocumentTitle{}) }
