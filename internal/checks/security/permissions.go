// Package security groups checks that inspect document security
// settings (encryption, permissions).
package security

import (
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// Permissions fails when an encrypted document does not grant the
// "extract for accessibility" permission. PDF/UA-1 §7.20 forbids
// security handlers that block assistive technology from reading
// the document content: a PDF that locks out AT cannot, by
// definition, be accessible.
//
// The check inspects /P bit 10 (mask 512), which the spec
// specifically reserves for accessibility extraction. /P bit 5
// (mask 16) covers generic copy/extract; we report on it as a
// hint but do not fail on it because PDF/UA-1 mentions only bit 10
// directly.
type Permissions struct{}

func (Permissions) ID() string                { return "MH-26-001" }
func (Permissions) Title() string             { return "Encryption permits accessibility extraction" }
func (Permissions) Category() engine.Category { return engine.CategorySecurity }
func (Permissions) Severity() engine.Severity { return engine.SeverityError }
func (Permissions) Spec() engine.Spec         { return engine.SpecBoth }
func (Permissions) WCAG() []string            { return []string{"1.3.1"} }
func (Permissions) Description() string {
	return "PDF/UA-1 §7.20 requires the document's security handler to grant assistive-technology extraction. In the standard PDF security handler this is /P bit 10 (mask 512); a permission set that clears that bit makes the PDF inaccessible by construction, even if every other PDF/UA rule is satisfied."
}

func (c Permissions) Run(doc model.Document) []engine.Finding {
	enc := doc.Encryption()
	if !enc.Encrypted {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document is not encrypted -- AT extraction is unrestricted",
		}}
	}
	if !enc.AllowAccessibility {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "encryption permits do not grant accessibility extraction (/P bit 10 unset)",
			Hint:     "Update the security handler so /P keeps bit 10 set (mask 512). PDF/UA-1 §7.20 makes this bit mandatory for accessible documents.",
		}}
	}
	if !enc.AllowExtractText {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityWarning,
			Message:  "encryption permits accessibility extraction but blocks general copy/extract (/P bit 5 unset)",
			Hint:     "Consider also setting /P bit 5 (mask 16). With only the accessibility bit, conformant AT can read the document but copy/paste in regular viewers is blocked.",
		}}
	}
	return nil
}

func init() { engine.Register(Permissions{}) }
