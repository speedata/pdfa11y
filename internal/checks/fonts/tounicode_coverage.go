package fonts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// ToUnicodeCoverage fails when a font's /ToUnicode CMap is present
// but does not actually map the codes that the content stream uses
// to render with that font. PDF/UA-1 §7.21.3 demands a working
// Unicode mapping for every rendered character; merely shipping a
// /ToUnicode stream is not enough if it is empty or omits the
// codes that the page actually shows.
//
// Complements MH-10-001:
//   - MH-10-001 fires when /ToUnicode is missing entirely (or the
//     font cannot rely on the predefined-encoding shortcut).
//   - MH-10-002 (this check) fires when /ToUnicode is present but
//     its bfchar/bfrange tables either are empty (the F01 case) or
//     do not cover the codes used in Tj/TJ.
//
// Fonts that have no /ToUnicode entry at all are intentionally
// skipped here: that case is MH-10-001's responsibility.
type ToUnicodeCoverage struct{}

func (ToUnicodeCoverage) ID() string                { return "MH-10-002" }
func (ToUnicodeCoverage) Title() string             { return "/ToUnicode covers every used code" }
func (ToUnicodeCoverage) Category() engine.Category { return engine.CategoryFonts }
func (ToUnicodeCoverage) Severity() engine.Severity { return engine.SeverityError }
func (ToUnicodeCoverage) Spec() engine.Spec         { return engine.SpecBoth }
func (ToUnicodeCoverage) WCAG() []string            { return []string{"1.3.1"} }
func (ToUnicodeCoverage) Description() string {
	return "PDF/UA-1 §7.21.3 requires every rendered code to have a working Unicode mapping. A /ToUnicode CMap that is empty (only a codespace range, no bfchar/bfrange) -- or that maps some codes but not the ones actually used on the page -- silently fails text extraction. This check matches each Tj/TJ code against the font's /ToUnicode coverage."
}

func (c ToUnicodeCoverage) Run(doc model.Document) []engine.Finding {
	pages, err := doc.Pages()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot walk pages: " + err.Error(),
		}}
	}
	if len(pages) == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no pages -- nothing to inspect",
		}}
	}

	// Aggregate per font (keyed by Subtype + BaseFont so the same
	// logical font referenced from multiple resource keys collapses
	// to one report). Per font we collect: the parsed ToUnicode map
	// (nil if the font has no /ToUnicode at all -- then MH-10-001's
	// territory) and the union of codes used across all pages.
	type fontUsage struct {
		font       model.Font
		usedCodes  map[uint32]bool
		firstPage  int
	}
	fonts := map[string]*fontUsage{}

	consideredAny := false
	for _, p := range pages {
		for key, f := range p.UsedFonts {
			if f.ToUnicodeMappings == nil {
				continue // MH-10-001 already covers missing /ToUnicode
			}
			consideredAny = true
			id := fontKey(f)
			fu, ok := fonts[id]
			if !ok {
				fu = &fontUsage{font: f, usedCodes: map[uint32]bool{}, firstPage: p.Number}
				fonts[id] = fu
			}
			for code := range p.FontCodes[key] {
				fu.usedCodes[code] = true
			}
		}
	}
	if !consideredAny {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "no fonts with a /ToUnicode CMap to inspect",
		}}
	}

	ids := make([]string, 0, len(fonts))
	for id := range fonts {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var findings []engine.Finding
	for _, id := range ids {
		fu := fonts[id]
		if len(fu.font.ToUnicodeMappings) == 0 {
			findings = append(findings, engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("font %s has an empty /ToUnicode CMap (no bfchar/bfrange entries)", describe(fu.font)),
				Hint:     "Add bfchar or bfrange entries to the /ToUnicode stream so each code rendered with this font has a Unicode equivalent.",
				Location: &engine.Location{Page: fu.firstPage},
			})
			continue
		}
		var missing []uint32
		for code := range fu.usedCodes {
			if _, mapped := fu.font.ToUnicodeMappings[code]; !mapped {
				missing = append(missing, code)
			}
		}
		if len(missing) == 0 {
			continue
		}
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		const maxSamples = 8
		samples := missing
		if len(samples) > maxSamples {
			samples = samples[:maxSamples]
		}
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  fmt.Sprintf("font %s renders %d code(s) not covered by its /ToUnicode CMap (e.g. %s)", describe(fu.font), len(missing), formatCodes(samples)),
			Hint:     "Extend the /ToUnicode stream's bfchar/bfrange entries to include every code the content stream renders with this font.",
			Location: &engine.Location{Page: fu.firstPage},
		})
	}
	return findings
}

func formatCodes(codes []uint32) string {
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = fmt.Sprintf("0x%04X", c)
	}
	return strings.Join(parts, ", ")
}

func init() { engine.Register(ToUnicodeCoverage{}) }
