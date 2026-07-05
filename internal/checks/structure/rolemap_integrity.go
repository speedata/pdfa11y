package structure

import (
	"fmt"
	"strings"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// RoleMapIntegrity fails on two role-map defects that PDF/UA forbids:
//
//   - a circular mapping (PDF/UA-1 §7.1 / PDF/UA-2 §8.2.4-2): following the
//     /RoleMap (or a namespace /RoleMapNS) chain returns to a type already
//     visited, so role resolution never terminates at a standard type; and
//   - a same-namespace mapping (PDF/UA-2 §8.2.4-3): a structure type in an
//     explicitly declared namespace is role-mapped -- directly or through a
//     chain -- back to a type in that same namespace, which is not permitted
//     (ISO 32000-2 §14.8.6).
//
// It reasons over the classic /RoleMap (namespace-agnostic type->type edges)
// and the PDF 2.0 /RoleMapNS maps (namespace-qualified edges). One finding per
// distinct defect. N/A when the document declares no role mappings.
type RoleMapIntegrity struct{}

func (RoleMapIntegrity) ID() string                { return "UA-31-009" }
func (RoleMapIntegrity) Title() string             { return "Role maps are acyclic and do not remap within a namespace" }
func (RoleMapIntegrity) Category() engine.Category { return engine.CategoryStructure }
func (RoleMapIntegrity) Severity() engine.Severity { return engine.SeverityError }
func (RoleMapIntegrity) Spec() engine.Spec         { return engine.SpecBoth }
func (RoleMapIntegrity) WCAG() []string            { return []string{"1.3.1"} }
func (RoleMapIntegrity) Description() string {
	return "PDF/UA-1 §7.1 and PDF/UA-2 §8.2.4 require role mappings to terminate: a circular /RoleMap or /RoleMapNS chain never resolves to a standard type. PDF/UA-2 §8.2.4-3 additionally forbids role-mapping a type to another type in the same explicitly declared namespace."
}

type nsNode struct{ ns, typ string }

func (c RoleMapIntegrity) Run(doc model.Document) []engine.Finding {
	classic := doc.RoleMap()
	namespaces := doc.Namespaces()

	// Build the namespace-qualified edge set (first target per source).
	nsEdges := map[nsNode]nsNode{}
	nsMappings := 0
	for _, ns := range namespaces {
		for src, targets := range ns.RoleMapNS {
			if len(targets) == 0 {
				continue
			}
			nsEdges[nsNode{ns.URI, src}] = nsNode{targets[0].NamespaceURI, targets[0].Type}
			nsMappings++
		}
	}

	if len(classic) == 0 && nsMappings == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document declares no role mappings -- nothing to inspect",
		}}
	}

	var findings []engine.Finding

	// Classic /RoleMap: functional graph, report a representative cycle.
	if chain, ok := classicCycle(classic); ok {
		findings = append(findings, engine.Finding{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "circular /RoleMap: " + chain,
			Hint:     "Break the cycle so every custom type eventually maps to a standard structure type.",
		})
	}

	// Namespace /RoleMapNS: cycle, or resolution back into the start namespace.
	for start := range nsEdges {
		if f, ok := c.inspectNS(start, nsEdges); ok {
			findings = append(findings, f)
		}
	}
	return findings
}

// inspectNS follows the role-map chain from start. It reports a same-namespace
// mapping when the chain reaches the start namespace again, or a cycle when it
// revisits a node.
func (c RoleMapIntegrity) inspectNS(start nsNode, edges map[nsNode]nsNode) (engine.Finding, bool) {
	visited := map[nsNode]bool{start: true}
	n := start
	for {
		next, ok := edges[n]
		if !ok {
			return engine.Finding{}, false
		}
		if next.ns == start.ns {
			return engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("role mapping stays within namespace %q: %s is mapped to %s in the same namespace", start.ns, start.typ, next.typ),
				Hint:     "Map the type to a standard type in a different (standard) namespace, or leave it unmapped.",
			}, true
		}
		if visited[next] {
			return engine.Finding{
				CheckID:  c.ID(),
				Severity: engine.SeverityError,
				Message:  fmt.Sprintf("circular /RoleMapNS starting at %s (namespace %q)", start.typ, start.ns),
				Hint:     "Break the cycle so the mapping resolves to a standard type.",
			}, true
		}
		visited[next] = true
		n = next
	}
}

// classicCycle follows the functional /RoleMap graph and returns a rendered
// chain if any mapping cycles.
func classicCycle(rm map[string]string) (string, bool) {
	for start := range rm {
		visited := map[string]bool{}
		order := []string{}
		n := start
		for {
			visited[n] = true
			order = append(order, n)
			next, ok := rm[n]
			if !ok {
				break
			}
			if visited[next] {
				order = append(order, next)
				return renderChain(order, next), true
			}
			n = next
		}
	}
	return "", false
}

// renderChain renders a walk "a -> b -> c" but only from the first occurrence
// of the repeated node, so the reported cycle is tight.
func renderChain(order []string, repeat string) string {
	start := 0
	for i, v := range order {
		if v == repeat {
			start = i
			break
		}
	}
	return strings.Join(order[start:], " -> ")
}

func init() { engine.Register(RoleMapIntegrity{}) }
