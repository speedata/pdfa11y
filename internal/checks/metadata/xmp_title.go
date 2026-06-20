package metadata

import (
	"regexp"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// XMPTitle fails when the document's XMP metadata stream lacks a
// dc:title element. PDF/UA-1 §7.1 requires the document title to
// be expressed in XMP under the Dublin Core namespace; PDF 2.0 /
// PDF/UA-2 keep this requirement and additionally deprecate the
// DocumentInfo dictionary for everything but ModDate, so XMP
// dc:title is the single normative source. UA-06-005 verifies that
// a DocumentInfo /Title (when present) agrees with XMP dc:title;
// it does not demand that DocumentInfo carry the title at all.
//
// Limitation: same regex-on-raw-XMP approach as UA-06-003. A namespace
// prefix other than "dc" will yield a false negative; replacing this
// with a proper XML/namespace-aware parser is a planned cleanup.
type XMPTitle struct{}

func (XMPTitle) ID() string    { return "UA-06-004" }
func (XMPTitle) Title() string { return "XMP metadata contains dc:title" }
func (XMPTitle) Description() string {
	return "PDF/UA-1 §7.1 requires the document title to be expressed via dc:title in the XMP metadata stream (Dublin Core namespace). PDF 2.0 / PDF/UA-2 keep this requirement and deprecate the DocumentInfo dictionary for everything except ModDate, so XMP dc:title is the normative single source. UA-06-005 still cross-checks DocumentInfo /Title when one happens to be present."
}
func (XMPTitle) Category() engine.Category { return engine.CategoryMetadata }
func (XMPTitle) Severity() engine.Severity { return engine.SeverityError }
func (XMPTitle) Spec() engine.Spec         { return engine.SpecBoth }
func (XMPTitle) WCAG() []string            { return []string{"2.4.2"} }

// dc:title can appear as a bare element or wrapped in rdf:Alt/rdf:li
// for language alternatives. Both forms include the literal "<dc:title"
// in the stream, so an opening-tag match is sufficient as a presence
// signal. Self-closing tags (<dc:title/>) would be empty and are not
// matched on purpose.
var dcTitleOpenTag = regexp.MustCompile(`<\s*dc:title[\s>]`)

func (c XMPTitle) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}

	mdObj, found := catalog.Find("Metadata")
	if !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "document has no /Metadata stream — XMP dc:title cannot be declared",
			Hint:     "Embed an XMP metadata stream containing a dc:title element.",
		}}
	}

	content, err := doc.DecodeStream(mdObj)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata stream: " + err.Error(),
		}}
	}

	if !dcTitleOpenTag.Match(content) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "XMP metadata contains no dc:title element",
			Hint:     "Add <dc:title>…</dc:title> (typically inside rdf:Alt with xml:lang) to the XMP stream so the title is exposed via Dublin Core.",
		}}
	}
	return nil
}

func init() { engine.Register(XMPTitle{}) }
