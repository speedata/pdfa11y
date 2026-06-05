package security_test

import (
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/security"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestPermissions(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantPass bool
	}{
		{"encryption permits accessibility passes", "testdata/encrypted-allow-access.pdf", true},
		{"encryption blocks accessibility fails", "testdata/encrypted-block-access.pdf", false},
	}

	check := security.Permissions{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := pdf.LoadFile(tc.fixture)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			findings := check.Run(doc)
			result := engine.Result{Check: check, Findings: findings}
			if got := result.Passed(); got != tc.wantPass {
				t.Fatalf("Passed() = %v, want %v (findings: %+v)", got, tc.wantPass, findings)
			}
		})
	}
}

// TestPermissions_Unencrypted asserts the check declines (N/A) on an
// unencrypted document -- there are no permissions to inspect.
func TestPermissions_Unencrypted(t *testing.T) {
	doc, err := pdf.LoadFile("../metadata/testdata/no-title.pdf")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	check := security.Permissions{}
	result := engine.Result{Check: check, Findings: check.Run(doc)}
	if result.State() != engine.VerdictNA {
		t.Errorf("State() = %v, want N/A", result.State())
	}
}
