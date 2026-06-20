// Package viewerprefs groups checks that inspect the catalog's
// ViewerPreferences dictionary -- settings that control how compliant PDF
// viewers present the document.
package viewerprefs

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// DisplayDocTitle fails when ViewerPreferences/DisplayDocTitle is not
// present or not set to true. PDF/UA-1 §7.1 requires this so that
// conforming viewers show the document title in the window title bar
// instead of the file name -- otherwise the document title declared in
// metadata is invisible to users of assistive technology.
type DisplayDocTitle struct{}

func (DisplayDocTitle) ID() string    { return "UA-07-001" }
func (DisplayDocTitle) Title() string { return "ViewerPreferences/DisplayDocTitle is true" }
func (DisplayDocTitle) Description() string {
	return "Conforming viewers must show the document title (not the file name) in their window title bar. This is controlled by /ViewerPreferences<</DisplayDocTitle true>> on the catalog; without it the document title declared in metadata is invisible to users of assistive technology."
}
func (DisplayDocTitle) Category() engine.Category { return engine.CategoryViewerPrefs }
func (DisplayDocTitle) Severity() engine.Severity { return engine.SeverityError }
func (DisplayDocTitle) Spec() engine.Spec         { return engine.SpecBoth }
func (DisplayDocTitle) WCAG() []string            { return []string{"2.4.2"} }

func (c DisplayDocTitle) Run(doc model.Document) []engine.Finding {
	catalog, err := doc.Catalog()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read document catalog: " + err.Error(),
		}}
	}

	vpObj, found := catalog.Find("ViewerPreferences")
	if !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "no ViewerPreferences dictionary — DisplayDocTitle defaults to false",
			Hint:     "Add /ViewerPreferences << /DisplayDocTitle true >> to the document Catalog.",
		}}
	}

	vp, err := doc.DereferenceDict(vpObj)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read ViewerPreferences dictionary: " + err.Error(),
		}}
	}

	ddtObj, found := vp.Find("DisplayDocTitle")
	if !found {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "ViewerPreferences has no DisplayDocTitle entry — defaults to false",
			Hint:     "Set /DisplayDocTitle true inside the ViewerPreferences dictionary.",
		}}
	}

	value, err := doc.DereferenceBoolean(ddtObj)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read DisplayDocTitle value: " + err.Error(),
		}}
	}
	if !value {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "DisplayDocTitle is false",
			Hint:     "Change /DisplayDocTitle from false to true so viewers display the document title.",
		}}
	}
	return nil
}

func init() { engine.Register(DisplayDocTitle{}) }
