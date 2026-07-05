package annotations_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/annotations"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

// TestFormRoleChild covers UA-28-032: a Form structure element omitting a
// PrintField /Role attribute must have exactly one child, an object reference
// to the widget annotation (ISO 14289-1 §7.18.4; veraPDF UA1:7.18.4-2).
func TestFormRoleChild(t *testing.T) {
	cases := []struct {
		fixture string
		want    engine.Verdict
	}{
		{"testdata/form-one-widget.pdf", engine.VerdictPass},    // no Role, one widget ref
		{"testdata/form-role-attr.pdf", engine.VerdictPass},     // Role present -> exempt
		{"testdata/form-two-widgets.pdf", engine.VerdictFail},   // no Role, two widget refs
		{"testdata/form-no-widget-ref.pdf", engine.VerdictFail}, // no Role, no widget ref
		{"testdata/form-role-na.pdf", engine.VerdictNA},         // no Form element
	}
	check := annotations.FormRoleChild{}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			result := engine.Result{Check: check, Findings: check.Run(doc)}
			if got := result.State(); got != tc.want {
				t.Fatalf("State() = %v, want %v (findings: %+v)", got, tc.want, result.Findings)
			}
		})
	}
}
