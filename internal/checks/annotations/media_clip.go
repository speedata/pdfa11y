package annotations

import (
	"fmt"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// MediaClipCT fails when a media clip data dictionary omits its /CT (content
// type) entry. ISO 14289-1 §7.18.6.2 (veraPDF UA1:7.18.6.2-1, ISO 32000-1
// §13.2.4.2 Table 274): although /CT is optional in ISO 32000-1, PDF/UA-1
// requires it so that a player can identify the media format and offer a
// suitable, accessible handler. Presence is enough -- an empty /CT () counts.
//
// One finding per offending media clip. N/A when the document has no media
// clips. PDF/UA-1 only.
type MediaClipCT struct{}

func (MediaClipCT) ID() string { return "UA-28-022" }
func (MediaClipCT) Title() string {
	return "Media clip data dictionaries carry a /CT (content type) entry"
}
func (MediaClipCT) Category() engine.Category { return engine.CategoryInteractive }
func (MediaClipCT) Severity() engine.Severity { return engine.SeverityError }
func (MediaClipCT) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (MediaClipCT) WCAG() []string            { return []string{"1.1.1"} }
func (MediaClipCT) Description() string {
	return "ISO 14289-1 §7.18.6.2 requires the /CT (content type) entry in every media clip data dictionary. The content type lets a conforming reader identify the media format and select an accessible player; a clip without /CT cannot be handled reliably."
}

func (c MediaClipCT) Run(doc model.Document) []engine.Finding {
	clips := doc.MediaClips()
	if len(clips) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no media clips -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, clip := range clips {
		if clip.HasCT {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("media clip on page %d has no /CT (content type) entry", clip.Page),
			Hint:     "Add a /CT entry to the media clip data dictionary giving the media MIME type (e.g. video/mp4).",
			Location: &engine.Location{Page: clip.Page},
		})
	}
	return findings
}

func init() { engine.Register(MediaClipCT{}) }

// MediaClipAlt fails when a media clip data dictionary lacks a correct /Alt
// (alternate text) entry. ISO 14289-1 §7.18.6.2 (veraPDF UA1:7.18.6.2-2, ISO
// 32000-1 §13.2.4.2 Table 274): the /Alt array must provide a default textual
// description -- a (culture, text) pair whose culture string is empty -- with
// non-empty text, so assistive technology can describe the media when it cannot
// be played. A missing /Alt, or a default entry with empty text, fails.
//
// One finding per offending media clip. N/A when the document has no media
// clips. PDF/UA-1 only.
type MediaClipAlt struct{}

func (MediaClipAlt) ID() string { return "UA-28-023" }
func (MediaClipAlt) Title() string {
	return "Media clip data dictionaries carry a default /Alt description"
}
func (MediaClipAlt) Category() engine.Category { return engine.CategoryInteractive }
func (MediaClipAlt) Severity() engine.Severity { return engine.SeverityError }
func (MediaClipAlt) Spec() engine.Spec         { return engine.SpecPDFUA1 }
func (MediaClipAlt) WCAG() []string            { return []string{"1.1.1"} }
func (MediaClipAlt) Description() string {
	return "ISO 14289-1 §7.18.6.2 requires the /Alt entry in every media clip data dictionary to supply a default textual description: a (culture, text) pair with an empty culture string and non-empty text. Without it, assistive technology has no fallback description when the media cannot be played."
}

func (c MediaClipAlt) Run(doc model.Document) []engine.Finding {
	clips := doc.MediaClips()
	if len(clips) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no media clips -- nothing to inspect",
		}}
	}
	var findings []engine.Finding
	for _, clip := range clips {
		if clip.HasCorrectAlt {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("media clip on page %d has no default /Alt description with non-empty text", clip.Page),
			Hint:     "Provide an /Alt array with a default entry: an empty culture string followed by a non-empty description string.",
			Location: &engine.Location{Page: clip.Page},
		})
	}
	return findings
}

func init() { engine.Register(MediaClipAlt{}) }
