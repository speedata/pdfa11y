package structure

import (
	"fmt"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// RubyStructure fails when a Ruby structure element does not have one of the
// two child sequences ISO 14289-2 §8.2.5.23 permits (veraPDF UA2:8.2.5.23-1):
//
//   - RB, RT                 (base text + ruby annotation), or
//   - RB, RP, RT, RP         (with ruby punctuation around the annotation).
//
// The comparison is over the ordered *standard* types of the direct children
// (custom types are resolved through the classic /RoleMap), so both order and
// exact membership matter. One finding per malformed Ruby. N/A when the
// document has no Ruby elements. PDF/UA-2 only.
type RubyStructure struct{}

func (RubyStructure) ID() string                { return "UA-01-017" }
func (RubyStructure) Title() string             { return "Ruby elements have a valid child sequence" }
func (RubyStructure) Category() engine.Category { return engine.CategoryStructure }
func (RubyStructure) Severity() engine.Severity { return engine.SeverityError }
func (RubyStructure) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (RubyStructure) WCAG() []string            { return []string{"1.3.1"} }
func (RubyStructure) Description() string {
	return "ISO 14289-2 §8.2.5.23 requires a Ruby structure element to contain either a single RB followed by a single RT, or the four-element sequence RB, RP, RT, RP. Any other child sequence leaves the base text and its phonetic annotation unpaired, so assistive technology cannot present the ruby relationship."
}

func (c RubyStructure) Run(doc model.Document) []engine.Finding {
	return checkChildSequence(doc, c.ID(), "Ruby",
		[]string{"RB&RT", "RB&RP&RT&RP"},
		"Use the child sequence RB, RT or RB, RP, RT, RP inside the Ruby element.")
}

func init() { engine.Register(RubyStructure{}) }

// WarichuStructure fails when a Warichu structure element does not consist of
// exactly the three-element sequence WP, WT, WP that ISO 14289-2 §8.2.5.24
// mandates (veraPDF UA2:8.2.5.24-1): the warichu text (WT) wrapped in a pair of
// warichu punctuation elements (WP). Any other child sequence is malformed.
//
// One finding per malformed Warichu. N/A when the document has no Warichu
// elements. PDF/UA-2 only.
type WarichuStructure struct{}

func (WarichuStructure) ID() string                { return "UA-01-018" }
func (WarichuStructure) Title() string             { return "Warichu elements have a valid child sequence" }
func (WarichuStructure) Category() engine.Category { return engine.CategoryStructure }
func (WarichuStructure) Severity() engine.Severity { return engine.SeverityError }
func (WarichuStructure) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (WarichuStructure) WCAG() []string            { return []string{"1.3.1"} }
func (WarichuStructure) Description() string {
	return "ISO 14289-2 §8.2.5.24 requires content typeset as warichu to be tagged as the three-element sequence WP, WT, WP grouped inside a Warichu element. A different child sequence separates the warichu text from its surrounding punctuation and breaks the intended reading."
}

func (c WarichuStructure) Run(doc model.Document) []engine.Finding {
	return checkChildSequence(doc, c.ID(), "Warichu",
		[]string{"WP&WT&WP"},
		"Use the child sequence WP, WT, WP inside the Warichu element.")
}

func init() { engine.Register(WarichuStructure{}) }

// checkChildSequence walks the structure tree and, for every element whose type
// is elemType, verifies that the ordered standard types of its direct children
// join to one of the allowed "A&B&C" patterns. Returns a single N/A finding
// when no element of that type exists.
func checkChildSequence(doc model.Document, checkID, elemType string, allowed []string, hint string) []engine.Finding {
	kids, hasRoot := doc.StructTreeRootKids()
	if !hasRoot {
		return []engine.Finding{{
			CheckID:  checkID,
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}
	roleMap := doc.RoleMap()
	allow := map[string]bool{}
	for _, a := range allowed {
		allow[a] = true
	}
	var findings []engine.Finding
	seen := 0
	var walk func(elem model.StructElement, path string)
	walk = func(elem model.StructElement, path string) {
		if elem.Type() == elemType {
			seen++
			var types []string
			for _, ch := range elem.Children() {
				types = append(types, resolveStandardType(ch.Type(), roleMap))
			}
			seq := strings.Join(types, "&")
			if !allow[seq] {
				got := seq
				if got == "" {
					got = "(no children)"
				}
				findings = append(findings, engine.Finding{
					CheckID:  checkID,
					Severity: engine.SeverityError,
					Message:  fmt.Sprintf("%s element at %s has invalid child sequence: %s", elemType, path, strings.ReplaceAll(got, "&", ", ")),
					Hint:     hint,
					Location: &engine.Location{Page: elem.Page(), StructPath: path},
				})
			}
		}
		for _, ch := range elem.Children() {
			walk(ch, path+"/"+ch.Type())
		}
	}
	for _, k := range kids {
		walk(k, "/"+k.Type())
	}
	if seen == 0 {
		return []engine.Finding{{
			CheckID:  checkID,
			Severity: engine.SeverityNotApplicable,
			Message:  fmt.Sprintf("document contains no %s elements -- nothing to inspect", elemType),
		}}
	}
	return findings
}

// resolveStandardType maps a raw structure type to its standard PDF type by
// following the classic /RoleMap chain (with a cycle guard). A type that is
// already standard maps to itself; a chain that never reaches a standard type
// yields its last link.
func resolveStandardType(raw string, roleMap map[string]string) string {
	seen := map[string]bool{}
	t := raw
	for {
		if model.IsStandardStructureType(t) {
			return t
		}
		if seen[t] {
			return t
		}
		seen[t] = true
		next, ok := roleMap[t]
		if !ok {
			return t
		}
		t = next
	}
}
