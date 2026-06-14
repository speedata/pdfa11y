package graphics_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/graphics"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestFormulaMathMLSubtype(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantState    engine.Verdict
		wantFindings int
	}{
		{
			"MathML AF with correct subtype passes",
			"testdata/formula-mathml-af.pdf",
			engine.VerdictPass,
			0,
		},
		{
			"Supplement AF with text/html subtype fails",
			"testdata/formula-mathml-wrong-subtype.pdf",
			engine.VerdictFail,
			1,
		},
		{
			"Supplement AF with correct subtype but wrong relationship is N/A here",
			// the AF is /Source, so no /Supplement to inspect -- MH-17-002
			// is N/A; MH-17-003 catches it instead.
			"testdata/formula-mathml-wrong-relationship.pdf",
			engine.VerdictNA,
			1,
		},
		{
			"PDF/UA-1 document is N/A",
			"testdata/formula-with-alt.pdf",
			engine.VerdictNA,
			1,
		},
	}
	check := graphics.FormulaMathMLSubtype{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantState {
				t.Errorf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Errorf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
		})
	}
}

func TestFormulaMathMLRelationship(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantState    engine.Verdict
		wantFindings int
	}{
		{
			"MathML AF with correct relationship passes",
			"testdata/formula-mathml-af.pdf",
			engine.VerdictPass,
			0,
		},
		{
			"MathML AF with /Source relationship fails",
			"testdata/formula-mathml-wrong-relationship.pdf",
			engine.VerdictFail,
			1,
		},
		{
			"AF with /Supplement but wrong subtype is N/A here",
			// no AF carries the application/mathml+xml subtype, so this
			// check has nothing to inspect; MH-17-002 catches it instead.
			"testdata/formula-mathml-wrong-subtype.pdf",
			engine.VerdictNA,
			1,
		},
		{
			"PDF/UA-1 document is N/A",
			"testdata/formula-with-alt.pdf",
			engine.VerdictNA,
			1,
		},
	}
	check := graphics.FormulaMathMLRelationship{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.State(); got != tc.wantState {
				t.Errorf("State() = %v, want %v (findings: %+v)", got, tc.wantState, findings)
			}
			if len(findings) != tc.wantFindings {
				t.Errorf("findings = %d, want %d (%+v)", len(findings), tc.wantFindings, findings)
			}
		})
	}
}
