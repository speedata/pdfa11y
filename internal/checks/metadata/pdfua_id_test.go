package metadata_test

import (
	"strings"
	"testing"

	"github.com/speedata/pdfa11y/internal/checks/metadata"
	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/pdf"
)

func TestPDFUAIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantPass    bool
		wantMessage string // substring expected in the failing finding's message
	}{
		{
			name:        "missing /Metadata stream fails",
			fixture:     "testdata/no-pdfua-id.pdf",
			wantPass:    false,
			wantMessage: "no /Metadata stream",
		},
		{
			name:     "XMP with pdfuaid:part passes",
			fixture:  "testdata/with-pdfua-id.pdf",
			wantPass: true,
		},
	}

	check := metadata.PDFUAIdentifier{}
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
