package annotations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// TextFieldRVValue fails when a text field carries a rich-text value (/RV) but
// no plain value (/V), or the two are not textually equivalent. ISO 14289-2
// §8.10.3.3 (veraPDF UA2:8.10.3.3-1): for text fields, when an /RV entry is
// present a /V entry shall also be present and the two shall be textually
// equivalent. Assistive technology reads the plain /V; if it is missing or
// disagrees with the displayed rich text, AT presents a value different from
// what the user sees.
//
// One finding per offending text field. N/A when no text field carries /RV.
// PDF/UA-2 only.
type TextFieldRVValue struct{}

func (TextFieldRVValue) ID() string                { return "UA-28-033" }
func (TextFieldRVValue) Title() string             { return "Text field /RV matches its /V value" }
func (TextFieldRVValue) Category() engine.Category { return engine.CategoryInteractive }
func (TextFieldRVValue) Severity() engine.Severity { return engine.SeverityError }
func (TextFieldRVValue) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (TextFieldRVValue) WCAG() []string            { return []string{"1.3.1", "4.1.2"} }
func (TextFieldRVValue) Description() string {
	return "ISO 14289-2 §8.10.3.3 requires that when a text field carries a rich-text value /RV, it also carries a plain value /V and the two are textually equivalent. Assistive technology reads /V; a missing or divergent /V makes AT announce a value different from the rich text the user sees. The comparison strips the /RV markup and compares the plain text."
}

func (c TextFieldRVValue) Run(doc model.Document) []engine.Finding {
	annots, err := doc.Annotations()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot enumerate annotations: " + err.Error(),
		}}
	}
	var findings []engine.Finding
	seen := 0
	reported := map[string]bool{}
	for _, a := range annots {
		if !a.IsTextField || a.RichTextValue == "" {
			continue
		}
		seen++
		// A field's widgets all resolve to the same /RV and /V; report the
		// field once.
		key := a.RichTextValue + "\x00" + a.FieldValue
		if reported[key] {
			continue
		}
		if a.FieldValue == "" {
			reported[key] = true
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  "text field carries a rich-text value /RV but no plain value /V",
				Hint:     "Add a /V entry holding the plain-text equivalent of the field's rich-text /RV value.",
				Location: &engine.Location{Page: a.Page},
			})
			continue
		}
		if normalizeRichText(a.RichTextValue) != normalizeRichText(a.FieldValue) {
			reported[key] = true
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("text field /RV and /V are not textually equivalent (/RV plain text %q vs /V %q)", normalizeRichText(a.RichTextValue), normalizeRichText(a.FieldValue)),
				Hint:     "Make the plain-text /V value equal to the text content of the rich-text /RV value.",
				Location: &engine.Location{Page: a.Page},
			})
		}
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no text field carries a rich-text /RV value -- nothing to inspect",
		}}
	}
	return findings
}

var richTextTag = regexp.MustCompile(`<[^>]*>`)

// normalizeRichText reduces an /RV rich-text (XHTML) fragment or a plain /V
// value to comparable plain text: strip XML tags and processing instructions,
// unescape the common XML entities, and collapse whitespace. This mirrors the
// plain-text extraction veraPDF applies before comparing /RV and /V.
func normalizeRichText(s string) string {
	s = richTextTag.ReplaceAllString(s, "")
	replacer := strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", `"`, "&apos;", "'", "&#39;", "'", "&nbsp;", " ",
	)
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func init() { engine.Register(TextFieldRVValue{}) }
