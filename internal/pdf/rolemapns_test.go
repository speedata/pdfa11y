package pdf_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/speedata/pdfa11y/internal/pdf"
)

// buildPDF assembles a PDF from numbered object bodies (object i+1 is
// objs[i]) with a correct xref table. Object 1 must be the Catalog.
func buildPDF(objs []string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offsets := make([]int, len(objs))
	for i, body := range objs {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return buf.Bytes()
}

// TestTypeResolvesRoleMapNS checks that StructElement.Type() follows a
// namespace's /RoleMapNS (ISO 32000-2 §14.8.6.2) to a type in a
// standard namespace, and leaves the raw type alone where no such
// resolution exists.
func TestTypeResolvesRoleMapNS(t *testing.T) {
	elem := func(s, ns string) string {
		if ns != "" {
			ns = " /NS " + ns + " 0 R"
		}
		return fmt.Sprintf("<< /Type /StructElem /S /%s /P 4 0 R%s >>", s, ns)
	}
	cases := []struct {
		s, ns string // /S and namespace object number ("" for none)
		want  string
	}{
		{"li", "6", "LI"},     // direct mapping into the PDF 2.0 namespace
		{"item", "6", "LI"},   // item -> li (same namespace) -> LI
		{"entry", "8", "LI"},  // urn:custom entry -> XHTML item -> li -> LI
		{"a", "6", "a"},       // cycle a -> b -> a: unresolved
		{"span", "6", "span"}, // no mapping in the namespace
		{"box", "8", "box"},   // chain ends in a non-standard namespace
		{"mi", "7", "mi"},     // MathML types are standard as they are
		{"P", "5", "P"},       // PDF 2.0 type, exempt from /RoleMap
		{"Para", "", "Div"},   // no namespace: classic /RoleMap
	}

	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
		"", // 4: StructTreeRoot, filled in below
		"<< /Type /Namespace /NS (http://iso.org/pdf2/ssn) >>",
		"<< /Type /Namespace /NS (http://www.w3.org/1999/xhtml) /RoleMapNS << /li [/LI 5 0 R] /item /li /a /b /b /a >> >>",
		"<< /Type /Namespace /NS (http://www.w3.org/1998/Math/MathML) >>",
		"<< /Type /Namespace /NS (urn:custom) /RoleMapNS << /entry [/item 6 0 R] /box [/x 6 0 R] >> >>",
	}
	kids := ""
	for _, c := range cases {
		objs = append(objs, elem(c.s, c.ns))
		kids += fmt.Sprintf(" %d 0 R", len(objs))
	}
	objs[3] = "<< /Type /StructTreeRoot /K [" + kids + " ] /Namespaces [5 0 R 6 0 R 7 0 R 8 0 R] /RoleMap << /Para /Div /P /Span >> >>"

	doc, err := pdf.Load(bytes.NewReader(buildPDF(objs)))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, ok := doc.StructTreeRootKids()
	if !ok || len(got) != len(cases) {
		t.Fatalf("got %d root kids (ok=%v), want %d", len(got), ok, len(cases))
	}
	for i, c := range cases {
		if typ := got[i].Type(); typ != c.want {
			t.Errorf("/S /%s in namespace obj %q: Type() = %q, want %q", c.s, c.ns, typ, c.want)
		}
	}
}

// TestTypeClassicRoleMapStopsAtStandard checks that the classic /RoleMap
// chain ends at the first standard structure type (ISO 32000-2 §14.7.3
// NOTE 2) instead of following a further mapping of that standard type.
func TestTypeClassicRoleMapStopsAtStandard(t *testing.T) {
	cases := []struct{ s, want string }{
		{"Para", "P"},         // Para -> P; the P -> Span entry is not followed
		{"DocTitle", "Title"}, // DocTitle -> Title; Title -> P is not followed
		{"Box", "Div"},        // Box -> Frame -> Div: custom links are followed
		{"A", "A"},            // cycle A -> B -> A: unresolved
	}
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
		"", // 4: StructTreeRoot, filled in below
	}
	kids := ""
	for _, c := range cases {
		objs = append(objs, fmt.Sprintf("<< /Type /StructElem /S /%s /P 4 0 R >>", c.s))
		kids += fmt.Sprintf(" %d 0 R", len(objs))
	}
	objs[3] = "<< /Type /StructTreeRoot /K [" + kids + " ] /RoleMap << /Para /P /P /Span /DocTitle /Title /Title /P /Box /Frame /Frame /Div /A /B /B /A >> >>"

	doc, err := pdf.Load(bytes.NewReader(buildPDF(objs)))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, ok := doc.StructTreeRootKids()
	if !ok || len(got) != len(cases) {
		t.Fatalf("got %d root kids (ok=%v), want %d", len(got), ok, len(cases))
	}
	for i, c := range cases {
		if typ := got[i].Type(); typ != c.want {
			t.Errorf("/S /%s: Type() = %q, want %q", c.s, typ, c.want)
		}
	}
}
