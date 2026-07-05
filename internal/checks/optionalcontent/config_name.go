package optionalcontent

import (
	"fmt"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
	"github.com/speedata/pdfa11y/internal/pdfua"
)

// ConfigName fails when an optional-content configuration dictionary -- the
// /OCProperties /D default config, or an entry of the /Configs array -- lacks
// a non-empty /Name. The requirement differs by conformance level:
//
//   - PDF/UA-1 §7.10-1 (veraPDF UA1:7.10-1): unconditional -- every
//     configuration (the /D default and each /Configs entry) shall have /Name.
//   - PDF/UA-2 §8.7-1 (veraPDF UA2:8.7-1): conditional -- /Name is required on
//     all configurations (including /D) only when the document declares a
//     /Configs array with at least one configuration. A document with only a
//     /D config and no /Configs is exempt.
//
// This is distinct from UA-20-001, which requires /Name on each OCG (layer)
// rather than on the configurations.
//
// One finding per unnamed configuration. N/A when the document declares no
// PDF/UA level, has no optional-content configurations, or (PDF/UA-2) is exempt
// for want of a /Configs array.
type ConfigName struct{}

func (ConfigName) ID() string                { return "UA-20-003" }
func (ConfigName) Title() string             { return "Optional Content configurations declare /Name" }
func (ConfigName) Category() engine.Category { return engine.CategoryOptionalContent }
func (ConfigName) Severity() engine.Severity { return engine.SeverityError }
func (ConfigName) Spec() engine.Spec         { return engine.SpecBoth }
func (ConfigName) WCAG() []string            { return []string{"1.3.1"} }
func (ConfigName) Description() string {
	return "ISO 14289-1 §7.10 / ISO 14289-2 §8.7 require every optional-content configuration dictionary (the /OCProperties /D default configuration and each member of /Configs) to carry a non-empty /Name, so the layer configuration can be identified in the viewer. This is separate from the per-OCG /Name requirement (UA-20-001)."
}

func (c ConfigName) Run(doc model.Document) []engine.Finding {
	part, found, err := pdfua.DetectPart(doc)
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read XMP metadata: " + err.Error(),
		}}
	}
	if !found || (part != 1 && part != 2) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document does not declare a PDF/UA conformance level (pdfuaid:part 1 or 2)",
		}}
	}
	configs := doc.OptionalContentConfigs()
	if len(configs) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no optional-content configurations to inspect",
		}}
	}
	// PDF/UA-2 §8.7-1 requires /Name only when a /Configs array is present
	// with at least one configuration. A /Configs member surfaces with a
	// Source of "Configs[i]"; a document with only the /D config is exempt.
	if part == 2 {
		hasConfigs := false
		for _, cfg := range configs {
			if strings.HasPrefix(cfg.Source, "Configs") {
				hasConfigs = true
				break
			}
		}
		if !hasConfigs {
			return []engine.Finding{{
				CheckID:  c.ID(),
				Severity: engine.SeverityNotApplicable,
				Message:  "PDF/UA-2 document has no /Configs array -- /Name is not required on the default configuration",
			}}
		}
	}
	var findings []engine.Finding
	for _, cfg := range configs {
		if cfg.HasName {
			continue
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("optional-content configuration /OCProperties/%s has no non-empty /Name", cfg.Source),
			Hint:     "Add a non-empty /Name string to the configuration dictionary describing the layer configuration.",
		})
	}
	return findings
}

func init() { engine.Register(ConfigName{}) }
