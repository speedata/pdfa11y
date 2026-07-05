package engine

import (
	"encoding/json"
	"fmt"

	"github.com/speedata/pdfa11y/internal/model"
)

// Spec identifies which PDF/UA version a check applies to.
type Spec uint8

const (
	SpecPDFUA1 Spec = 1 << iota
	SpecPDFUA2
	SpecBoth = SpecPDFUA1 | SpecPDFUA2
)

func (s Spec) String() string {
	switch s {
	case SpecPDFUA1:
		return "PDF/UA-1"
	case SpecPDFUA2:
		return "PDF/UA-2"
	case SpecBoth:
		return "PDF/UA-1+2"
	}
	return "unknown"
}

func (s Spec) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

func (s *Spec) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	switch str {
	case "PDF/UA-1":
		*s = SpecPDFUA1
	case "PDF/UA-2":
		*s = SpecPDFUA2
	case "PDF/UA-1+2":
		*s = SpecBoth
	default:
		return fmt.Errorf("unknown spec %q", str)
	}
	return nil
}

// Severity describes how a finding affects conformance. The
// "notApplicable" severity signals "this check could not be applied
// because the document does not exercise its preconditions" (e.g. a
// structure-walking check on a document without a structure tree).
// N/A findings never break conformance.
type Severity uint8

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityNotApplicable
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityNotApplicable:
		return "notApplicable"
	}
	return "unknown"
}

func (s Severity) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

func (s *Severity) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	switch str {
	case "info":
		*s = SeverityInfo
	case "warning":
		*s = SeverityWarning
	case "error":
		*s = SeverityError
	case "notApplicable":
		*s = SeverityNotApplicable
	default:
		return fmt.Errorf("unknown severity %q", str)
	}
	return nil
}

// Category groups checks by subject area. The leading number mirrors
// the Matterhorn protocol's schedule grouping (kept as a stable
// ordering, not a claim of Matterhorn coverage for PDF/UA-2).
type Category string

const (
	CategoryRealContent     Category = "01 Real content"
	CategoryFonts           Category = "09 Fonts"
	CategoryNaturalLanguage Category = "11 Natural language"
	CategoryEmbeddedFiles   Category = "12 Embedded files"
	CategoryGraphics        Category = "13 Graphics"
	CategoryHeadings        Category = "14 Headings"
	CategoryTables          Category = "15 Tables"
	CategoryLists           Category = "16 Lists"
	CategoryMetadata        Category = "06 Metadata"
	CategoryViewerPrefs     Category = "07 Viewer preferences"
	CategoryInteractive     Category = "28 Annotations and forms"
	CategoryStructure       Category = "01 Structure tree"
	CategoryNavigation      Category = "27 Navigation"
	CategoryTabOrder        Category = "08 Tab order"
	CategoryOptionalContent Category = "20 Optional content"
	CategorySecurity        Category = "26 Security"
	CategoryNotes           Category = "19 Notes and references"
	CategoryXObjects        Category = "30 XObjects"
)

// Location optionally pinpoints where a finding occurs.
type Location struct {
	Page         int    // 1-based, 0 if document-wide
	ObjectNumber int    // PDF object number, 0 if not applicable
	StructPath   string // e.g. "/Document/Sect/P[3]"
}

// Finding is a single result produced by a Check.
type Finding struct {
	CheckID  string
	Severity Severity
	Message  string
	Hint     string
	Location *Location
}

// Check is the interface every accessibility rule implements.
type Check interface {
	ID() string          // stable identifier "UA-<schedule>-<nnn>", e.g. "UA-01-005" (schedule numbers follow the Matterhorn protocol's grouping)
	Title() string       // short human-readable summary
	Description() string // 1-3 sentence explanation: what does this check verify, and why
	Category() Category
	Severity() Severity // severity used when the check fails
	Spec() Spec         // PDF/UA version(s) the check applies to
	WCAG() []string     // related WCAG 2.x success criteria, e.g. ["1.1.1"]
	Run(model.Document) []Finding
}

// Result bundles a check with its findings.
type Result struct {
	Check    Check
	Findings []Finding
}

// Passed reports whether the check produced no error-level findings.
func (r Result) Passed() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return false
		}
	}
	return true
}
