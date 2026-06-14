package graphics

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/speedata/pdfa11y/internal/engine"
	"github.com/speedata/pdfa11y/internal/model"
)

// FormulaMathMLXML fails when a Formula's MathML Associated File
// stream is empty, not well-formed XML, has a root element other
// than <math>, or declares a namespace other than the W3C MathML
// namespace.
//
// PDF/UA-2 §8.2.5.29.1 inherits its MathML definition from PDF 2.0
// §14.8.6.3, which adopts the W3C MathML Core specification. The
// BPG "Math in PDF" repeats this: the embedded stream is expected
// to be MathML proper, not some other XML dialect.
//
// PDF/UA-2 only: gated via the shared isPDFUA2 helper. This is the
// content-level companion to MH-17-002 / MH-17-003 (filespec
// metadata level). It only inspects AFs whose Subtype the producer
// already declared as application/mathml+xml -- mis-declared AFs
// are MH-17-002's territory.
type FormulaMathMLXML struct{}

func (FormulaMathMLXML) ID() string                { return "MH-17-004" }
func (FormulaMathMLXML) Title() string             { return "MathML associated file is well-formed XML rooted at <math>" }
func (FormulaMathMLXML) Category() engine.Category { return engine.CategoryGraphics }
func (FormulaMathMLXML) Severity() engine.Severity { return engine.SeverityError }
func (FormulaMathMLXML) Spec() engine.Spec         { return engine.SpecPDFUA2 }
func (FormulaMathMLXML) WCAG() []string            { return []string{"1.1.1"} }
func (FormulaMathMLXML) Description() string {
	return "PDF/UA-2 §8.2.5.29.1 (and PDF 2.0 §14.8.6.3, which adopts W3C MathML Core) requires every MathML Associated File on a Formula to carry well-formed XML whose root element is <math> in the http://www.w3.org/1998/Math/MathML namespace. A stream declared as application/mathml+xml but containing empty data, malformed XML, a different root element, or the wrong namespace is not consumable by assistive technology."
}

func (c FormulaMathMLXML) Run(doc model.Document) []engine.Finding {
	if !isPDFUA2(doc) {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "check applies to PDF/UA-2 only; document does not declare pdfuaid:part = 2",
		}}
	}
	root, err := doc.StructTreeRoot()
	if err != nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityError,
			Message:  "cannot read structure tree: " + err.Error(),
		}}
	}
	if root == nil {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document has no structure tree -- nothing to inspect",
		}}
	}

	var findings []engine.Finding
	mathmlCount := 0
	walkFormulas(root, "/"+root.Type(), func(formula model.StructElement, path string) {
		for _, af := range formula.AssociatedFiles() {
			if af.Subtype != mathMLSubtype {
				continue
			}
			mathmlCount++
			if problem := validateMathMLStream(af.Content); problem != "" {
				findings = append(findings, engine.Finding{
					CheckID:  c.ID(),
					Severity: engine.SeverityError,
					Message: fmt.Sprintf(
						"Formula MathML AF %q: %s",
						displayFilename(af.Filename), problem),
					Hint:     "Ensure the embedded-file stream is well-formed XML whose root element is <math xmlns=\"http://www.w3.org/1998/Math/MathML\">. MathML Core (W3C CR) defines the permitted subset.",
					Location: &engine.Location{Page: formula.Page(), StructPath: path},
				})
			}
		}
	})
	if mathmlCount == 0 {
		return []engine.Finding{{
			CheckID:  c.ID(),
			Severity: engine.SeverityNotApplicable,
			Message:  "document contains no Formula with a MathML AF -- nothing to inspect",
		}}
	}
	return findings
}

// validateMathMLStream walks the decoded AF bytes as XML and
// returns "" when (a) the stream parses as well-formed XML to
// completion and (b) the first real element is <math> in the W3C
// MathML namespace. Otherwise returns a short human-readable
// description of what is wrong.
func validateMathMLStream(content []byte) string {
	if len(bytes.TrimSpace(content)) == 0 {
		return "embedded stream is empty"
	}
	dec := xml.NewDecoder(bytes.NewReader(content))
	rootSeen := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			if !rootSeen {
				return "XML contains no elements"
			}
			return ""
		}
		if err != nil {
			return "not well-formed XML: " + err.Error()
		}
		if rootSeen {
			// Keep draining tokens so the decoder reports any
			// downstream parse errors (truncated tags etc.).
			continue
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			// Skip comments, processing instructions, leading whitespace.
			continue
		}
		rootSeen = true
		if start.Name.Local != "math" {
			return fmt.Sprintf("root element is <%s>, expected <math>", start.Name.Local)
		}
		if start.Name.Space != mathMLNamespace {
			if start.Name.Space == "" {
				return "root <math> declares no namespace; expected " + mathMLNamespace
			}
			return fmt.Sprintf("root <math> declares namespace %q, expected %q",
				start.Name.Space, mathMLNamespace)
		}
	}
}

// mathMLNamespace is the W3C MathML namespace URI registered by
// ISO 32000-2 §14.8.6.3. Defined here (not in formula_alt.go's
// shared constants) because MH-17-004 is the first check that
// inspects content rather than metadata.
const mathMLNamespace = "http://www.w3.org/1998/Math/MathML"

func init() { engine.Register(FormulaMathMLXML{}) }
