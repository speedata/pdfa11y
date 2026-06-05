package metadata

import (
	"regexp"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// TitleAgreement fails when the document title in DocumentInfo
// /Title and in the XMP dc:title element disagree. PDF/UA-1 §7.1
// requires both to be present (covered by MH-06-001 and MH-06-004)
// and to express the same title -- otherwise readers see different
// titles in different surfaces (window title, library catalogues,
// search results, AT announcements).
//
// Limitation: the XMP side is parsed with a small regex over the raw
// stream, same approach as MH-06-004. Namespace prefixes other than
// "dc" or unusual structural variants will produce false negatives.
// A proper XML-aware parser is a planned cleanup across all three
// metadata checks.
type TitleAgreement struct{}

func (TitleAgreement) ID() string                { return "MH-06-005" }
func (TitleAgreement) Title() string             { return "DocumentInfo /Title and XMP dc:title agree" }
func (TitleAgreement) Category() engine.Category { return engine.CategoryMetadata }
func (TitleAgreement) Severity() engine.Severity { return engine.SeverityError }
func (TitleAgreement) Spec() engine.Spec         { return engine.SpecBoth }
func (TitleAgreement) WCAG() []string            { return []string{"2.4.2"} }
func (TitleAgreement) Description() string {
	return "PDF/UA-1 §7.1 requires the title to be declared identically in DocumentInfo /Title and in the XMP dc:title element. Divergence between the two leaves AT, viewers and indexers showing different titles depending on which metadata source they consult."
}

// dc:title may be a bare element or wrap an rdf:Alt with rdf:li per
// language. We extract the inner text of the first rdf:li, falling
// back to the dc:title body when there is no rdf:Alt wrapper. Same
// heuristic as MH-06-004's presence regex, extended to capture.
var (
	dcTitleBody  = regexp.MustCompile(`(?s)<dc:title[^>]*>(.*?)</dc:title>`)
	rdfLiBody    = regexp.MustCompile(`(?s)<rdf:li[^>]*>(.*?)</rdf:li>`)
	xmlTagStrip  = regexp.MustCompile(`<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

func (c TitleAgreement) Run(doc model.Document) []engine.Finding {
	infoTitle := normaliseTitle(doc.DocInfo().Title)
	xmpTitle, found, err := extractXMPTitle(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata stream: " + err.Error(),
		}}
	}
	if !found {
		// MH-06-004 already flags the missing XMP title. Decline.
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "XMP metadata has no dc:title -- title agreement cannot be evaluated",
		}}
	}
	if infoTitle == "" {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "DocumentInfo /Title is absent -- title agreement cannot be evaluated",
		}}
	}
	xmp := normaliseTitle(xmpTitle)
	if infoTitle == xmp {
		return nil
	}
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "DocumentInfo /Title and XMP dc:title differ: " + quote(infoTitle) + " vs " + quote(xmp),
		Hint:     "Synchronise the two title fields: most authoring tools have a single 'document title' input that should populate both DocumentInfo /Title and XMP dc:title.",
	}}
}

func extractXMPTitle(doc model.Document) (string, bool, error) {
	cat, err := doc.Catalog()
	if err != nil {
		return "", false, err
	}
	mdObj, ok := cat.Find("Metadata")
	if !ok {
		return "", false, nil
	}
	content, err := doc.DecodeStream(mdObj)
	if err != nil {
		return "", false, err
	}
	m := dcTitleBody.FindSubmatch(content)
	if m == nil {
		return "", false, nil
	}
	body := m[1]
	if li := rdfLiBody.FindSubmatch(body); li != nil {
		body = li[1]
	}
	// Strip any residual XML tags (e.g. namespaced wrappers we did
	// not anticipate) and decode the small set of XML entities that
	// can appear in title text.
	stripped := xmlTagStrip.ReplaceAll(body, []byte(""))
	text := decodeXMLEntities(string(stripped))
	return strings.TrimSpace(text), true, nil
}

func normaliseTitle(s string) string {
	return whitespaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

func decodeXMLEntities(s string) string {
	r := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&apos;", "'",
	)
	return r.Replace(s)
}

func quote(s string) string {
	return `"` + s + `"`
}

func init() { engine.Register(TitleAgreement{}) }
