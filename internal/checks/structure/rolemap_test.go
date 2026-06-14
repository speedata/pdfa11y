package structure_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/structure"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestRoleMap(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantState   engine.Verdict
		wantMessage string
	}{
		{
			// tagged.pdf has only a Document StructElem -- all
			// standard types, no role-map needed.
			name:      "all-standard structure tree passes",
			fixture:   "testdata/tagged.pdf",
			wantState: engine.VerdictPass,
		},
		{
			// unmapped-custom-tag.pdf carries Document ->
			// MyCustomTag -> P, with no /RoleMap entry. The
			// MyCustomTag node must be flagged.
			name:        "unmapped custom tag fails",
			fixture:     "testdata/unmapped-custom-tag.pdf",
			wantState:   engine.VerdictFail,
			wantMessage: "MyCustomTag",
		},
		{
			// untagged.pdf has no structure tree at all -- the
			// check declines to apply.
			name:      "untagged document is N/A",
			fixture:   "testdata/untagged.pdf",
			wantState: engine.VerdictNA,
		},
	}

	check := structure.RoleMap{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}

			if got := result.State(); got != tc.wantState {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
			if tc.wantMessage != "" {
				if len(findings) == 0 {
					t.Fatalf("expected a finding mentioning %q, got none", tc.wantMessage)
				}
				if !strings.Contains(findings[0].Message, tc.wantMessage) {
					t.Errorf("message %q does not contain %q", findings[0].Message, tc.wantMessage)
				}
			}
		})
	}
}

// TestRoleMap_MathMLNamespacePasses ensures elements in the W3C
// MathML namespace (ISO 32000-2 §14.8.6.3) are not treated as
// custom PDF tags. A 'math' element with /NS pointing at the
// MathML URI must pass, even though "math" is not in the default
// PDF standard-type set.
func TestRoleMap_MathMLNamespacePasses(t *testing.T) {
	doc, err := pdf.LoadFile("../graphics/testdata/formula-math-ns-mathml.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	check := structure.RoleMap{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictPass {
		t.Errorf("State() = %v on a MathML-namespaced document, want PASS (findings: %+v)",
			result.State(), result.Findings)
	}
}

// TestRoleMap_MathTagInDefaultNSFails ensures that a 'math'-named
// element with no /NS at all (or a non-standard namespace) still
// fires the check -- only elements explicitly declared in one of
// the standard PDF / MathML / other registered namespaces escape
// the unmapped-custom-type rule. This catches producers that
// invent MathML-shaped tag names without the namespace machinery.
func TestRoleMap_MathTagInDefaultNSFails(t *testing.T) {
	doc, err := pdf.LoadFile("../graphics/testdata/formula-math-ns-none.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	check := structure.RoleMap{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictFail {
		t.Errorf("State() = %v on default-NS 'math' tag, want FAIL (findings: %+v)",
			result.State(), result.Findings)
	}
}

// TestRoleMap_MappedCustomTagPasses ensures that a document with a
// custom structure type WHICH IS declared in /RoleMap satisfies the
// check -- the role-map indirection is the whole point of the rule.
func TestRoleMap_MappedCustomTagPasses(t *testing.T) {
	// heading-rolemap-skip.pdf maps MyHeading -> H1 via /RoleMap.
	// Our model resolves Type() through the map, so the walk sees
	// "H1" (standard), not "MyHeading" (custom). Check passes.
	doc, err := pdf.LoadFile("../headings/testdata/heading-rolemap-skip.pdf")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	check := structure.RoleMap{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictPass {
		t.Errorf("State() = %v on a role-mapped document, want PASS (findings: %+v)",
			result.State(), result.Findings)
	}
}
