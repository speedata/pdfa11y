package structure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// standardNamespaces are the namespaces in which the ISO 32000 standard
// structure types are defined; a standard type is "the standard type" only
// inside one of these. Remapping such a type is forbidden
// (ISO 32000-2 §14.8.6 / §14.8.4).
var standardNamespaces = map[string]bool{
	"http://iso.org/pdf/ssn":  true, // PDF 1.7
	"http://iso.org/pdf2/ssn": true, // PDF 2.0
}

// RemappedStandardType fails when a standard structure type is role-mapped to
// another type. PDF/UA-1 §7.1 (test 7.1-7) and PDF/UA-2 §8.2.4-4 forbid
// remapping the standard tags defined in ISO 32000 §14.8.4: assistive
// technology relies on their fixed meaning, so redefining e.g. H1 or Table
// breaks that contract.
//
// The classic /RoleMap is namespace-agnostic and lives in the default
// (standard) namespace, so any standard-type key there is a violation. A
// /RoleMapNS entry is a violation only when its key is a standard type and the
// declaring namespace is itself a standard namespace -- a "standard-named"
// type in a custom namespace is not the standard type and may be remapped.
type RemappedStandardType struct{}

func (RemappedStandardType) ID() string                { return "UA-31-010" }
func (RemappedStandardType) Title() string             { return "Standard structure types are not remapped" }
func (RemappedStandardType) Category() engine.Category { return engine.CategoryStructure }
func (RemappedStandardType) Severity() engine.Severity { return engine.SeverityError }
func (RemappedStandardType) Spec() engine.Spec         { return engine.SpecBoth }
func (RemappedStandardType) WCAG() []string            { return []string{"1.3.1"} }
func (RemappedStandardType) Description() string {
	return "PDF/UA-1 §7.1 and PDF/UA-2 §8.2.4 forbid role-mapping a standard structure type (the tags defined in ISO 32000 §14.8.4, e.g. P, H1, Table) to another type. Their meaning is fixed; remapping them misleads assistive technology."
}

func (c RemappedStandardType) Run(doc model.Document) []engine.Finding {
	classic := doc.RoleMap()
	namespaces := doc.Namespaces()

	nsMappings := 0
	for _, ns := range namespaces {
		nsMappings += len(ns.RoleMapNS)
	}
	if len(classic) == 0 && nsMappings == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document declares no role mappings -- nothing to inspect",
		}}
	}

	var remapped []string

	// A standard type is "remapped" in the forbidden sense when it is
	// mapped to a NON-standard type -- that strips its ISO 32000 meaning.
	// Mapping to another standard type (e.g. the PDF 2.0 type Title, which
	// is not standard under PDF/UA-1, onto P) is how producers legitimately
	// fold extra heading/title levels into the standard set, so it is not
	// flagged. This also keeps the spec-agnostic standard-type set from
	// over-reporting on PDF/UA-1 documents.
	for src, dst := range classic {
		if model.IsStandardStructureType(src) && !model.IsStandardStructureType(dst) {
			remapped = append(remapped, fmt.Sprintf("%s (-> %s)", src, dst))
		}
	}

	// Namespace /RoleMapNS: only standard types in a standard namespace,
	// mapped to a non-standard target.
	for _, ns := range namespaces {
		if !standardNamespaces[ns.URI] {
			continue
		}
		for src, targets := range ns.RoleMapNS {
			if model.IsStandardStructureType(src) && len(targets) > 0 && !model.IsStandardStructureType(targets[0].Type) {
				remapped = append(remapped, fmt.Sprintf("%s (namespace %s)", src, ns.URI))
			}
		}
	}

	if len(remapped) == 0 {
		return nil
	}
	sort.Strings(remapped)
	return []engine.Finding{{
		CheckID:  c.ID(),
		Severity: engine.SeverityError,
		Message:  "standard structure type(s) are remapped in the role map: " + strings.Join(remapped, ", "),
		Hint:     "Remove these entries from /RoleMap / /RoleMapNS: standard types must keep their ISO 32000 meaning. Role-map your own custom types instead.",
	}}
}

func init() { engine.Register(RemappedStandardType{}) }
