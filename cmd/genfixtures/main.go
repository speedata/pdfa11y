// genfixtures regenerates all check fixtures from the canonical base PDF.
// Run from the repository root: go run ./cmd/genfixtures
//
// Every fixture under internal/checks/*/testdata/ is either an exact copy of
// the base or a programmatic derivative. Keeping the generator in source
// means new contributors can rebuild fixtures with one command instead of
// guessing how each PDF was produced. Output files are intended to be
// committed -- pdfcpu's writer is not byte-deterministic across runs, so do
// not run this in test setup (it would produce noisy diffs on every test
// run).
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// basePath is the canonical fixture: an untagged PDF with no /Title, no
// /Lang, no /Metadata stream and no /ViewerPreferences. Every other
// fixture is derived from it.
const basePath = "internal/checks/metadata/testdata/no-title.pdf"

const xmpUA1 = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>1</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

// xmpUA1FullID is xmpUA1 plus a pdfuaid:rev so the PDF/UA
// identification schema is complete per ISO 14289-2 §5. Used by
// the UA-06-006 passing fixture; xmpUA1 keeps the part-only shape
// used by UA-06-006's failing fixture and unchanged downstream
// callers.
const xmpUA1FullID = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>1</pdfuaid:part>
      <pdfuaid:rev>2014</pdfuaid:rev>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

const xmpUA1WithDCTitle = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/"
        xmlns:dc="http://purl.org/dc/elements/1.1/">
      <pdfuaid:part>1</pdfuaid:part>
      <dc:title>
        <rdf:Alt>
          <rdf:li xml:lang="x-default">Sample</rdf:li>
        </rdf:Alt>
      </dc:title>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

func main() {
	if err := chdirRepoRoot(); err != nil {
		log.Fatal(err)
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// (Re)write the canonical base PDF before deriving anything from it.
	// The base is a hand-emitted blank one-page document so the
	// repository carries no third-party copyrighted material.
	if err := writeBlankPDF(basePath); err != nil {
		return fmt.Errorf("write base PDF: %w", err)
	}
	fmt.Println("wrote", basePath, "(blank base)")

	// Pure copies of the base PDF, used as failure fixtures for various
	// checks. The base itself stays where it is and is not rewritten.
	for _, dst := range []string{
		"internal/checks/structure/testdata/untagged.pdf",
		"internal/checks/metadata/testdata/no-pdfua-id.pdf",
		"internal/checks/language/testdata/no-lang.pdf",
		"internal/checks/viewerprefs/testdata/no-viewerprefs.pdf",
	} {
		if err := copyBase(dst); err != nil {
			return err
		}
	}

	if err := withInfoProperty("internal/checks/metadata/testdata/with-title.pdf",
		"Title", "Sample"); err != nil {
		return err
	}
	if err := withCatalogString("internal/checks/language/testdata/with-lang.pdf",
		"Lang", "en-US"); err != nil {
		return err
	}
	// UA-11-003: Catalog /Lang with a malformed tag (9-letter primary subtag).
	if err := withCatalogString("internal/checks/language/testdata/lang-format-invalid.pdf",
		"Lang", "portugues-pt"); err != nil {
		return err
	}
	if err := withMetadataStream("internal/checks/metadata/testdata/with-pdfua-id.pdf",
		xmpUA1); err != nil {
		return err
	}
	if err := withMetadataStream("internal/checks/metadata/testdata/with-pdfua-id-rev.pdf",
		xmpUA1FullID); err != nil {
		return err
	}
	if err := withMetadataStream("internal/checks/metadata/testdata/with-xmp-title.pdf",
		xmpUA1WithDCTitle); err != nil {
		return err
	}
	if err := withDisplayDocTitle("internal/checks/viewerprefs/testdata/displaydoctitle-true.pdf",
		true); err != nil {
		return err
	}
	if err := withDisplayDocTitle("internal/checks/viewerprefs/testdata/displaydoctitle-false.pdf",
		false); err != nil {
		return err
	}
	if err := withStructTree("internal/checks/structure/testdata/tagged.pdf"); err != nil {
		return err
	}
	// UA-01-015 (§8.2.5.2-1): StructTreeRoot must have a single Document as
	// its only child. Fail shape: two top-level Document elements.
	if err := withRootChildren("internal/checks/structure/testdata/root-multi-document.pdf",
		"Document", "Document"); err != nil {
		return err
	}
	// UA-01-014 (§8.2.1-2): every structure element carries /P. Fail shape:
	// a P element below Document that omits its own /P.
	if err := withMissingParent("internal/checks/structure/testdata/struct-missing-parent.pdf"); err != nil {
		return err
	}
	if err := withUntaggedContent("internal/checks/structure/testdata/untagged-content.pdf"); err != nil {
		return err
	}
	// UA-13-004: Figure has /Alt or /ActualText. Same UA-1 §7.3
	// asymmetry as UA-17-001 -- empty /Alt fails, empty /ActualText
	// passes (decorative-figure idiom).
	if err := withFigure("internal/checks/graphics/testdata/figure-with-alt.pdf",
		figureSpec{alt: strPtr("Sunset over the mountains")}); err != nil {
		return err
	}
	if err := withFigure("internal/checks/graphics/testdata/figure-no-alt.pdf",
		figureSpec{}); err != nil {
		return err
	}
	// UA-11-004: Alt/ActualText/E need a determinable language. A Figure with
	// /Alt and its own /Lang passes.
	if err := withFigure("internal/checks/language/testdata/figure-alt-lang.pdf",
		figureSpec{alt: strPtr("Sunset over the mountains"), lang: strPtr("en-US")}); err != nil {
		return err
	}
	if err := withFigure("internal/checks/graphics/testdata/figure-alt-empty.pdf",
		figureSpec{alt: strPtr("")}); err != nil {
		return err
	}
	if err := withFigure("internal/checks/graphics/testdata/figure-actualtext.pdf",
		figureSpec{actualText: strPtr("Decorative bullet")}); err != nil {
		return err
	}
	if err := withFigure("internal/checks/graphics/testdata/figure-actualtext-empty.pdf",
		figureSpec{actualText: strPtr("")}); err != nil {
		return err
	}
	if err := withHeadings("internal/checks/headings/testdata/heading-ok.pdf",
		[]int{1, 2, 3}); err != nil {
		return err
	}
	if err := withHeadings("internal/checks/headings/testdata/heading-skip.pdf",
		[]int{1, 3}); err != nil {
		return err
	}
	if err := withRoleMappedHeading("internal/checks/headings/testdata/heading-rolemap-skip.pdf"); err != nil {
		return err
	}
	if err := withHeadings("internal/checks/headings/testdata/heading-not-h1.pdf",
		[]int{3, 4}); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/headings/testdata/heading-two-h.pdf",
		"Sect", "H", "H"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/headings/testdata/heading-single-h.pdf",
		"Sect", "H", "P"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/structure/testdata/ruby-ok.pdf",
		"Ruby", "RB", "RT"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/structure/testdata/ruby-bad.pdf",
		"Ruby", "RB"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/structure/testdata/warichu-ok.pdf",
		"Warichu", "WP", "WT", "WP"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/structure/testdata/warichu-bad.pdf",
		"Warichu", "WT", "WP"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/tables/testdata/table-ok.pdf",
		"Table", "TR"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/tables/testdata/table-no-tr.pdf",
		"Table", "P"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/lists/testdata/list-ok.pdf",
		"L", "LI"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/lists/testdata/list-no-li.pdf",
		"L", "P"); err != nil {
		return err
	}
	if err := withParentChildren("internal/checks/structure/testdata/unmapped-custom-tag.pdf",
		"MyCustomTag", "P"); err != nil {
		return err
	}
	if err := withFont("internal/checks/fonts/testdata/font-ok.pdf",
		"PDFA11YTestOK", true, true); err != nil {
		return err
	}
	if err := withFont("internal/checks/fonts/testdata/font-not-embedded.pdf",
		"PDFA11YTestNoEmbed", false, true); err != nil {
		return err
	}
	if err := withFont("internal/checks/fonts/testdata/font-no-tounicode.pdf",
		"PDFA11YTestNoToU", true, false); err != nil {
		return err
	}
	// UA-10-003: a /ToUnicode value of U+0000 is forbidden. Map code <20>
	// to <0000>.
	if err := withFontCMap("internal/checks/fonts/testdata/font-tounicode-null.pdf",
		"PDFA11YTestToUNull", true, true,
		"/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n"+
			"1 begincodespacerange\n<00> <FF>\nendcodespacerange\n"+
			"1 beginbfchar\n<20> <0000>\nendbfchar\nendcmap\nend\nend\n"); err != nil {
		return err
	}

	// Annotation fixtures: one passing + one failing pair per UA-28 check.
	if err := withLinkAnnotation("internal/checks/annotations/testdata/link-with-contents.pdf",
		"Project documentation"); err != nil {
		return err
	}
	if err := withLinkAnnotation("internal/checks/annotations/testdata/link-no-contents.pdf",
		""); err != nil {
		return err
	}
	// UA-28-001: in PDF/UA-2 a missing /Contents is only a
	// recommendation, so the same shape is reported as an advisory
	// (Info) rather than an error.
	if err := withLinkAnnotationUA2("internal/checks/annotations/testdata/link-no-contents-ua2.pdf",
		""); err != nil {
		return err
	}
	if err := withWidgetAnnotation("internal/checks/annotations/testdata/widget-with-tu.pdf",
		"First name"); err != nil {
		return err
	}
	if err := withWidgetAnnotation("internal/checks/annotations/testdata/widget-no-tu.pdf",
		""); err != nil {
		return err
	}
	if err := withTextAnnotation("internal/checks/annotations/testdata/annot-with-structparent.pdf",
		true); err != nil {
		return err
	}
	if err := withTextAnnotation("internal/checks/annotations/testdata/annot-no-structparent.pdf",
		false); err != nil {
		return err
	}
	if err := withArtifactAnnotation("internal/checks/annotations/testdata/watermark-no-structparent.pdf",
		"Watermark", false); err != nil {
		return err
	}
	if err := withArtifactAnnotation("internal/checks/annotations/testdata/watermark-with-structparent.pdf",
		"Watermark", true); err != nil {
		return err
	}
	// UA-28-012: a visible TrapNet annotation is forbidden (PDF/UA-1 §7.18.2).
	if err := withArtifactAnnotation("internal/checks/annotations/testdata/trapnet-visible.pdf",
		"TrapNet", false); err != nil {
		return err
	}
	// UA-28-013: a Link annotation must be enclosed in a Link element.
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/link-tagged.pdf",
		"Link", "Link"); err != nil {
		return err
	}
	// UA-28-014: a Widget annotation must be nested within a Form element.
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/widget-in-form.pdf",
		"Form", "Widget"); err != nil {
		return err
	}
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/widget-in-p.pdf",
		"P", "Widget"); err != nil {
		return err
	}
	// UA-28-027: a Form element with one widget passes; two widgets fail.
	if err := withFormWidgets("internal/checks/annotations/testdata/form-one-widget.pdf",
		1); err != nil {
		return err
	}
	if err := withFormWidgets("internal/checks/annotations/testdata/form-two-widgets.pdf",
		2); err != nil {
		return err
	}
	// UA-28-032: a Form omitting a Role attribute must have exactly one child,
	// an object reference to the widget. Role present -> pass regardless of
	// children; no Role and no widget reference -> fail; a non-Form element ->
	// N/A. (The one-widget pass case reuses form-one-widget.pdf above.)
	if err := withFormRole("internal/checks/annotations/testdata/form-role-attr.pdf",
		"Form", "rb", 1); err != nil {
		return err
	}
	if err := withFormRole("internal/checks/annotations/testdata/form-no-widget-ref.pdf",
		"Form", "", 0); err != nil {
		return err
	}
	if err := withFormRole("internal/checks/annotations/testdata/form-role-na.pdf",
		"P", "", 0); err != nil {
		return err
	}
	// UA-28-033: a text field with /RV must also carry a textually equivalent
	// /V. Missing /V fails; a divergent /V fails; a matching /V passes; a text
	// field without /RV is N/A.
	rvHTML := `<body xmlns="http://www.w3.org/1999/xhtml"><p>Hello</p></body>`
	if err := withTextFieldRV("internal/checks/annotations/testdata/rv-no-value.pdf",
		rvHTML, ""); err != nil {
		return err
	}
	if err := withTextFieldRV("internal/checks/annotations/testdata/rv-value-mismatch.pdf",
		rvHTML, "Goodbye"); err != nil {
		return err
	}
	if err := withTextFieldRV("internal/checks/annotations/testdata/rv-value-match.pdf",
		rvHTML, "Hello"); err != nil {
		return err
	}
	if err := withTextFieldRV("internal/checks/annotations/testdata/rv-absent.pdf",
		"", "Hello"); err != nil {
		return err
	}
	// UA-01-020: real content mapping to Unicode PUA needs /ActualText or
	// /Alt. PUA without either fails; a content /ActualText or a structure
	// /Alt passes; non-PUA content is N/A.
	if err := withPUAContent("internal/checks/structure/testdata/pua-content-no-alt.pdf",
		true, false, false); err != nil {
		return err
	}
	if err := withPUAContent("internal/checks/structure/testdata/pua-content-actualtext.pdf",
		true, true, false); err != nil {
		return err
	}
	if err := withPUAContent("internal/checks/structure/testdata/pua-content-struct-alt.pdf",
		true, false, true); err != nil {
		return err
	}
	if err := withPUAContent("internal/checks/structure/testdata/pua-content-nonpua.pdf",
		false, false, false); err != nil {
		return err
	}
	// UA-11-009: a form field /TU needs a determinable language. No /Lang
	// anywhere fails; a structure-element /Lang or catalog /Lang passes; a
	// field without /TU is N/A.
	if err := withFieldTU("internal/checks/language/testdata/tu-no-lang.pdf",
		true, false, false); err != nil {
		return err
	}
	if err := withFieldTU("internal/checks/language/testdata/tu-struct-lang.pdf",
		true, true, false); err != nil {
		return err
	}
	if err := withFieldTU("internal/checks/language/testdata/tu-catalog-lang.pdf",
		true, false, true); err != nil {
		return err
	}
	if err := withFieldTU("internal/checks/language/testdata/tu-absent.pdf",
		false, false, false); err != nil {
		return err
	}
	// UA-28-036: a signature appearance graphic (Figure) needs /Alt. A
	// signature Figure without /Alt fails; with /Alt passes; no Figure is N/A.
	if err := withSignatureGraphic("internal/checks/annotations/testdata/sig-figure-no-alt.pdf",
		true, true, false); err != nil {
		return err
	}
	if err := withSignatureGraphic("internal/checks/annotations/testdata/sig-figure-with-alt.pdf",
		true, true, true); err != nil {
		return err
	}
	if err := withSignatureGraphic("internal/checks/annotations/testdata/sig-no-figure.pdf",
		true, false, false); err != nil {
		return err
	}
	// UA-28-035: a field widget with neither a Lbl label nor /Contents fails;
	// a Lbl child passes; /Contents passes.
	if err := withFormLabelWidget("internal/checks/annotations/testdata/widget-no-lbl-no-contents.pdf",
		false, false); err != nil {
		return err
	}
	if err := withFormLabelWidget("internal/checks/annotations/testdata/widget-with-lbl.pdf",
		true, false); err != nil {
		return err
	}
	if err := withFormLabelWidget("internal/checks/annotations/testdata/widget-with-contents.pdf",
		false, true); err != nil {
		return err
	}
	// UA-28-034: a markup annotation with both /RC and /Contents must have
	// them textually equivalent. Divergent fails; equal passes; /RC without
	// /Contents passes; neither is N/A.
	if err := withMarkupRC("internal/checks/annotations/testdata/rc-mismatch.pdf",
		rvHTML, "Goodbye"); err != nil {
		return err
	}
	if err := withMarkupRC("internal/checks/annotations/testdata/rc-match.pdf",
		rvHTML, "Hello"); err != nil {
		return err
	}
	if err := withMarkupRC("internal/checks/annotations/testdata/rc-no-contents.pdf",
		rvHTML, ""); err != nil {
		return err
	}
	// UA-10-004: a Type0/Identity font showing code 0 references the .notdef
	// glyph (CID 0 / glyph 0). Code 0 fails; a real code passes; a
	// CIDToGIDMap remapping stream makes glyph 0 unconfirmable, so it passes.
	if err := withType0Notdef("internal/checks/fonts/testdata/notdef-code0.pdf",
		0, false); err != nil {
		return err
	}
	if err := withType0Notdef("internal/checks/fonts/testdata/notdef-realglyph.pdf",
		1, false); err != nil {
		return err
	}
	if err := withType0Notdef("internal/checks/fonts/testdata/notdef-cidgidmap-stream.pdf",
		0, true); err != nil {
		return err
	}
	// UA-01-017: a human-readable text string must not use Unicode PUA. A
	// form-field /T with a PUA character fails; a clean /T passes.
	if err := withWidgetName("internal/checks/structure/testdata/puatext-field-pua.pdf",
		"Field", true); err != nil {
		return err
	}
	if err := withWidgetName("internal/checks/structure/testdata/puatext-field-clean.pdf",
		"Field", false); err != nil {
		return err
	}
	// UA-11-008: a Span with /ActualText, /Alt or /E must have a determinable
	// language (Span /Lang, inherited struct /Lang, or catalog /Lang).
	if err := withSpanLang("internal/checks/language/testdata/span-actualtext-no-lang.pdf",
		"ActualText", "none"); err != nil {
		return err
	}
	if err := withSpanLang("internal/checks/language/testdata/span-alt-no-lang.pdf",
		"Alt", "none"); err != nil {
		return err
	}
	if err := withSpanLang("internal/checks/language/testdata/span-e-no-lang.pdf",
		"E", "none"); err != nil {
		return err
	}
	if err := withSpanLang("internal/checks/language/testdata/span-span-lang.pdf",
		"ActualText", "span"); err != nil {
		return err
	}
	if err := withSpanLang("internal/checks/language/testdata/span-catalog-lang.pdf",
		"Alt", "catalog"); err != nil {
		return err
	}
	if err := withSpanLang("internal/checks/language/testdata/span-struct-lang.pdf",
		"ActualText", "struct"); err != nil {
		return err
	}
	// UA-14-010: Artifact and structure-tagged marked content must not nest
	// inside one another (ISO 14289-1 §7.1). Either nesting direction fails;
	// keeping them as siblings passes.
	if err := withMCNesting("internal/checks/structure/testdata/mc-artifact-in-tagged.pdf",
		"artifact-in-tagged"); err != nil {
		return err
	}
	if err := withMCNesting("internal/checks/structure/testdata/mc-tagged-in-artifact.pdf",
		"tagged-in-artifact"); err != nil {
		return err
	}
	if err := withMCNesting("internal/checks/structure/testdata/mc-nesting-ok.pdf",
		"siblings"); err != nil {
		return err
	}
	// UA-30-001: a form XObject with /Ref (reference XObject) fails; a plain
	// form XObject passes.
	if err := withFormXObject("internal/checks/xobjects/testdata/reference-xobject.pdf",
		true); err != nil {
		return err
	}
	if err := withFormXObject("internal/checks/xobjects/testdata/plain-xobject.pdf",
		false); err != nil {
		return err
	}
	// UA-30-002: a marked-content form XObject drawn once passes; drawn twice
	// fails (its MCID has no unique semantic parent). A plain XObject drawn
	// twice passes -- multiple invocations are only a problem for marked content.
	if err := withMarkedFormXObject("internal/checks/xobjects/testdata/xobj-mc-once.pdf",
		true, 1); err != nil {
		return err
	}
	if err := withMarkedFormXObject("internal/checks/xobjects/testdata/xobj-mc-twice.pdf",
		true, 2); err != nil {
		return err
	}
	if err := withMarkedFormXObject("internal/checks/xobjects/testdata/xobj-plain-twice.pdf",
		false, 2); err != nil {
		return err
	}
	// UA-28-028: a dynamic XFA form fails; a static XFA form passes.
	if err := withXFA("internal/checks/annotations/testdata/xfa-dynamic.pdf",
		true); err != nil {
		return err
	}
	if err := withXFA("internal/checks/annotations/testdata/xfa-static.pdf",
		false); err != nil {
		return err
	}
	// UA-28-029: a field widget with /AA but no /Contents fails; with
	// /Contents it passes.
	if err := withWidgetAA("internal/checks/annotations/testdata/widget-aa-no-contents.pdf",
		false); err != nil {
		return err
	}
	if err := withWidgetAA("internal/checks/annotations/testdata/widget-aa-contents.pdf",
		true); err != nil {
		return err
	}
	// UA-28-030: one Link element enclosing links to different targets
	// fails; enclosing links to the same target passes.
	if err := withLinkTargets("internal/checks/annotations/testdata/link-different-targets.pdf",
		false); err != nil {
		return err
	}
	if err := withLinkTargets("internal/checks/annotations/testdata/link-same-target.pdf",
		true); err != nil {
		return err
	}
	// UA-28-031: annotation /Contents differing from the enclosing element's
	// /Alt fails; identical passes; /Alt without /Contents is N/A.
	if err := withAnnotAltContents("internal/checks/annotations/testdata/annot-alt-contents-differ.pdf",
		"Alt text", "Different contents"); err != nil {
		return err
	}
	if err := withAnnotAltContents("internal/checks/annotations/testdata/annot-alt-contents-match.pdf",
		"Same text", "Same text"); err != nil {
		return err
	}
	if err := withAnnotAltContents("internal/checks/annotations/testdata/annot-alt-no-contents.pdf",
		"Alt text", ""); err != nil {
		return err
	}
	// UA-11-006: bookmarks without a catalog /Lang fail; with /Lang pass;
	// no bookmarks is N/A.
	if err := withOutlineLang("internal/checks/language/testdata/outline-no-lang.pdf",
		true, false); err != nil {
		return err
	}
	if err := withOutlineLang("internal/checks/language/testdata/outline-with-lang.pdf",
		true, true); err != nil {
		return err
	}
	if err := withOutlineLang("internal/checks/language/testdata/no-outline.pdf",
		false, false); err != nil {
		return err
	}
	// UA-11-007: XMP x-default metadata without a catalog /Lang fails; with
	// /Lang passes; no x-default is N/A.
	if err := withMetadataLang("internal/checks/language/testdata/metadata-no-lang.pdf",
		true, false); err != nil {
		return err
	}
	if err := withMetadataLang("internal/checks/language/testdata/metadata-with-lang.pdf",
		true, true); err != nil {
		return err
	}
	if err := withMetadataLang("internal/checks/language/testdata/metadata-no-xdefault.pdf",
		false, false); err != nil {
		return err
	}
	// UA-28-015: a markup annotation must be enclosed in an Annot element.
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/markup-in-annot.pdf",
		"Annot", "Highlight"); err != nil {
		return err
	}
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/markup-in-p.pdf",
		"P", "Highlight"); err != nil {
		return err
	}
	// UA-28-016: a PDF/UA-2 Link annotation must be in a Link or Reference element.
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/link-in-reference.pdf",
		"Reference", "Link"); err != nil {
		return err
	}
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/link-in-p.pdf",
		"P", "Link"); err != nil {
		return err
	}
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/popup-in-p.pdf",
		"P", "Popup"); err != nil {
		return err
	}
	if err := withArtifactAnnotation("internal/checks/annotations/testdata/popup-untagged.pdf",
		"Popup", false); err != nil {
		return err
	}
	if err := withZeroSizeWidget("internal/checks/annotations/testdata/widget-zero-tagged.pdf",
		true); err != nil {
		return err
	}
	if err := withZeroSizeWidget("internal/checks/annotations/testdata/widget-zero-untagged.pdf",
		false); err != nil {
		return err
	}
	// UA-28-026: a Widget with no /TU but an /Alt on the enclosing element passes.
	if err := withAnnotationInAltElement("internal/checks/annotations/testdata/widget-in-alt.pdf",
		"Widget", "Date of birth"); err != nil {
		return err
	}
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/watermark-in-annot.pdf",
		"Annot", "Watermark"); err != nil {
		return err
	}
	if err := withTaggedAnnotation("internal/checks/annotations/testdata/watermark-in-p.pdf",
		"P", "Watermark"); err != nil {
		return err
	}
	if err := withFlaggedAnnotation("internal/checks/annotations/testdata/annot-invisible-tagged.pdf",
		1, true); err != nil {
		return err
	}
	if err := withFlaggedAnnotation("internal/checks/annotations/testdata/annot-invisible-untagged.pdf",
		1, false); err != nil {
		return err
	}
	if err := withContentsAnnotation("internal/checks/annotations/testdata/stamp-none.pdf",
		"Stamp", ""); err != nil {
		return err
	}
	if err := withContentsAnnotation("internal/checks/annotations/testdata/stamp-contents.pdf",
		"Stamp", "Approved by the board"); err != nil {
		return err
	}
	if err := withNamedStamp("internal/checks/annotations/testdata/stamp-name.pdf",
		"Approved"); err != nil {
		return err
	}
	if err := withMediaClip("internal/checks/annotations/testdata/mediaclip-ok.pdf",
		true, true); err != nil {
		return err
	}
	if err := withMediaClip("internal/checks/annotations/testdata/mediaclip-no-ct.pdf",
		false, true); err != nil {
		return err
	}
	if err := withMediaClip("internal/checks/annotations/testdata/mediaclip-no-alt.pdf",
		true, false); err != nil {
		return err
	}
	if err := withFileAttachment("internal/checks/annotations/testdata/fileattach-afrel.pdf",
		true); err != nil {
		return err
	}
	if err := withFileAttachment("internal/checks/annotations/testdata/fileattach-no-afrel.pdf",
		false); err != nil {
		return err
	}
	if err := withContentsAnnotation("internal/checks/annotations/testdata/annot-highlight-bare.pdf",
		"Highlight", ""); err != nil {
		return err
	}
	if err := withContentsAnnotation("internal/checks/annotations/testdata/annot-highlight-contents.pdf",
		"Highlight", "Highlighted passage"); err != nil {
		return err
	}
	if err := withAnnotationInAltElement("internal/checks/annotations/testdata/annot-highlight-alt.pdf",
		"Highlight", "Highlighted passage"); err != nil {
		return err
	}
	// UA-01-016: /MarkInfo /Suspects must not be true (PDF/UA-1 §7.1).
	if err := withSuspects("internal/checks/structure/testdata/suspects-true.pdf"); err != nil {
		return err
	}
	// UA-28-011: Ink/Screen/3D/RichMedia annotations must declare /Contents.
	if err := withContentsAnnotation("internal/checks/annotations/testdata/screen-with-contents.pdf",
		"Screen", "Intro video"); err != nil {
		return err
	}
	// UA-11-005: annotation /Contents needs a determinable language. A Link
	// with /Contents and its own /Lang passes.
	if err := withContentsAnnotationLang("internal/checks/language/testdata/annot-contents-lang.pdf",
		"Link", "External link to example.com", "en-US"); err != nil {
		return err
	}
	if err := withContentsAnnotation("internal/checks/annotations/testdata/screen-no-contents.pdf",
		"Screen", ""); err != nil {
		return err
	}
	if err := withContentsAnnotation("internal/checks/annotations/testdata/ink-no-contents.pdf",
		"Ink", ""); err != nil {
		return err
	}

	// UA-15-004 / UA-15-005: Table row child types and TH /Scope.
	if err := withTableRow("internal/checks/tables/testdata/table-row-cells.pdf",
		[]string{"TD", "TD"}); err != nil {
		return err
	}
	if err := withTableRow("internal/checks/tables/testdata/table-row-mixed.pdf",
		[]string{"P", "TD"}); err != nil {
		return err
	}
	if err := withTableHeader("internal/checks/tables/testdata/table-th-with-scope.pdf",
		"Row"); err != nil {
		return err
	}
	if err := withTableHeader("internal/checks/tables/testdata/table-th-no-scope.pdf",
		""); err != nil {
		return err
	}
	if err := withTableHeaderClassMap(
		"internal/checks/tables/testdata/table-th-scope-via-classmap.pdf"); err != nil {
		return err
	}

	// UA-15-006: table regularity (grid model with RowSpan/ColSpan).
	if err := withTableGrid("internal/checks/tables/testdata/grid-regular.pdf",
		[][]gridCell{{{kind: "TD"}, {kind: "TD"}}, {{kind: "TD"}, {kind: "TD"}}}); err != nil {
		return err
	}
	if err := withTableGrid("internal/checks/tables/testdata/grid-short-row.pdf",
		[][]gridCell{{{kind: "TD"}, {kind: "TD"}}, {{kind: "TD"}}}); err != nil {
		return err
	}
	// Row 1: [TD, TD rowspan=2]; row 2: [TD colspan=2] -> the colspan runs
	// into the column reserved by the rowspan -> overlap.
	if err := withTableGrid("internal/checks/tables/testdata/grid-overlap.pdf",
		[][]gridCell{{{kind: "TD"}, {kind: "TD", rowSpan: 2}}, {{kind: "TD", colSpan: 2}}}); err != nil {
		return err
	}
	// Regular table using a rowspan: [TD rowspan=2, TD] / [TD].
	if err := withTableGrid("internal/checks/tables/testdata/grid-rowspan-regular.pdf",
		[][]gridCell{{{kind: "TD", rowSpan: 2}, {kind: "TD"}}, {{kind: "TD"}}}); err != nil {
		return err
	}

	// UA-15-007: table header association (Scope or Headers/ID).
	// TH cells all scoped -> associated.
	if err := withTableGrid("internal/checks/tables/testdata/header-scope.pdf",
		[][]gridCell{
			{{kind: "TH", scope: "Column"}, {kind: "TH", scope: "Column"}},
			{{kind: "TD"}, {kind: "TD"}},
		}); err != nil {
		return err
	}
	// TH cells with no scope and no Headers/ID -> not associated.
	if err := withTableGrid("internal/checks/tables/testdata/header-none.pdf",
		[][]gridCell{
			{{kind: "TH"}, {kind: "TH"}},
			{{kind: "TD"}, {kind: "TD"}},
		}); err != nil {
		return err
	}
	// TD /Headers referencing a defined TH /ID -> associated.
	if err := withTableGrid("internal/checks/tables/testdata/header-ids.pdf",
		[][]gridCell{
			{{kind: "TH", id: "h1"}, {kind: "TH", id: "h2"}},
			{{kind: "TD", headers: []string{"h1"}}, {kind: "TD", headers: []string{"h2"}}},
		}); err != nil {
		return err
	}
	// TD /Headers referencing an undefined ID -> undefined header.
	if err := withTableGrid("internal/checks/tables/testdata/header-undefined.pdf",
		[][]gridCell{
			{{kind: "TH", id: "h1"}, {kind: "TH", id: "h2"}},
			{{kind: "TD", headers: []string{"nope"}}, {kind: "TD", headers: []string{"h2"}}},
		}); err != nil {
		return err
	}
	// UA-15-009: Table row-group cardinality (THead/TFoot count, TBody
	// presence). bodies=1 is well-formed; bodies=0 has a THead+TFoot but no
	// TBody (fails 7.2-13/-14).
	if err := withGroupedTable("internal/checks/tables/testdata/table-groups-ok.pdf", 1); err != nil {
		return err
	}
	if err := withGroupedTable("internal/checks/tables/testdata/table-groups-no-body.pdf", 0); err != nil {
		return err
	}

	// UA-01-011: Caption must be the first or last child of its parent.
	if err := withCaptionTable("internal/checks/structure/testdata/caption-first.pdf",
		[]string{"Caption", "TR", "TR"}); err != nil {
		return err
	}
	if err := withCaptionTable("internal/checks/structure/testdata/caption-last.pdf",
		[]string{"TR", "TR", "Caption"}); err != nil {
		return err
	}
	if err := withCaptionTable("internal/checks/structure/testdata/caption-middle.pdf",
		[]string{"TR", "Caption", "TR"}); err != nil {
		return err
	}

	// UA-01-013: /Alt and /ActualText must not contain PUA characters.
	if err := withFigure("internal/checks/structure/testdata/figure-alt-pua.pdf",
		figureSpec{alt: strPtr(utf16BEText("a caption \uE000 with PUA"))}); err != nil {
		return err
	}
	if err := withFigure("internal/checks/structure/testdata/figure-alt-clean.pdf",
		figureSpec{alt: strPtr("a clean caption")}); err != nil {
		return err
	}

	// UA-19-002: FENote bidirectional /Ref and /NoteType.
	if err := withFENote("internal/checks/notes/testdata/fenote-valid.pdf",
		true, true, "Footnote"); err != nil {
		return err
	}
	if err := withFENote("internal/checks/notes/testdata/fenote-orphan.pdf",
		true, false, "Footnote"); err != nil { // citation refs note, note does not ref back
		return err
	}
	if err := withFENote("internal/checks/notes/testdata/fenote-bad-notetype.pdf",
		true, true, "Sidebar"); err != nil {
		return err
	}

	// UA-31-009: role-map integrity (circular mapping).
	if err := withRoleMap("internal/checks/structure/testdata/rolemap-cycle.pdf",
		map[string]string{"Foo": "Bar", "Bar": "Foo"}); err != nil {
		return err
	}
	if err := withRoleMap("internal/checks/structure/testdata/rolemap-acyclic.pdf",
		map[string]string{"Foo": "P"}); err != nil {
		return err
	}
	// UA-31-010: remapping a standard type (H1) is forbidden.
	if err := withRoleMap("internal/checks/structure/testdata/rolemap-remaps-standard.pdf",
		map[string]string{"H1": "Foo"}); err != nil {
		return err
	}

	// UA-01-012: TOCI must reference its target via /Ref.
	if err := withTOCItem("internal/checks/structure/testdata/toci-with-ref.pdf",
		true); err != nil {
		return err
	}
	if err := withTOCItem("internal/checks/structure/testdata/toci-no-ref.pdf",
		false); err != nil {
		return err
	}
	// UA-27-002: TOC may contain only TOC/TOCI/Caption (stray P child fails).
	if err := withBadTOC("internal/checks/navigation/testdata/toc-bad-child.pdf"); err != nil {
		return err
	}

	// UA-16-002 / UA-16-003: List item children and /ListNumbering.
	if err := withListItem("internal/checks/lists/testdata/list-li-with-lbody.pdf",
		[]string{"Lbl", "LBody"}); err != nil {
		return err
	}
	if err := withListItem("internal/checks/lists/testdata/list-li-no-lbody.pdf",
		[]string{"P"}); err != nil {
		return err
	}
	if err := withListNumbering("internal/checks/lists/testdata/list-with-numbering.pdf",
		"Decimal"); err != nil {
		return err
	}
	if err := withListNumbering("internal/checks/lists/testdata/list-no-numbering.pdf",
		""); err != nil {
		return err
	}
	// UA-16-003 escalation: when the list carries Lbl children,
	// ISO 14289-2 §8.2.5.25 makes /ListNumbering a hard "shall".
	// withListItem with Lbl + LBody under LI but no /ListNumbering
	// produces the Error variant.
	if err := withListItem("internal/checks/lists/testdata/list-no-numbering-with-lbl.pdf",
		[]string{"Lbl", "LBody"}); err != nil {
		return err
	}

	// UA-14-006: heading style consistency (no /H + /H<n> mix).
	if err := withMixedHeadings("internal/checks/headings/testdata/heading-style-hn-only.pdf",
		[]string{"H1", "H2"}); err != nil {
		return err
	}
	if err := withMixedHeadings("internal/checks/headings/testdata/heading-style-mixed.pdf",
		[]string{"H", "H1"}); err != nil {
		return err
	}

	// UA-17-001: Formula has /Alt or /ActualText. PDF/UA-1 §7.7 is
	// asymmetric -- empty /Alt fails, empty /ActualText passes
	// (veraPDF corpus 7.7-t01 pass-c). The fixtures below cover both
	// halves so the check exercises both code paths.
	if err := withFormula("internal/checks/graphics/testdata/formula-with-alt.pdf",
		formulaSpec{alt: strPtr("Pythagorean theorem")}); err != nil {
		return err
	}
	if err := withFormula("internal/checks/graphics/testdata/formula-no-alt.pdf",
		formulaSpec{}); err != nil {
		return err
	}
	if err := withFormula("internal/checks/graphics/testdata/formula-alt-empty.pdf",
		formulaSpec{alt: strPtr("")}); err != nil {
		return err
	}
	if err := withFormula("internal/checks/graphics/testdata/formula-actualtext.pdf",
		formulaSpec{actualText: strPtr("a^2 + b^2 = c^2")}); err != nil {
		return err
	}
	if err := withFormula("internal/checks/graphics/testdata/formula-actualtext-empty.pdf",
		formulaSpec{actualText: strPtr("")}); err != nil {
		return err
	}
	// UA-17-001 (PDF/UA-2): MathML associated file passes; LaTeX-only AF
	// (no MathML, no /Alt) fails — the BPG "Use of Associated files"
	// pattern requires MathML as the supplement.
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-af.pdf",
		afSpec{rel: "Supplement", subtype: "application/mathml+xml", filename: "math.xml"}); err != nil {
		return err
	}
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-tex-only-af.pdf",
		afSpec{rel: "Source", subtype: "application/x-tex", filename: "math.tex"}); err != nil {
		return err
	}
	// UA-17-002 / UA-17-003 failure fixtures: AF on a Formula with one
	// half of the (AFRelationship=/Supplement, Subtype=application/mathml+xml)
	// pair declared wrong. The "wrong-subtype" file declares the right
	// relationship but the wrong Subtype; "wrong-relationship" is the
	// mirror.
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-wrong-subtype.pdf",
		afSpec{rel: "Supplement", subtype: "text/html", filename: "math.xml"}); err != nil {
		return err
	}
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-wrong-relationship.pdf",
		afSpec{rel: "Source", subtype: "application/mathml+xml", filename: "math.xml"}); err != nil {
		return err
	}
	// UA-17-004: MathML AF stream content failure modes. Filespec
	// metadata is identical to the passing fixture above; only the
	// embedded stream's bytes differ.
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-empty.pdf",
		afSpec{rel: "Supplement", subtype: "application/mathml+xml", filename: "math.xml",
			content: []byte("")}); err != nil {
		return err
	}
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-malformed.pdf",
		afSpec{rel: "Supplement", subtype: "application/mathml+xml", filename: "math.xml",
			content: []byte(`<math xmlns="http://www.w3.org/1998/Math/MathML"><mi`)}); err != nil {
		return err
	}
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-wrong-root.pdf",
		afSpec{rel: "Supplement", subtype: "application/mathml+xml", filename: "math.xml",
			content: []byte(`<foo xmlns="http://www.w3.org/1998/Math/MathML"/>`)}); err != nil {
		return err
	}
	if err := withFormulaUA2AF("internal/checks/graphics/testdata/formula-mathml-wrong-namespace.pdf",
		afSpec{rel: "Supplement", subtype: "application/mathml+xml", filename: "math.xml",
			content: []byte(`<math xmlns="http://example.org/wrong"><mi>x</mi></math>`)}); err != nil {
		return err
	}
	// UA-17-006: mtext children whitelist. Variant A of BPG math
	// representation (math struct child) lets us exercise mtext
	// directly. Three fixtures cover the cases: whitelisted PDF tag
	// child (Span), non-whitelisted known PDF tag child (P), and an
	// unknown tag (MathML-like, heuristic punts).
	if err := withFormulaMTextChildren("internal/checks/graphics/testdata/mtext-children-ok.pdf",
		"Span"); err != nil {
		return err
	}
	if err := withFormulaMTextChildren("internal/checks/graphics/testdata/mtext-children-bad.pdf",
		"P"); err != nil {
		return err
	}
	if err := withFormulaMTextChildren("internal/checks/graphics/testdata/mtext-children-mathml-tag.pdf",
		"mi"); err != nil {
		return err
	}
	// UA-17-005: math struct child namespace. Three fixtures cover:
	// /NS resolves to the W3C MathML URI (PASS), /NS resolves to a
	// different URI (FAIL), and /NS is absent entirely (FAIL).
	if err := withFormulaMathNamespace("internal/checks/graphics/testdata/formula-math-ns-mathml.pdf",
		"http://www.w3.org/1998/Math/MathML"); err != nil {
		return err
	}
	if err := withFormulaMathNamespace("internal/checks/graphics/testdata/formula-math-ns-wrong.pdf",
		"http://example.org/wrong"); err != nil {
		return err
	}
	if err := withFormulaMathNamespace("internal/checks/graphics/testdata/formula-math-ns-none.pdf",
		""); err != nil {
		return err
	}
	// UA-17-015: math must have a Formula ancestor. The stranded
	// fixture places math directly under Document, skipping Formula.
	if err := withMathWithoutFormula(
		"internal/checks/graphics/testdata/math-stranded.pdf"); err != nil {
		return err
	}

	// UA-06-005: DocumentInfo /Title and XMP dc:title agree.
	if err := withTitleAgreement("internal/checks/metadata/testdata/title-agreement-ok.pdf",
		"Sample", "Sample"); err != nil {
		return err
	}
	if err := withTitleAgreement("internal/checks/metadata/testdata/title-agreement-mismatch.pdf",
		"Sample", "Different"); err != nil {
		return err
	}

	// UA-20-001: OCG /Name.
	if err := withOCG("internal/checks/optionalcontent/testdata/ocg-with-name.pdf",
		"Drawing geometry"); err != nil {
		return err
	}
	if err := withOCG("internal/checks/optionalcontent/testdata/ocg-no-name.pdf",
		""); err != nil {
		return err
	}

	// UA-27-001: /Outlines on > 21-page documents.
	if err := withOutlines("internal/checks/navigation/testdata/outlines-present.pdf",
		22, true); err != nil {
		return err
	}
	if err := withOutlines("internal/checks/navigation/testdata/outlines-missing.pdf",
		22, false); err != nil {
		return err
	}

	// UA-27-003: outline/GoTo destinations. A page target fails (§8.8),
	// a structure-element target passes.
	if err := withOutlineDestination("internal/checks/navigation/testdata/dest-page.pdf",
		"page"); err != nil {
		return err
	}
	if err := withOutlineDestination("internal/checks/navigation/testdata/dest-structelem.pdf",
		"structelem"); err != nil {
		return err
	}
	if err := withOutlineDestination("internal/checks/navigation/testdata/dest-goto-page.pdf",
		"goto-page"); err != nil {
		return err
	}

	// UA-08-001: page with an annotation and /Tabs = S passes; /Tabs = R
	// fails. The annotation is what makes the requirement apply.
	if err := withTabs("internal/checks/taborder/testdata/tabs-s.pdf",
		"S", true); err != nil {
		return err
	}
	if err := withTabs("internal/checks/taborder/testdata/tabs-r.pdf",
		"R", true); err != nil {
		return err
	}
	// UA-08-001: annotation present but no /Tabs at all -> fail.
	if err := withTabs("internal/checks/taborder/testdata/tabs-missing.pdf",
		"", true); err != nil {
		return err
	}
	// UA-08-001: bad /Tabs but no annotation -> N/A (requirement does
	// not apply to annotation-free pages).
	if err := withTabs("internal/checks/taborder/testdata/tabs-no-annot.pdf",
		"R", false); err != nil {
		return err
	}
	// UA-14-009: Note structure type forbidden in PDF/UA-2.
	if err := withNoteUA2(
		"internal/checks/notes/testdata/note-in-ua2.pdf"); err != nil {
		return err
	}
	// UA-14-009 regression: an Aside (a valid PDF 2.0 type in the
	// standard namespace) that is role-mapped to Note for legacy 1.7
	// viewers must NOT be treated as a Note. The 2.0 type is
	// authoritative; the global /RoleMap is only a compatibility hint.
	if err := withAsideRolemappedToNoteUA2(
		"internal/checks/notes/testdata/aside-rolemapped-to-note-ua2.pdf"); err != nil {
		return err
	}
	// UA-28-010: XFA forms forbidden in PDF/UA-2.
	if err := withXFAUA2(
		"internal/checks/annotations/testdata/xfa-in-ua2.pdf"); err != nil {
		return err
	}
	// UA-28-009: deprecated annotation types forbidden in PDF/UA-2.
	if err := withDeprecatedAnnotUA2(
		"internal/checks/annotations/testdata/sound-in-ua2.pdf",
		"Sound"); err != nil {
		return err
	}
	// UA-14-007: untyped H structure type forbidden in PDF/UA-2.
	if err := withHeadingUA2(
		"internal/checks/headings/testdata/heading-untyped-h-ua2.pdf",
		"H"); err != nil {
		return err
	}
	// UA-20-002: AS key in OCProperties/D forbidden in PDF/UA-2.
	if err := withOCASUA2(
		"internal/checks/optionalcontent/testdata/ocg-with-as-ua2.pdf"); err != nil {
		return err
	}
	// UA-11-001: PDF/UA-2 catalog /Lang must be present and non-empty.
	if err := withCatalogLangUA2(
		"internal/checks/language/testdata/catalog-lang-empty-ua2.pdf", ""); err != nil {
		return err
	}
	// UA-20-003: optional-content configuration /Name. PDF/UA-1 §7.10-1 is
	// unconditional; PDF/UA-2 §8.7-1 requires /Name only when a /Configs array
	// is present with at least one configuration.
	if err := withOCConfig("internal/checks/optionalcontent/testdata/oc-ua1-unnamed.pdf",
		1, false, nil); err != nil { // UA-1: /D lacks /Name -> fail
		return err
	}
	if err := withOCConfig("internal/checks/optionalcontent/testdata/oc-ua2-noconfigs-unnamed.pdf",
		2, false, nil); err != nil { // UA-2: /D lacks /Name, no /Configs -> pass
		return err
	}
	if err := withOCConfig("internal/checks/optionalcontent/testdata/oc-ua2-configs-unnamed.pdf",
		2, true, []bool{false}); err != nil { // UA-2: /Configs entry lacks /Name -> fail
		return err
	}
	if err := withOCConfig("internal/checks/optionalcontent/testdata/oc-ua2-configs-named.pdf",
		2, true, []bool{true}); err != nil { // UA-2: all named -> pass
		return err
	}
	// UA-01-008: Document structure element in PDF 2.0 namespace.
	if err := withDocumentNamespaceUA2(
		"internal/checks/structure/testdata/document-ns-pdf2.pdf",
		"http://iso.org/pdf2/ssn"); err != nil {
		return err
	}
	if err := withDocumentNamespaceUA2(
		"internal/checks/structure/testdata/document-ns-pdf17.pdf",
		"http://iso.org/pdf/ssn"); err != nil {
		return err
	}
	// PDF/UA-2 broadens the allowed set to S, A, W. Two fixtures
	// drive the UA-2 branch of the check; both pass under UA-2 but
	// would fail under UA-1.
	if err := withTabsUA2("internal/checks/taborder/testdata/tabs-a-ua2.pdf",
		"A"); err != nil {
		return err
	}
	if err := withTabsUA2("internal/checks/taborder/testdata/tabs-w-ua2.pdf",
		"W"); err != nil {
		return err
	}
	// And: R is still disallowed even under UA-2.
	if err := withTabsUA2("internal/checks/taborder/testdata/tabs-r-ua2.pdf",
		"R"); err != nil {
		return err
	}

	// UA-26-001: encryption permits accessibility extraction.
	if err := withPermissions("internal/checks/security/testdata/encrypted-allow-access.pdf",
		true); err != nil {
		return err
	}
	if err := withPermissions("internal/checks/security/testdata/encrypted-block-access.pdf",
		false); err != nil {
		return err
	}

	// UA-28-007: AcroForm widget linked from the structure tree.
	if err := withAcroFormField("internal/checks/annotations/testdata/acroform-linked.pdf",
		true); err != nil {
		return err
	}
	if err := withAcroFormField("internal/checks/annotations/testdata/acroform-orphan.pdf",
		false); err != nil {
		return err
	}

	// UA-11-002: per-element /Lang coverage when Catalog /Lang is absent.
	if err := withLangCoverage("internal/checks/language/testdata/lang-coverage-ok.pdf",
		"en-US"); err != nil {
		return err
	}
	if err := withLangCoverage("internal/checks/language/testdata/lang-coverage-missing.pdf",
		""); err != nil {
		return err
	}

	// UA-19-001: Note carries /ID; /Ref targets resolve to /ID-bearing
	// structure elements.
	if err := withNote("internal/checks/notes/testdata/note-with-id.pdf",
		"n1"); err != nil {
		return err
	}
	if err := withNote("internal/checks/notes/testdata/note-no-id.pdf",
		""); err != nil {
		return err
	}
	if err := withTwoNotes("internal/checks/notes/testdata/note-dup-id.pdf",
		"n1", "n1"); err != nil {
		return err
	}
	if err := withTwoNotes("internal/checks/notes/testdata/note-unique-id.pdf",
		"n1", "n2"); err != nil {
		return err
	}
	if err := withReferenceToBareP("internal/checks/notes/testdata/ref-unresolved.pdf"); err != nil {
		return err
	}

	// UA-31-001: CIDFontType2 declares /CIDToGIDMap = Identity or stream.
	if err := withCIDFontType2("internal/checks/fonts/testdata/cid-identity.pdf",
		"Identity"); err != nil {
		return err
	}
	if err := withCIDFontType2("internal/checks/fonts/testdata/cid-bad-name.pdf",
		"Other"); err != nil {
		return err
	}

	// UA-12-001: Associated Files declare /AFRelationship.
	if err := withAssociatedFile("internal/checks/files/testdata/af-with-relationship.pdf",
		"Source"); err != nil {
		return err
	}
	if err := withAssociatedFile("internal/checks/files/testdata/af-no-relationship.pdf",
		""); err != nil {
		return err
	}
	// UA-12-002: embedded files in the /Names /EmbeddedFiles tree need /Desc.
	if err := withEmbeddedNameTree("internal/checks/files/testdata/embedded-desc.pdf", true, true); err != nil {
		return err
	}
	if err := withEmbeddedNameTree("internal/checks/files/testdata/embedded-no-desc.pdf", false, true); err != nil {
		return err
	}
	// UA-12-003: embedded filespec needs non-empty /F and /UF (omit them here).
	if err := withEmbeddedNameTree("internal/checks/files/testdata/embedded-no-names.pdf", true, false); err != nil {
		return err
	}

	return nil
}

func copyBase(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	return writeAndLog(ctx, dst)
}

func withInfoProperty(dst, key, value string) error {
	if err := api.AddPropertiesFile(basePath, dst, map[string]string{key: value}, nil); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

func withCatalogString(dst, key, value string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	cat, err := ctx.XRefTable.Catalog()
	if err != nil {
		return err
	}
	cat.InsertString(key, value)
	return writeAndLog(ctx, dst)
}

// withSuspects sets the catalog /MarkInfo /Suspects entry to true -- the
// UA-01-016 failure fixture (ISO 14289-1 §7.1 requires Suspects false).
func withSuspects(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	cat, err := ctx.XRefTable.Catalog()
	if err != nil {
		return err
	}
	cat["MarkInfo"] = types.Dict{
		"Marked":   types.Boolean(true),
		"Suspects": types.Boolean(true),
	}
	return writeAndLog(ctx, dst)
}

// withMetadataLang builds a document whose XMP metadata (optionally) carries a
// dc:title language alternative tagged xml:lang="x-default", and whose catalog
// (optionally) declares /Lang. UA-11-007 (ISO 14289-1 §7.2): an x-default
// alternative without a catalog /Lang fails; with /Lang passes; no x-default is
// N/A.
func withMetadataLang(dst string, xDefault, catalogLang bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	title := `<rdf:li>Plain title</rdf:li>`
	if xDefault {
		title = `<rdf:li xml:lang="x-default">Default title</rdf:li>`
	}
	xmp := `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title><rdf:Alt>` + title + `</rdf:Alt></dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta><?xpacket end="w"?>`

	sd := types.StreamDict{
		Dict:    types.Dict{"Type": types.Name("Metadata"), "Subtype": types.Name("XML")},
		Content: []byte(xmp),
	}
	if err := sd.Encode(); err != nil {
		return err
	}
	ref, err := xrt.IndRefForNewObject(sd)
	if err != nil {
		return err
	}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["Metadata"] = *ref
	if catalogLang {
		cat["Lang"] = types.StringLiteral("en")
	}
	return writeAndLog(ctx, dst)
}

func withMetadataStream(dst, xmp string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	sd := types.StreamDict{
		Dict: types.Dict{
			"Type":    types.Name("Metadata"),
			"Subtype": types.Name("XML"),
		},
		Content: []byte(xmp),
	}
	if err := sd.Encode(); err != nil {
		return err
	}
	ref, err := ctx.XRefTable.IndRefForNewObject(sd)
	if err != nil {
		return err
	}
	cat, err := ctx.XRefTable.Catalog()
	if err != nil {
		return err
	}
	cat["Metadata"] = *ref
	return writeAndLog(ctx, dst)
}

func withDisplayDocTitle(dst string, value bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	cat, err := ctx.XRefTable.Catalog()
	if err != nil {
		return err
	}
	cat["ViewerPreferences"] = types.Dict{
		"DisplayDocTitle": types.Boolean(value),
	}
	return writeAndLog(ctx, dst)
}

// withFont attaches a synthetic TrueType font dict to the first page's
// /Resources/Font so the font's /BaseFont can drive Fonts()-based
// checks. embedded toggles whether the font carries a FontDescriptor
// with a FontFile2 stream; hasToUnicode toggles the /ToUnicode entry.
//
// The base PDF already contains four real fonts (all embedded; three
// without ToUnicode); the synthesised font keeps a distinct BaseFont
// name so tests can assert findings against it without being misled
// by the inherited noise.
func withFont(dst, baseFont string, embedded, hasToUnicode bool) error {
	return withFontCMap(dst, baseFont, embedded, hasToUnicode, "")
}

// withFontCMap is withFont with an explicit /ToUnicode CMap body. When
// cmapBody is empty and hasToUnicode is true, a stub (empty) CMap is written
// as before; otherwise cmapBody becomes the CMap stream content -- used to
// seed a forbidden mapping (e.g. a bfchar to <0000>) for UA-10-003.
func withFontCMap(dst, baseFont string, embedded, hasToUnicode bool, cmapBody string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	font := types.Dict{
		"Type":      types.Name("Font"),
		"Subtype":   types.Name("TrueType"),
		"BaseFont":  types.Name(baseFont),
		"FirstChar": types.Integer(32),
		"LastChar":  types.Integer(32),
		"Widths":    types.Array{types.Integer(500)},
	}

	if embedded {
		ff2 := types.StreamDict{
			Dict:    types.Dict{"Length1": types.Integer(4)},
			Content: []byte("STUB"),
		}
		if err := ff2.Encode(); err != nil {
			return err
		}
		ff2Ref, err := xrt.IndRefForNewObject(ff2)
		if err != nil {
			return err
		}
		fd := types.Dict{
			"Type":        types.Name("FontDescriptor"),
			"FontName":    types.Name(baseFont),
			"Flags":       types.Integer(32),
			"FontBBox":    types.Array{types.Integer(0), types.Integer(0), types.Integer(1000), types.Integer(1000)},
			"ItalicAngle": types.Integer(0),
			"Ascent":      types.Integer(800),
			"Descent":     types.Integer(-200),
			"CapHeight":   types.Integer(700),
			"StemV":       types.Integer(80),
			"FontFile2":   *ff2Ref,
		}
		fdRef, err := xrt.IndRefForNewObject(fd)
		if err != nil {
			return err
		}
		font["FontDescriptor"] = *fdRef
	}

	if hasToUnicode {
		body := "/CIDInit /ProcSet findresource begin\n" +
			"12 dict begin\nbegincmap\nendcmap\nend\nend\n"
		if cmapBody != "" {
			body = cmapBody
		}
		cmap := types.StreamDict{
			Dict:    types.Dict{},
			Content: []byte(body),
		}
		if err := cmap.Encode(); err != nil {
			return err
		}
		cmapRef, err := xrt.IndRefForNewObject(cmap)
		if err != nil {
			return err
		}
		font["ToUnicode"] = *cmapRef
	}

	fontRef, err := xrt.IndRefForNewObject(font)
	if err != nil {
		return err
	}

	// Attach to the first page's Resources/Font so pdfcpu doesn't
	// strip the new font as unreferenced during write-back.
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	resObj, found := pageDict.Find("Resources")
	var resources types.Dict
	if found {
		resources, _ = xrt.DereferenceDict(resObj)
	}
	if resources == nil {
		resources = types.Dict{}
		pageDict["Resources"] = resources
	}
	var fontMap types.Dict
	if obj, ok := resources.Find("Font"); ok {
		fontMap, _ = xrt.DereferenceDict(obj)
	}
	if fontMap == nil {
		fontMap = types.Dict{}
		resources["Font"] = fontMap
	}
	fontMap["FProbe"] = *fontRef

	// Add a minimal content stream that references the synthetic font
	// via Tf. The content-stream-aware checks only flag fonts that are
	// actually used on a page; without this, declaring a font in
	// /Resources is no longer enough to trigger a finding.
	content := types.StreamDict{
		Dict:    types.Dict{},
		Content: []byte("BT /FProbe 12 Tf 72 720 Td (X) Tj ET\n"),
	}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef

	return writeAndLog(ctx, dst)
}

// withParentChildren builds a tagged PDF whose Document contains a
// single StructElem of type parentType, whose /K is an array of
// StructElems with the given childTypes (in order). Used to compose
// fixture pairs that vary only the structural shape (e.g. Table → TR
// vs Table → P) for the corresponding structural checks.
func withParentChildren(dst, parentType string, childTypes ...string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	parentElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name(parentType),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	parentRef, err := xrt.IndRefForNewObject(parentElem)
	if err != nil {
		return err
	}

	var kids types.Array
	for _, t := range childTypes {
		child := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name(t),
			"P":    *parentRef,
			"Pg":   pageRef,
		}
		ref, err := xrt.IndRefForNewObject(child)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	parentElem["K"] = kids
	docElem["K"] = *parentRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// withRootChildren builds a tagged PDF whose StructTreeRoot /K is an array
// of StructElems with the given types, each a direct child of the root. Used
// to produce the UA-01-015 failure shape: more than one top-level element (or
// a single non-Document child). Every element carries /P so the fixture only
// exercises the single-Document-child rule, not the /P rule (UA-01-014).
func withRootChildren(dst string, childTypes ...string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	var kids types.Array
	for _, t := range childTypes {
		el := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name(t),
			"P":    *streeRef,
			"Pg":   pageRef,
		}
		ref, err := xrt.IndRefForNewObject(el)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	streeDict["K"] = kids
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// withMissingParent builds a tagged PDF whose Document contains a single P
// element that omits its own /P (parent) entry -- the UA-01-014 failure
// shape. The Document element itself keeps its /P (pointing at the
// StructTreeRoot), so exactly one element is defective.
func withMissingParent(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	// Deliberately omit /P on this child element.
	child := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("P"),
		"Pg":   pageRef,
	}
	childRef, err := xrt.IndRefForNewObject(child)
	if err != nil {
		return err
	}
	docElem["K"] = *childRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// withRoleMappedHeading writes a tagged PDF that uses a custom structure
// type "MyHeading" role-mapped to H1, followed by H3. With role-map
// resolution working, the structure should be read as H1 -> H3 and
// trigger UA-14-003 (level skip). Without resolution the custom type
// would be ignored and the level skip undetected -- the fixture is
// therefore a regression net for the role-mapping plumbing.
func withRoleMappedHeading(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{
		"Type":    types.Name("StructTreeRoot"),
		"RoleMap": types.Dict{"MyHeading": types.Name("H1")},
	}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}

	myHeading := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("MyHeading"),
		"P":    *docRef,
	}
	myHRef, err := xrt.IndRefForNewObject(myHeading)
	if err != nil {
		return err
	}
	h3 := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("H3"),
		"P":    *docRef,
	}
	h3Ref, err := xrt.IndRefForNewObject(h3)
	if err != nil {
		return err
	}
	docElem["K"] = types.Array{*myHRef, *h3Ref}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// withHeadings derives a tagged PDF whose Document StructElem has one
// H<level> child per entry in levels, in order. Used to build both the
// well-nested (levels = 1,2,3) and the level-skipping (levels = 1,3)
// fixtures for UA-14-003.
func withHeadings(dst string, levels []int) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}

	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	var kids types.Array
	for _, level := range levels {
		h := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name(fmt.Sprintf("H%d", level)),
			"P":    *docRef,
			"Pg":   pageRef,
		}
		hRef, err := xrt.IndRefForNewObject(h)
		if err != nil {
			return err
		}
		kids = append(kids, *hRef)
	}
	docElem["K"] = kids
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// figureSpec mirrors formulaSpec: nil means "entry absent", a non-nil
// pointer to "" means "entry present but empty". UA-13-004 treats
// /Alt and /ActualText with the same asymmetry as UA-17-001 -- empty
// /Alt fails, empty /ActualText passes ("render as silence").
type figureSpec struct {
	alt        *string
	actualText *string
	lang       *string // /Lang on the Figure element; nil omits it
}

// withFigure derives a tagged PDF whose structure tree contains a single
// Figure StructElem with the requested /Alt and/or /ActualText entries.
// All other fields match the minimal scaffolding from withStructTree.
func withFigure(dst string, spec figureSpec) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	figElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Figure"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	if spec.alt != nil {
		figElem["Alt"] = types.StringLiteral(*spec.alt)
	}
	if spec.actualText != nil {
		figElem["ActualText"] = types.StringLiteral(*spec.actualText)
	}
	if spec.lang != nil {
		figElem["Lang"] = types.StringLiteral(*spec.lang)
	}
	figRef, err := xrt.IndRefForNewObject(figElem)
	if err != nil {
		return err
	}
	docElem["K"] = *figRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// withStructTree adds a minimal tagged-PDF structure to the base: an
// empty StructTreeRoot containing a single Document StructElem, an empty
// ParentTree, plus MarkInfo /Marked true on the Catalog. Not a fully
// PDF/UA-conforming tagged document (no marked content sequences) -- the
// purpose is to exercise checks that only verify the presence and shape
// of the structure scaffolding.
func withStructTree(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	// StructTreeRoot is added first so child elements can reference it
	// via /P; /K and /ParentTree are wired in after the kids exist.
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}

	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}

	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}

	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	return writeAndLog(ctx, dst)
}

// withUntaggedContent emits a tagged PDF whose page paints text via Tj
// outside of any BDC/BMC marked-content sequence. The structure tree
// declares /MarkInfo /Marked true so the document claims tagged
// status, making the loose content a real PDF/UA-1 §7.1 violation
// rather than just an untagged file.
func withUntaggedContent(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	// Build a font + a content stream that paints text NOT inside any
	// BDC, plus a separate text run that IS inside a BDC -- so the
	// finding has both kinds of evidence to verify against.
	font := types.Dict{
		"Type":      types.Name("Font"),
		"Subtype":   types.Name("TrueType"),
		"BaseFont":  types.Name("PDFA11YTestUntagged"),
		"FirstChar": types.Integer(32),
		"LastChar":  types.Integer(32),
		"Widths":    types.Array{types.Integer(500)},
	}
	fontRef, err := xrt.IndRefForNewObject(font)
	if err != nil {
		return err
	}

	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	pageDict["Resources"] = types.Dict{
		"Font": types.Dict{"FProbe": *fontRef},
	}

	// First Tj is outside any BDC -- the violation. Second Tj sits
	// inside a /P BDC sequence -- legitimately tagged.
	content := types.StreamDict{
		Dict: types.Dict{},
		Content: []byte("BT /FProbe 12 Tf 72 720 Td (X) Tj ET\n" +
			"/P <</MCID 0 >> BDC BT /FProbe 12 Tf 72 700 Td (Y) Tj ET EMC\n"),
	}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef

	return writeAndLog(ctx, dst)
}

func writeAndLog(ctx *pdfmodel.Context, dst string) error {
	if err := api.WriteContextFile(ctx, dst); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// firstPageRef returns an IndirectRef to the first Page leaf in the
// document. Used to set /Pg on synthesised StructElems so the resulting
// fixtures look like real PDF/UA documents (every leaf carries its
// page reference, which downstream reporters surface as "page N").
func firstPageRef(xrt *pdfmodel.XRefTable) (types.IndirectRef, error) {
	pagesRef, err := xrt.Pages()
	if err != nil {
		return types.IndirectRef{}, err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return types.IndirectRef{}, err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	if len(kids) == 0 {
		return types.IndirectRef{}, fmt.Errorf("no pages in document")
	}
	ref, ok := kids[0].(types.IndirectRef)
	if !ok {
		return types.IndirectRef{}, fmt.Errorf("first /Kids entry is not an IndirectRef")
	}
	return ref, nil
}

// writeBlankPDF emits a minimal one-page PDF (no fonts, no content) at
// dst. The byte layout is hand-built: header, three objects (Catalog,
// Pages, Page), xref table with computed offsets, trailer, startxref,
// EOF. pdfcpu happily reads this and lets us mutate it like any other
// document. Keeping the base under our own copyright avoids embedding
// vendor-licensed material into the fixture chain.
func writeBlankPDF(dst string) error {
	var buf bytes.Buffer

	// Header. The high-bit comment line is the conventional "this is
	// binary" marker that some viewers and validators look for.
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("%\xff\xff\xff\xff\n")

	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 4\n")
	buf.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&buf, "%010d 00000 n \n", off1)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off2)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off3)

	buf.WriteString("trailer\n<< /Size 4 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	return os.WriteFile(dst, buf.Bytes(), 0o644)
}

// withAssociatedFile writes a PDF with a Catalog /AF array of one
// filespec entry. The PDF bytes are emitted by hand because pdfcpu's
// Writer strips the filespec during validation -- the minimal four-
// object layout (Catalog, Pages, Page, Filespec) is identical to
// writeBlankPDF plus the /AF reference. When relationship is
// non-empty the filespec declares /AFRelationship (the UA-12-001
// passing case); otherwise the entry is omitted (failing case).
// The embedded file stream itself is intentionally absent -- the
// check inspects only the filespec dictionary's /AFRelationship.
func withAssociatedFile(dst, relationship string) error {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	buf.WriteString("%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /AF [4 0 R] >>\nendobj\n")

	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")

	off4 := offset()
	if relationship != "" {
		fmt.Fprintf(&buf,
			"4 0 obj\n<< /Type /Filespec /F (data.txt) /UF (data.txt) /AFRelationship /%s >>\nendobj\n",
			relationship)
	} else {
		buf.WriteString("4 0 obj\n<< /Type /Filespec /F (data.txt) /UF (data.txt) >>\nendobj\n")
	}

	xrefOff := offset()
	buf.WriteString("xref\n0 5\n")
	buf.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&buf, "%010d 00000 n \n", off1)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off2)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off3)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off4)

	buf.WriteString("trailer\n<< /Size 5 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withEmbeddedNameTree writes a PDF whose catalog /Names /EmbeddedFiles name
// tree holds one file specification with an /EF EmbeddedFile stream. withDesc
// toggles the /Desc entry (UA-12-002); withNames toggles the /F and /UF file
// name keys (UA-12-003). Emitted by hand because pdfcpu's Writer rewrites the
// name tree during validation.
func withEmbeddedNameTree(dst string, withDesc, withNames bool) error {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	buf.WriteString("%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }
	offs := make([]int, 7)

	offs[1] = offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /Names 5 0 R >>\nendobj\n")

	offs[2] = offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	offs[3] = offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")

	offs[4] = offset()
	desc := ""
	if withDesc {
		desc = " /Desc (A data file)"
	}
	names := ""
	if withNames {
		names = " /F (data.txt) /UF (data.txt)"
	}
	fmt.Fprintf(&buf,
		"4 0 obj\n<< /Type /Filespec%s%s /EF << /F 6 0 R /UF 6 0 R >> >>\nendobj\n",
		names, desc)

	offs[5] = offset()
	buf.WriteString("5 0 obj\n<< /EmbeddedFiles << /Names [(data.txt) 4 0 R] >> >>\nendobj\n")

	offs[6] = offset()
	payload := "hello"
	fmt.Fprintf(&buf, "6 0 obj\n<< /Type /EmbeddedFile /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(payload), payload)

	xrefOff := offset()
	buf.WriteString("xref\n0 7\n")
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offs[i])
	}
	buf.WriteString("trailer\n<< /Size 7 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// afSpec describes one /AF filespec attached to the Formula in
// withFormulaUA2AF: the relationship name, the embedded-file MIME
// subtype, the filename, and an optional explicit stream payload.
// When content is nil, withFormulaUA2AF picks a minimal payload
// per subtype (a tiny MathML fragment for application/mathml+xml,
// a literal "$x$" for x-tex). Tests for UA-17-004 pass an explicit
// content (empty, malformed XML, wrong root, wrong namespace) to
// exercise the content-validation paths.
type afSpec struct {
	rel      string
	subtype  string
	filename string
	content  []byte
}

// withFormulaUA2AF writes a PDF/UA-2-flavoured one-page document
// with a Document → Formula structure tree where the Formula
// carries /AF [filespecRef]. The filespec resolves to an
// EmbeddedFile stream whose /Subtype is the supplied MIME type and
// whose /AFRelationship is the supplied name. The XMP Metadata
// stream declares pdfuaid:part = 2 so UA-17-001's spec autodetect
// runs the UA-2 branch.
//
// Written by hand because the path goes through several object
// types pdfcpu's Writer is happy to rearrange or strip; the
// UA-12-001 fixture took the same approach for the same reason.
func withFormulaUA2AF(dst string, af afSpec) error {
	streamContent := af.content
	if streamContent == nil {
		switch af.subtype {
		case "application/mathml+xml":
			streamContent = []byte(`<math xmlns="http://www.w3.org/1998/Math/MathML"><mi>x</mi></math>`)
		default:
			streamContent = []byte(`$x$`)
		}
	}
	// Encode '/' as #2F in the Subtype Name per PDF name-escape rules.
	subtypeEnc := strings.ReplaceAll(af.subtype, "/", "#2F")

	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	// 1 Catalog -> StructTreeRoot, MarkInfo, Metadata.
	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 8 0 R >>\nendobj\n")

	// 2 Pages -> 3.
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// 3 Page (no real content; the structure tree references nothing in
	// the content stream because no Formula check inspects MCIDs).
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")

	// 4 StructTreeRoot -> 5.
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")

	// 5 Document -> 6.
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")

	// 6 Formula with /AF [7 0 R].
	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /Formula /P 5 0 R /Pg 3 0 R /AF [7 0 R] >>\nendobj\n")

	// 7 Filespec -> 9 (EmbeddedFile stream).
	off7 := offset()
	fmt.Fprintf(&buf,
		"7 0 obj\n<< /Type /Filespec /F (%s) /UF (%s) /AFRelationship /%s /EF << /F 9 0 R /UF 9 0 R >> >>\nendobj\n",
		af.filename, af.filename, af.rel)

	// 8 XMP Metadata.
	off8 := offset()
	fmt.Fprintf(&buf,
		"8 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	// 9 EmbeddedFile stream with the supplied Subtype.
	off9 := offset()
	fmt.Fprintf(&buf,
		"9 0 obj\n<< /Type /EmbeddedFile /Subtype /%s /Length %d >>\nstream\n",
		subtypeEnc, len(streamContent))
	buf.Write(streamContent)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 10\n")
	buf.WriteString("0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7, off8, off9} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 10 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withFormulaMTextChildren writes a PDF/UA-2-flavoured one-page
// document with a Document → Formula → math → mtext → <childTag>
// structure tree. The XMP Metadata stream declares
// pdfuaid:part = 2 so UA-17-006's spec autodetect runs the UA-2
// branch. The check inspects mtext's children and decides per
// childTag whether the document conforms; we only need to vary
// that single leaf to exercise every branch (whitelisted,
// non-whitelisted known PDF tag, unknown tag).
func withFormulaMTextChildren(dst, childTag string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 9 0 R >>\nendobj\n")

	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")

	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")

	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")

	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /Formula /P 5 0 R /Pg 3 0 R /K 7 0 R >>\nendobj\n")

	off7 := offset()
	buf.WriteString("7 0 obj\n<< /Type /StructElem /S /math /P 6 0 R /Pg 3 0 R /K 8 0 R >>\nendobj\n")

	off8 := offset()
	fmt.Fprintf(&buf,
		"8 0 obj\n<< /Type /StructElem /S /mtext /P 7 0 R /Pg 3 0 R /K 10 0 R >>\nendobj\n")

	off9 := offset()
	fmt.Fprintf(&buf,
		"9 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	off10 := offset()
	fmt.Fprintf(&buf,
		"10 0 obj\n<< /Type /StructElem /S /%s /P 8 0 R /Pg 3 0 R >>\nendobj\n", childTag)

	xrefOff := offset()
	buf.WriteString("xref\n0 11\n")
	buf.WriteString("0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7, off8, off9, off10} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 11 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withTableHeaderClassMap writes a minimal document whose TH
// carries its /Scope attribute through a ClassMap reference
// rather than a direct /A entry: TH has /C /TH-col, and
// StructTreeRoot has /ClassMap << /TH-col << /O /Table /Scope
// /Column >> >>. ISO 32000-1 §14.7.5.3/4 says both routes are
// equivalent; UA-15-005 must accept either.
//
// Written by hand because pdfcpu's withTablePattern was built
// before ClassMap support landed in the fixture set; extending
// it would touch every existing table fixture for one extra
// case.
func withTableHeaderClassMap(dst string) error {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> /ClassMap << /TH-col << /O /Table /Scope /Column >> >> >>\nendobj\n")
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")
	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /Table /P 5 0 R /Pg 3 0 R /K 7 0 R >>\nendobj\n")
	off7 := offset()
	buf.WriteString("7 0 obj\n<< /Type /StructElem /S /TR /P 6 0 R /Pg 3 0 R /K 8 0 R >>\nendobj\n")
	off8 := offset()
	buf.WriteString("8 0 obj\n<< /Type /StructElem /S /TH /P 7 0 R /Pg 3 0 R /C /TH-col >>\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 9\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7, off8} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 9 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withMathWithoutFormula writes a PDF/UA-2 document whose
// structure tree contains a `math` element (in the MathML
// namespace) placed directly under Document, with no Formula
// ancestor. UA-17-015 fires on this layout.
func withMathWithoutFormula(dst string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 8 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")
	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /math /P 5 0 R /Pg 3 0 R /NS 7 0 R >>\nendobj\n")
	off7 := offset()
	buf.WriteString("7 0 obj\n<< /Type /Namespace /NS (http://www.w3.org/1998/Math/MathML) >>\nendobj\n")
	off8 := offset()
	fmt.Fprintf(&buf,
		"8 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 9\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7, off8} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 9 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withFormulaMathNamespace writes a PDF/UA-2 document with a
// Document → Formula → math structure tree where math optionally
// declares /NS pointing at a Namespace dictionary carrying the
// supplied URI. When nsURI is empty the math element gets no /NS
// entry, exercising the "no namespace" failure path of UA-17-005.
func withFormulaMathNamespace(dst, nsURI string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	// Object layout:
	//   1 Catalog, 2 Pages, 3 Page, 4 StructTreeRoot, 5 Document,
	//   6 Formula, 7 math (optionally /NS 9), 8 Metadata,
	//   9 Namespace (only when nsURI != "")
	withNS := nsURI != ""
	count := 8
	if withNS {
		count = 9
	}

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 8 0 R >>\nendobj\n")

	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")

	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")

	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")

	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /Formula /P 5 0 R /Pg 3 0 R /K 7 0 R >>\nendobj\n")

	off7 := offset()
	if withNS {
		buf.WriteString("7 0 obj\n<< /Type /StructElem /S /math /P 6 0 R /Pg 3 0 R /NS 9 0 R >>\nendobj\n")
	} else {
		buf.WriteString("7 0 obj\n<< /Type /StructElem /S /math /P 6 0 R /Pg 3 0 R >>\nendobj\n")
	}

	off8 := offset()
	fmt.Fprintf(&buf,
		"8 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	off9 := offset()
	if withNS {
		fmt.Fprintf(&buf,
			"9 0 obj\n<< /Type /Namespace /NS (%s) >>\nendobj\n", nsURI)
	}

	xrefOff := offset()
	fmt.Fprintf(&buf, "xref\n0 %d\n", count+1)
	buf.WriteString("0000000000 65535 f \n")
	offsets := []int{off1, off2, off3, off4, off5, off6, off7, off8}
	if withNS {
		offsets = append(offsets, off9)
	}
	for _, o := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", count+1)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withCIDFontType2 builds a Type 0 composite font whose descendant
// is a CIDFontType2 with the given /CIDToGIDMap Name value (e.g.
// "Identity" for the passing case, "Other" for the UA-31-001
// failure case). The font program is embedded so UA-09-001 stays
// quiet on the fixture.
func withCIDFontType2(dst, cidToGIDMap string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	// Embedded font program (FontFile2 for TrueType-backed CID).
	ff2 := types.StreamDict{
		Dict:    types.Dict{"Length1": types.Integer(4)},
		Content: []byte("STUB"),
	}
	if err := ff2.Encode(); err != nil {
		return err
	}
	ff2Ref, err := xrt.IndRefForNewObject(ff2)
	if err != nil {
		return err
	}
	fd := types.Dict{
		"Type":        types.Name("FontDescriptor"),
		"FontName":    types.Name("PDFA11YCIDTest"),
		"Flags":       types.Integer(32),
		"FontBBox":    types.Array{types.Integer(0), types.Integer(0), types.Integer(1000), types.Integer(1000)},
		"ItalicAngle": types.Integer(0),
		"Ascent":      types.Integer(800),
		"Descent":     types.Integer(-200),
		"CapHeight":   types.Integer(700),
		"StemV":       types.Integer(80),
		"FontFile2":   *ff2Ref,
	}
	fdRef, err := xrt.IndRefForNewObject(fd)
	if err != nil {
		return err
	}

	// CIDSystemInfo (mandatory on a CIDFont).
	csi := types.Dict{
		"Registry":   types.StringLiteral("Adobe"),
		"Ordering":   types.StringLiteral("Identity"),
		"Supplement": types.Integer(0),
	}

	cidFont := types.Dict{
		"Type":           types.Name("Font"),
		"Subtype":        types.Name("CIDFontType2"),
		"BaseFont":       types.Name("PDFA11YCIDTest"),
		"CIDSystemInfo":  csi,
		"FontDescriptor": *fdRef,
		"CIDToGIDMap":    types.Name(cidToGIDMap),
	}
	cidRef, err := xrt.IndRefForNewObject(cidFont)
	if err != nil {
		return err
	}

	font := types.Dict{
		"Type":            types.Name("Font"),
		"Subtype":         types.Name("Type0"),
		"BaseFont":        types.Name("PDFA11YCIDTest"),
		"Encoding":        types.Name("Identity-H"),
		"DescendantFonts": types.Array{*cidRef},
	}
	fontRef, err := xrt.IndRefForNewObject(font)
	if err != nil {
		return err
	}

	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	pageDict["Resources"] = types.Dict{
		"Font": types.Dict{"FProbe": *fontRef},
	}
	return writeAndLog(ctx, dst)
}

// withNote builds Document → Note. When id is non-empty the Note
// carries /ID; otherwise the entry is omitted -- the UA-19-001
// half-A failure pattern.
func withNote(dst, id string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	note := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Note"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	if id != "" {
		note["ID"] = types.StringLiteral(id)
	}
	noteRef, err := xrt.IndRefForNewObject(note)
	if err != nil {
		return err
	}
	docElem["K"] = *noteRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withTwoNotes builds Document → [Note(/ID=id1), Note(/ID=id2)]. Passing the
// same value for both yields the UA-19-003 failure shape (duplicate Note /ID);
// distinct values pass.
func withTwoNotes(dst, id1, id2 string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	var kids types.Array
	for _, id := range []string{id1, id2} {
		note := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name("Note"),
			"P":    *docRef,
			"Pg":   pageRef,
			"ID":   types.StringLiteral(id),
		}
		ref, err := xrt.IndRefForNewObject(note)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	docElem["K"] = kids
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withReferenceToBareP builds Document → [Note(/ID="n1"), P, Reference].
// The Reference's /Ref array points at the P (which carries no /ID),
// so UA-19-001's half B emits a finding while the Note itself passes.
func withReferenceToBareP(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	note := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Note"),
		"P":    *docRef,
		"Pg":   pageRef,
		"ID":   types.StringLiteral("n1"),
	}
	noteRef, err := xrt.IndRefForNewObject(note)
	if err != nil {
		return err
	}
	bareP := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("P"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	bareRef, err := xrt.IndRefForNewObject(bareP)
	if err != nil {
		return err
	}
	reference := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Reference"),
		"P":    *docRef,
		"Pg":   pageRef,
		"Ref":  types.Array{*bareRef},
	}
	refRef, err := xrt.IndRefForNewObject(reference)
	if err != nil {
		return err
	}
	docElem["K"] = types.Array{*noteRef, *bareRef, *refRef}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withLangCoverage builds a tagged PDF with no Catalog /Lang and a
// single P StructElem. When lang is non-empty, the P carries /Lang
// (UA-11-002 passes); empty lang omits the entry and the P inherits
// nothing -- the failure path. Catalog /Lang is deliberately omitted
// so the check enters its per-element branch instead of declining.
func withLangCoverage(dst, lang string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("P"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	if lang != "" {
		pElem["Lang"] = types.StringLiteral(lang)
	}
	pRef, err := xrt.IndRefForNewObject(pElem)
	if err != nil {
		return err
	}
	docElem["K"] = *pRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withAcroFormField attaches a single AcroForm field whose merged
// widget annotation sits on the first page. When linked is true the
// widget carries /StructParent (the UA-28-007 passing case);
// otherwise /StructParent is omitted -- the failure pattern. The
// fixture also wires the widget into the page's /Annots so it looks
// like a realistic form field rather than an orphan AcroForm entry.
func withAcroFormField(dst string, linked bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}

	widget := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"FT":      types.Name("Tx"),
		"T":       types.StringLiteral("name"),
		"TU":      types.StringLiteral("Full name"),
		"Rect":    types.Array{types.Integer(100), types.Integer(100), types.Integer(200), types.Integer(120)},
		"P":       kids[0],
	}
	if linked {
		widget["StructParent"] = types.Integer(0)
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["AcroForm"] = types.Dict{
		"Fields": types.Array{*widgetRef},
	}
	return writeAndLog(ctx, dst)
}

// withOCG adds an Optional Content Group to the catalog. When name is
// non-empty, the OCG carries /Name; otherwise /Name is omitted -- the
// UA-20-001 failure pattern.
func withOCG(dst, name string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	ocg := types.Dict{
		"Type": types.Name("OCG"),
	}
	if name != "" {
		ocg["Name"] = types.StringLiteral(name)
	}
	ocgRef, err := xrt.IndRefForNewObject(ocg)
	if err != nil {
		return err
	}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["OCProperties"] = types.Dict{
		"OCGs": types.Array{*ocgRef},
		"D":    types.Dict{"Order": types.Array{*ocgRef}},
	}
	return writeAndLog(ctx, dst)
}

// withOutlines produces a multi-page PDF whose page count exceeds the
// UA-27-001 threshold (21). When withTree is true, the catalog carries
// an /Outlines dictionary with one /First entry; otherwise the entry
// is omitted -- the failing path.
func withOutlines(dst string, pageCount int, withTree bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	if err := extendPages(xrt, pageCount); err != nil {
		return err
	}
	if withTree {
		cat, err := xrt.Catalog()
		if err != nil {
			return err
		}
		outline := types.Dict{
			"Type":  types.Name("Outline"),
			"Title": types.StringLiteral("Top"),
		}
		outlineRef, err := xrt.IndRefForNewObject(outline)
		if err != nil {
			return err
		}
		outlines := types.Dict{
			"Type":  types.Name("Outlines"),
			"First": *outlineRef,
			"Last":  *outlineRef,
			"Count": types.Integer(1),
		}
		outlinesRef, err := xrt.IndRefForNewObject(outlines)
		if err != nil {
			return err
		}
		outline["Parent"] = *outlinesRef
		cat["Outlines"] = *outlinesRef
	}
	return writeAndLog(ctx, dst)
}

// extendPages grows the page tree to total pages by appending fresh
// Page dicts that mirror the base page's MediaBox. The /Pages /Count
// is updated to match. Used by the Outlines fixture to clear the
// page-count threshold UA-27-001 enforces.
func extendPages(xrt *pdfmodel.XRefTable, total int) error {
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	for len(kids) < total {
		page := types.Dict{
			"Type":      types.Name("Page"),
			"Parent":    *pagesRef,
			"MediaBox":  types.Array{types.Integer(0), types.Integer(0), types.Integer(612), types.Integer(792)},
			"Resources": types.Dict{},
		}
		ref, err := xrt.IndRefForNewObject(page)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	pagesDict["Kids"] = kids
	pagesDict["Count"] = types.Integer(len(kids))
	return nil
}

// withOutlineLang builds a document with (optionally) an /Outlines tree of one
// bookmark and (optionally) a catalog /Lang. UA-11-006 (ISO 14289-1 §7.2)
// requires a document with bookmarks to declare a catalog /Lang so the outline
// language is determinable: outlines without /Lang fail, with /Lang pass, and
// no outlines is N/A.
func withOutlineLang(dst string, withOutline, withLang bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	if withLang {
		cat["Lang"] = types.StringLiteral("en")
	}
	if withOutline {
		outline := types.Dict{"Type": types.Name("Outline"), "Title": types.StringLiteral("Section")}
		outlineRef, err := xrt.IndRefForNewObject(outline)
		if err != nil {
			return err
		}
		outlines := types.Dict{
			"Type":  types.Name("Outlines"),
			"First": *outlineRef,
			"Last":  *outlineRef,
			"Count": types.Integer(1),
		}
		outlinesRef, err := xrt.IndRefForNewObject(outlines)
		if err != nil {
			return err
		}
		outline["Parent"] = *outlinesRef
		cat["Outlines"] = *outlinesRef
	}
	return writeAndLog(ctx, dst)
}

// withOutlineDestination writes a single-page tagged document with one
// outline item whose destination targets either the page or the document's
// root structure element, exercising UA-27-003 (ISO 14289-2 §8.8):
//
//   - "page":       the item's /Dest points at the page       -> fail (8.8-1)
//   - "structelem": the item's /Dest points at a StructElem   -> pass
//   - "goto-page":  the item's /A GoTo /D points at the page  -> fail (8.8-2)
func withOutlineDestination(dst, mode string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	// First page reference -- the page-destination target.
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageRef := kids[0]

	// Minimal structure tree so a structure-element destination target
	// exists (the root Document element).
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	// The outline item carrying the destination under test.
	item := types.Dict{
		"Type":  types.Name("Outline"),
		"Title": types.StringLiteral("Section"),
	}
	switch mode {
	case "page":
		item["Dest"] = types.Array{pageRef, types.Name("XYZ"), types.Integer(56), types.Integer(723), types.Integer(0)}
	case "structelem":
		item["Dest"] = types.Array{*docRef, types.Name("XYZ"), types.Integer(56), types.Integer(723), types.Integer(0)}
	case "goto-page":
		action := types.Dict{
			"Type": types.Name("Action"),
			"S":    types.Name("GoTo"),
			"D":    types.Array{pageRef, types.Name("Fit")},
		}
		actionRef, err := xrt.IndRefForNewObject(action)
		if err != nil {
			return err
		}
		item["A"] = *actionRef
	default:
		return fmt.Errorf("withOutlineDestination: unknown mode %q", mode)
	}
	itemRef, err := xrt.IndRefForNewObject(item)
	if err != nil {
		return err
	}
	outlines := types.Dict{
		"Type":  types.Name("Outlines"),
		"First": *itemRef,
		"Last":  *itemRef,
		"Count": types.Integer(1),
	}
	outlinesRef, err := xrt.IndRefForNewObject(outlines)
	if err != nil {
		return err
	}
	item["Parent"] = *outlinesRef
	cat["Outlines"] = *outlinesRef

	return writeAndLog(ctx, dst)
}

// withTabs sets /Tabs on the first page to tabs ("S", "R", "C"); an
// empty tabs string omits /Tabs entirely (the missing-/Tabs case).
// When annot is true a Link annotation is added to the page, which is
// what makes the /Tabs requirement apply (UA-08-001 only binds to
// pages that carry an annotation). UA-08-001 passes only on "S".
func withTabs(dst, tabs string, annot bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	if tabs != "" {
		pageDict["Tabs"] = types.Name(tabs)
	}
	if annot {
		link := types.Dict{
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Link"),
			"Rect":    types.Array{types.Integer(10), types.Integer(10), types.Integer(50), types.Integer(50)},
			"P":       kids[0],
		}
		linkRef, err := xrt.IndRefForNewObject(link)
		if err != nil {
			return err
		}
		pageDict["Annots"] = types.Array{*linkRef}
	}
	return writeAndLog(ctx, dst)
}

// withDocumentNamespaceUA2 writes a minimal PDF/UA-2 document whose
// root Document structure element declares the supplied namespace
// URI via /NS. UA-01-008 passes when nsURI is the PDF 2.0
// namespace (http://iso.org/pdf2/ssn) and fails for any other URI.
func withDocumentNamespaceUA2(dst, nsURI string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 7 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /NS 6 0 R >>\nendobj\n")
	off6 := offset()
	fmt.Fprintf(&buf,
		"6 0 obj\n<< /Type /Namespace /NS (%s) >>\nendobj\n", nsURI)
	off7 := offset()
	fmt.Fprintf(&buf,
		"7 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 8\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 8 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withOCASUA2 writes a minimal PDF/UA-2 document with an
// OCProperties/D configuration dictionary that carries an /AS
// entry. UA-20-002 fires on this layout.
func withOCASUA2(dst string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /OCProperties << /OCGs [4 0 R] /D << /Name (Default) /AS [<< /Event /View /Category [/View] /OCGs [4 0 R] >>] >> >> /Metadata 5 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /OCG /Name (Layer 1) >>\nendobj\n")
	off5 := offset()
	fmt.Fprintf(&buf,
		"5 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withCatalogLangUA2 writes a minimal PDF/UA-2 document whose catalog carries
// a /Lang entry set to lang. Passing an empty string emits /Lang () -- the
// present-but-empty case UA-11-001 must reject under PDF/UA-2 (§8.4.4).
// Hand-assembled so the exact (possibly empty) /Lang string survives.
func withCatalogLangUA2(dst, lang string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	fmt.Fprintf(&buf, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R /Lang (%s) /Metadata 5 0 R >>\nendobj\n", lang)
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off5 := offset()
	fmt.Fprintf(&buf, "5 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	// object 4 is unused; emit a free entry so the table stays valid.
	for i, o := range []int{off1, off2, off3, -1, off5} {
		if i == 3 {
			buf.WriteString("0000000000 65535 f \n")
			continue
		}
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withOCConfig writes a tagged PDF declaring pdfuaid:part `part` whose
// /OCProperties default config /D optionally carries a /Name (dNamed), plus an
// optional /Configs array. configNames is nil for no /Configs array, otherwise
// one entry per /Configs member giving whether it carries a /Name. Exercises
// the /Name requirement on optional-content configurations: unconditional for
// PDF/UA-1 §7.10-1, and for PDF/UA-2 §8.7-1 conditional on a non-empty /Configs
// array. Hand-assembled so the exact /OCProperties layout survives.
func withOCConfig(dst string, part int, dNamed bool, configNames []bool) error {
	xmp := []byte(fmt.Sprintf(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>%d</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`, part))

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }
	var offs []int

	dName := ""
	if dNamed {
		dName = " /Name (Default)"
	}
	// /Configs entries occupy object numbers 6.. ; build the array text.
	configsEntry := ""
	if configNames != nil {
		refs := ""
		for i := range configNames {
			refs += fmt.Sprintf("%d 0 R ", 6+i)
		}
		configsEntry = fmt.Sprintf(" /Configs [%s]", refs[:len(refs)-1])
	}
	offs = append(offs, offset())
	fmt.Fprintf(&buf,
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R /OCProperties << /OCGs [4 0 R] /D <<%s /Order [4 0 R] >>%s >> /Metadata 5 0 R /MarkInfo << /Marked true >> >>\nendobj\n",
		dName, configsEntry)
	offs = append(offs, offset())
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	offs = append(offs, offset())
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	offs = append(offs, offset())
	buf.WriteString("4 0 obj\n<< /Type /OCG /Name (Layer 1) >>\nendobj\n")
	offs = append(offs, offset())
	fmt.Fprintf(&buf,
		"5 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")
	for i, named := range configNames {
		cfgName := ""
		if named {
			cfgName = fmt.Sprintf(" /Name (Config %d)", i+1)
		}
		offs = append(offs, offset())
		fmt.Fprintf(&buf, "%d 0 obj\n<<%s /OCGs [4 0 R] >>\nendobj\n", 6+i, cfgName)
	}

	xrefOff := offset()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offs)+1)
	for _, o := range offs {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(offs)+1)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withHeadingUA2 writes a minimal PDF/UA-2 document whose
// structure tree is Document → <headingType>. Used by UA-14-007
// to drive the untyped-H failure path; passing "H1" instead would
// produce the passing variant.
func withHeadingUA2(dst, headingType string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 7 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")
	off6 := offset()
	fmt.Fprintf(&buf,
		"6 0 obj\n<< /Type /StructElem /S /%s /P 5 0 R /Pg 3 0 R >>\nendobj\n", headingType)
	off7 := offset()
	fmt.Fprintf(&buf,
		"7 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 8\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 8 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withDeprecatedAnnotUA2 writes a minimal PDF/UA-2 document with
// a single annotation of the supplied (deprecated) subtype on
// page 1. Used by UA-28-009.
func withDeprecatedAnnotUA2(dst, subtype string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /Metadata 5 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> /Annots [4 0 R] >>\nendobj\n")
	off4 := offset()
	fmt.Fprintf(&buf,
		"4 0 obj\n<< /Type /Annot /Subtype /%s /Rect [50 50 100 100] >>\nendobj\n", subtype)
	off5 := offset()
	fmt.Fprintf(&buf,
		"5 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withXFAUA2 writes a minimal PDF/UA-2 document whose Catalog
// /AcroForm carries an /XFA entry, exercising UA-28-010. The XFA
// payload is a one-byte stub; the check only inspects the
// presence of the entry, not its contents.
func withXFAUA2(dst string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R /Metadata 5 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Fields [] /XFA (stub) >>\nendobj\n")
	off5 := offset()
	fmt.Fprintf(&buf,
		"5 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withNoteUA2 writes a minimal PDF/UA-2 document with a Document
// → Note structure tree where Note lives in the default PDF
// structure namespace. UA-14-009 fires on this fixture; an
// equivalent FENote-based document would pass.
func withNoteUA2(dst string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 7 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> >>\nendobj\n")
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")
	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /Note /P 5 0 R /Pg 3 0 R /ID (n1) >>\nendobj\n")
	off7 := offset()
	fmt.Fprintf(&buf,
		"7 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 8\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 8 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withAsideRolemappedToNoteUA2 writes a PDF/UA-2 document whose
// structure tree is Document → Aside, where the Aside element declares
// the PDF 2.0 standard structure namespace via /NS, and the
// StructTreeRoot carries a global /RoleMap that maps Aside → Note (the
// pattern producers use so legacy PDF 1.7 viewers, which have no Aside,
// still render something). UA-14-009 must not flag this as a forbidden
// Note: the declared 2.0 type is authoritative and the role map is only
// a compatibility hint (ISO 14289-2 §8.2.5.14 Note 1).
func withAsideRolemappedToNoteUA2(dst string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /StructTreeRoot 4 0 R /MarkInfo << /Marked true >> /Metadata 7 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>\nendobj\n")
	off4 := offset()
	buf.WriteString("4 0 obj\n<< /Type /StructTreeRoot /K 5 0 R /ParentTree << /Nums [] >> /RoleMap << /Aside /Note >> >>\nendobj\n")
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /StructElem /S /Document /P 4 0 R /K 6 0 R >>\nendobj\n")
	off6 := offset()
	buf.WriteString("6 0 obj\n<< /Type /StructElem /S /Aside /P 5 0 R /Pg 3 0 R /NS 8 0 R >>\nendobj\n")
	off7 := offset()
	fmt.Fprintf(&buf,
		"7 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")
	off8 := offset()
	buf.WriteString("8 0 obj\n<< /Type /Namespace /NS (http://iso.org/pdf2/ssn) >>\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 9\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5, off6, off7, off8} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 9 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withLinkAnnotationUA2 writes a minimal PDF/UA-2 document (XMP
// declares pdfuaid:part = 2) with a single Link annotation on the
// page. When contents is empty the /Contents entry is omitted, which
// UA-28-001 reports as an advisory (Info) under UA-2 rather than the
// error it raises under UA-1.
func withLinkAnnotationUA2(dst, contents string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	contentsEntry := ""
	if contents != "" {
		contentsEntry = fmt.Sprintf(" /Contents (%s)", contents)
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /MarkInfo << /Marked true >> /Metadata 5 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> /Annots [4 0 R] >>\nendobj\n")
	off4 := offset()
	fmt.Fprintf(&buf,
		"4 0 obj\n<< /Type /Annot /Subtype /Link /Rect [72 700 200 720]%s >>\nendobj\n", contentsEntry)
	off5 := offset()
	fmt.Fprintf(&buf,
		"5 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withTabsUA2 writes a minimal PDF/UA-2 document (XMP declares
// pdfuaid:part = 2) with /Tabs set to the supplied value on a
// single page. Used by UA-08-001 to drive the UA-2 branch of the
// check (which accepts S, A, W) without disturbing the legacy
// UA-1 fixtures that withTabs already produces.
func withTabsUA2(dst, tabs string) error {
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/">
      <pdfuaid:part>2</pdfuaid:part>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)

	var buf bytes.Buffer
	buf.WriteString("%PDF-2.0\n%\xff\xff\xff\xff\n")
	offset := func() int { return buf.Len() }

	off1 := offset()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R /Metadata 4 0 R >>\nendobj\n")
	off2 := offset()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := offset()
	fmt.Fprintf(&buf,
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> /Tabs /%s /Annots [5 0 R] >>\nendobj\n",
		tabs)
	off4 := offset()
	fmt.Fprintf(&buf,
		"4 0 obj\n<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n", len(xmp))
	buf.Write(xmp)
	buf.WriteString("\nendstream\nendobj\n")
	// An annotation on the page is what makes the /Tabs requirement
	// apply (ISO 14289-2 §8.9.3.3).
	off5 := offset()
	buf.WriteString("5 0 obj\n<< /Type /Annot /Subtype /Link /Rect [10 10 50 50] >>\nendobj\n")

	xrefOff := offset()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, o := range []int{off1, off2, off3, off4, off5} {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOff)

	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}

// withPermissions writes the base PDF, then re-encrypts it with
// pdfcpu's standard security handler. allowAccess toggles the PDF/UA
// accessibility bit (/P bit 10). When false, every permission is
// cleared so the resulting fixture exercises the UA-26-001 failure
// path. Encryption uses an empty user password (pdfdisassembler opens
// such documents without prompting) and a fixed owner password.
func withPermissions(dst string, allowAccess bool) error {
	// pdfcpu's EncryptFile reads from inFile, so we first materialise a
	// fresh copy of the base PDF at dst and then encrypt it in place.
	if err := copyBase(dst); err != nil {
		return err
	}
	conf := pdfmodel.NewDefaultConfiguration()
	conf.UserPW = ""
	conf.OwnerPW = "owner"
	if allowAccess {
		conf.Permissions = pdfmodel.PermissionsAll
	} else {
		conf.Permissions = pdfmodel.PermissionsNone
	}
	if err := api.EncryptFile(dst, dst, conf); err != nil {
		return err
	}
	fmt.Println("encrypted", dst)
	return nil
}

// withMixedHeadings builds Document → [headingTypes...] where each
// entry is an explicit structure type like "H", "H1", "H2". Used by
// UA-14-006 to assemble both the pure-/H<n> fixture and the mixed
// /H + /H<n> failure fixture.
func withMixedHeadings(dst string, headingTypes []string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	var kids types.Array
	for _, t := range headingTypes {
		h := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name(t),
			"P":    *docRef,
			"Pg":   pageRef,
		}
		hRef, err := xrt.IndRefForNewObject(h)
		if err != nil {
			return err
		}
		kids = append(kids, *hRef)
	}
	docElem["K"] = kids
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// formulaSpec captures which of /Alt and /ActualText are present on
// the Formula element and with what value. nil means the entry is
// absent altogether; a non-nil pointer to an empty string means
// the entry is present with an empty PDF string literal.
//
// /Alt and /ActualText are NOT symmetric in PDF/UA-1 §7.7: an empty
// /Alt does not satisfy the accessible-math requirement, but an
// empty /ActualText does (it conveys "render as silence"). The
// veraPDF UA-1 §7.7 corpus encodes this asymmetry as three
// fixtures (pass-c with empty ActualText, fail-b with empty Alt,
// fail-a with neither).
type formulaSpec struct {
	alt        *string
	actualText *string
}

// strPtr returns a pointer to s. Helper for formulaSpec call sites
// so they can be written as a single-line literal.
func strPtr(s string) *string { return &s }

// utf16BEText encodes s as a PDF text string in UTF-16BE with a leading BOM,
// so code points outside PDFDocEncoding (e.g. Private Use Area) survive the
// round-trip. The returned string holds the raw bytes for a StringLiteral.
func utf16BEText(s string) string {
	b := []byte{0xFE, 0xFF}
	for _, u := range utf16.Encode([]rune(s)) {
		b = append(b, byte(u>>8), byte(u))
	}
	return string(b)
}

// withFormula derives a tagged PDF whose structure tree contains a
// single Formula StructElem with the requested /Alt and/or
// /ActualText entries. Mirrors withFigure for the Figure check.
func withFormula(dst string, spec formulaSpec) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formula := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Formula"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	if spec.alt != nil {
		formula["Alt"] = types.StringLiteral(*spec.alt)
	}
	if spec.actualText != nil {
		formula["ActualText"] = types.StringLiteral(*spec.actualText)
	}
	formulaRef, err := xrt.IndRefForNewObject(formula)
	if err != nil {
		return err
	}
	docElem["K"] = *formulaRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withTaggedAnnotation builds Document → <structType> → OBJR → annotation of
// the given subtype, wiring the /ParentTree so the annotation's /StructParent
// (0) resolves to the enclosing structure element. Used by UA-28-013: a Link
// annotation enclosed in a Link element passes; enclosing it in some other
// type (e.g. P) fails.
func withTaggedAnnotation(dst, structType, subtype string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	elem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name(structType), "P": *docRef, "Pg": pageRef}
	elemRef, err := xrt.IndRefForNewObject(elem)
	if err != nil {
		return err
	}
	annot := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name(subtype),
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}
	elem["K"] = *objrRef
	docElem["K"] = *elemRef

	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *elemRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	// Attach the annotation to the first page.
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withFormWidgets builds Document → Form → (OBJR → Widget)×count, wiring each
// widget onto the first page. UA-28-027 (ISO 14289-2 §8.10.1) passes when the
// Form encloses a single widget and fails when it encloses more than one.
func withFormWidgets(dst string, count int) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Form"), "P": *docRef, "Pg": pageRef}
	formRef, err := xrt.IndRefForNewObject(formElem)
	if err != nil {
		return err
	}

	var kids types.Array
	var annots types.Array
	for i := range count {
		annot := types.Dict{
			"Type":         types.Name("Annot"),
			"Subtype":      types.Name("Widget"),
			"FT":           types.Name("Tx"),
			"Rect":         onPageRect(),
			"StructParent": types.Integer(i),
		}
		annotRef, err := xrt.IndRefForNewObject(annot)
		if err != nil {
			return err
		}
		objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
		objrRef, err := xrt.IndRefForNewObject(objr)
		if err != nil {
			return err
		}
		kids = append(kids, *objrRef)
		annots = append(annots, *annotRef)
	}
	formElem["K"] = kids
	docElem["K"] = *formRef

	// One ParentTree entry per widget, all pointing at the single Form.
	var nums types.Array
	for i := range count {
		nums = append(nums, types.Integer(i), *formRef)
	}
	parentTree := types.Dict{"Nums": nums}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = annots

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withFormRole builds Document → <sType> for the UA-28-032 fixtures. When
// sType is "Form" the element is a Form structure element; a non-Form value
// (e.g. "P") yields a document with no Form element at all (the N/A case).
// When role is non-empty the element carries an /A PrintField owner dict with
// that /Role value (Table 348). widgetCount OBJR children, each pointing at a
// /Subtype /Widget annotation, are attached as the element's /K. A Form
// omitting /Role must have exactly one such child (ISO 14289-1 §7.18.4,
// veraPDF UA1:7.18.4-2).
func withFormRole(dst, sType, role string, widgetCount int) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name(sType), "P": *docRef, "Pg": pageRef}
	if role != "" {
		formElem["A"] = types.Dict{"O": types.Name("PrintField"), "Role": types.Name(role)}
	}
	formRef, err := xrt.IndRefForNewObject(formElem)
	if err != nil {
		return err
	}

	var kids types.Array
	var annots types.Array
	for i := range widgetCount {
		annot := types.Dict{
			"Type":         types.Name("Annot"),
			"Subtype":      types.Name("Widget"),
			"FT":           types.Name("Tx"),
			"Rect":         onPageRect(),
			"StructParent": types.Integer(i),
		}
		annotRef, err := xrt.IndRefForNewObject(annot)
		if err != nil {
			return err
		}
		objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
		objrRef, err := xrt.IndRefForNewObject(objr)
		if err != nil {
			return err
		}
		kids = append(kids, *objrRef)
		annots = append(annots, *annotRef)
	}
	if len(kids) > 0 {
		formElem["K"] = kids
	}
	docElem["K"] = *formRef

	var nums types.Array
	for i := range widgetCount {
		nums = append(nums, types.Integer(i), *formRef)
	}
	parentTree := types.Dict{"Nums": nums}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	if len(annots) > 0 {
		pageDict, err := xrt.DereferenceDict(pageRef)
		if err != nil {
			return err
		}
		pageDict["Annots"] = annots
	}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withWidgetAA builds Document → Form → OBJR → Widget where the widget is a
// form field (/FT) carrying an /AA additional-actions dictionary. When
// withContents is true the widget also carries /Contents (UA-28-029 passes);
// otherwise it does not (the failure shape, ISO 14289-2 §8.10.2.3).
func withWidgetAA(dst string, withContents bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Form"), "P": *docRef, "Pg": pageRef}
	formRef, err := xrt.IndRefForNewObject(formElem)
	if err != nil {
		return err
	}

	widget := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name("Widget"),
		"FT":           types.Name("Tx"),
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
		"AA":           types.Dict{"F": types.Dict{"S": types.Name("JavaScript"), "JS": types.StringLiteral("null;")}},
	}
	if withContents {
		widget["Contents"] = types.StringLiteral("Amount field")
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *widgetRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}
	formElem["K"] = *objrRef
	docElem["K"] = *formRef

	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *formRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withPUAContent builds a tagged page whose single glyph decodes (via
// /ToUnicode) to a Unicode value. When pua is true the value is U+E000 (a
// Private Use Area code point), else U+0041 ("A"). contentActualText adds an
// /ActualText property to the marked-content sequence; structAlt adds /Alt to
// the enclosing P structure element. Exercises UA-01-020 (ISO 14289-2 §8.4.3):
// PUA real content needs an /ActualText or /Alt.
func withPUAContent(dst string, pua, contentActualText, structAlt bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	uni := "0041"
	if pua {
		uni = "E000"
	}
	cmap := "/CIDInit /ProcSet findresource begin 12 dict begin begincmap " +
		"1 begincodespacerange <00> <FF> endcodespacerange " +
		"1 beginbfchar <41> <" + uni + "> endbfchar endcmap " +
		"CMapName currentdict /CMap defineresource pop end end"
	touStream := types.StreamDict{Dict: types.Dict{}, Content: []byte(cmap)}
	if err := touStream.Encode(); err != nil {
		return err
	}
	touRef, err := xrt.IndRefForNewObject(touStream)
	if err != nil {
		return err
	}
	font := types.Dict{
		"Type":      types.Name("Font"),
		"Subtype":   types.Name("Type1"),
		"BaseFont":  types.Name("Helvetica"),
		"Encoding":  types.Name("WinAnsiEncoding"),
		"ToUnicode": *touRef,
	}
	fontRef, err := xrt.IndRefForNewObject(font)
	if err != nil {
		return err
	}

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef, "Pg": pageRef, "K": types.Integer(0)}
	if structAlt {
		pElem["Alt"] = types.StringLiteral("Alternative description")
	}
	pRef, err := xrt.IndRefForNewObject(pElem)
	if err != nil {
		return err
	}
	docElem["K"] = *pRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = types.Dict{"Nums": types.Array{types.Integer(0), *pRef}}

	bdc := "/P <</MCID 0>>"
	if contentActualText {
		bdc = "/P <</MCID 0 /ActualText (Real text)>>"
	}
	body := bdc + " BDC BT /F1 12 Tf 72 720 Td (A) Tj ET EMC\n"
	content := types.StreamDict{Dict: types.Dict{}, Content: []byte(body)}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef
	pageDict["Resources"] = types.Dict{"Font": types.Dict{"F1": *fontRef}}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withSignatureGraphic builds Document → Form → (OBJR → Widget, optional
// Figure) where the widget's field type is /Sig when isSig is true (else /Tx).
// hasFigure adds a Figure structure element (the signature appearance graphic);
// figureAlt gives it an /Alt. Exercises UA-28-036 (ISO 14289-2 §8.10.3.5): a
// signature appearance graphic must carry /Alt.
func withSignatureGraphic(dst string, isSig, hasFigure, figureAlt bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Form"), "P": *docRef, "Pg": pageRef}
	formRef, err := xrt.IndRefForNewObject(formElem)
	if err != nil {
		return err
	}

	ft := types.Name("Tx")
	if isSig {
		ft = types.Name("Sig")
	}
	widget := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name("Widget"),
		"FT":           ft,
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *widgetRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}

	formKids := types.Array{*objrRef}
	if hasFigure {
		figure := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Figure"), "P": *formRef, "Pg": pageRef}
		if figureAlt {
			figure["Alt"] = types.StringLiteral("Signature of the approver")
		}
		figRef, err := xrt.IndRefForNewObject(figure)
		if err != nil {
			return err
		}
		formKids = append(formKids, *figRef)
	}
	formElem["K"] = formKids
	docElem["K"] = *formRef

	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *formRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withFieldTU builds Document → Form → OBJR → text-field Widget carrying a /TU
// when hasTU is true. structLang adds /Lang to the Form structure element;
// catalogLang adds a catalog /Lang. Exercises UA-11-009 (ISO 14289-1 §7.2): a
// form field /TU needs a determinable language.
func withFieldTU(dst string, hasTU, structLang, catalogLang bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Form"), "P": *docRef, "Pg": pageRef}
	if structLang {
		formElem["Lang"] = types.StringLiteral("en-US")
	}
	formRef, err := xrt.IndRefForNewObject(formElem)
	if err != nil {
		return err
	}

	widget := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name("Widget"),
		"FT":           types.Name("Tx"),
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
	}
	if hasTU {
		widget["TU"] = types.StringLiteral("firstname")
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *widgetRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}
	formElem["K"] = *objrRef
	docElem["K"] = *formRef

	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *formRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	if catalogLang {
		cat["Lang"] = types.StringLiteral("en-US")
	}
	return writeAndLog(ctx, dst)
}

// withFormLabelWidget builds Document → Form → (optional Lbl, OBJR → Widget)
// where the widget is a form field (/FT). hasLbl adds a Lbl structure element
// as the Form's first child; hasContents adds /Contents to the widget.
// Exercises UA-28-035 (ISO 14289-2 §8.10.2.3): a field widget with neither a
// Lbl label nor /Contents fails.
func withFormLabelWidget(dst string, hasLbl, hasContents bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	formElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Form"), "P": *docRef, "Pg": pageRef}
	formRef, err := xrt.IndRefForNewObject(formElem)
	if err != nil {
		return err
	}

	widget := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name("Widget"),
		"FT":           types.Name("Tx"),
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
	}
	if hasContents {
		widget["Contents"] = types.StringLiteral("Amount field")
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *widgetRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}

	var formKids types.Array
	if hasLbl {
		lbl := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Lbl"), "P": *formRef, "Pg": pageRef}
		lblRef, err := xrt.IndRefForNewObject(lbl)
		if err != nil {
			return err
		}
		formKids = append(formKids, *lblRef)
	}
	formKids = append(formKids, *objrRef)
	formElem["K"] = formKids
	docElem["K"] = *formRef

	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *formRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withLinkTargets builds Document → Link → (OBJR → Link annotation)×2, both
// annotations enclosed by the single Link structure element. When sameTarget is
// true both annotations point at the same destination (UA-28-030 passes);
// otherwise they point at different destinations on the page (the failure
// shape, ISO 14289-2 §8.2.5.20).
func withLinkTargets(dst string, sameTarget bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	linkElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Link"), "P": *docRef, "Pg": pageRef}
	linkRef, err := xrt.IndRefForNewObject(linkElem)
	if err != nil {
		return err
	}

	// Two link annotations. The second targets a different vertical
	// position unless sameTarget is set.
	dest := func(y int) types.Array {
		return types.Array{pageRef, types.Name("XYZ"), types.Integer(0), types.Integer(y), types.Integer(0)}
	}
	secondY := 200
	if sameTarget {
		secondY = 100
	}
	var objrs types.Array
	var annots types.Array
	for i, y := range []int{100, secondY} {
		annot := types.Dict{
			"Type":         types.Name("Annot"),
			"Subtype":      types.Name("Link"),
			"Rect":         onPageRect(),
			"StructParent": types.Integer(i),
			"Dest":         dest(y),
		}
		annotRef, err := xrt.IndRefForNewObject(annot)
		if err != nil {
			return err
		}
		objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
		objrRef, err := xrt.IndRefForNewObject(objr)
		if err != nil {
			return err
		}
		objrs = append(objrs, *objrRef)
		annots = append(annots, *annotRef)
	}
	linkElem["K"] = objrs
	docElem["K"] = *linkRef

	parentTree := types.Dict{"Nums": types.Array{
		types.Integer(0), *linkRef,
		types.Integer(1), *linkRef,
	}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = annots

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withFormXObject adds a form XObject to the first page's /Resources. When
// withRef is true the XObject carries a /Ref reference dictionary, making it a
// reference XObject -- the UA-30-001 failure shape (ISO 14289-1 §7.20). When
// false it is an ordinary form XObject and the check passes.
func withFormXObject(dst string, withRef bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	xobj := types.StreamDict{
		Dict: types.Dict{
			"Type":      types.Name("XObject"),
			"Subtype":   types.Name("Form"),
			"BBox":      types.Array{types.Integer(0), types.Integer(0), types.Integer(100), types.Integer(100)},
			"Resources": types.Dict{},
		},
		Content: []byte("q 1 0 0 1 0 0 cm 0 0 100 100 re f Q\n"),
	}
	if withRef {
		// A /Ref reference dictionary (ISO 32000-1 §8.10.4): the presence of
		// the key is what makes this a reference XObject.
		xobj.Dict["Ref"] = types.Dict{
			"F":    types.StringLiteral("external.pdf"),
			"Page": types.Integer(0),
		}
	}
	if err := xobj.Encode(); err != nil {
		return err
	}
	xobjRef, err := xrt.IndRefForNewObject(xobj)
	if err != nil {
		return err
	}

	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Resources"] = types.Dict{
		"XObject": types.Dict{"Fm0": *xobjRef},
	}
	return writeAndLog(ctx, dst)
}

// withMarkupRC builds a FreeText markup annotation carrying an optional
// rich-text contents /RC and plain /Contents (each omitted when empty).
// Exercises UA-28-034 (ISO 14289-2 §8.9.2.3): when both are present they must
// be textually equivalent.
func withMarkupRC(dst, rc, contents string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("FreeText"),
		"Rect":    onPageRect(),
	}
	if rc != "" {
		annot["RC"] = types.StringLiteral(rc)
	}
	if contents != "" {
		annot["Contents"] = types.StringLiteral(contents)
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}
	return writeAndLog(ctx, dst)
}

// withTextFieldRV builds an AcroForm text field widget carrying an optional
// rich-text value /RV and plain value /V (each omitted when empty). Exercises
// UA-28-033 (ISO 14289-2 §8.10.3.3): a text field with /RV must also have a /V
// that is textually equivalent.
func withTextFieldRV(dst, rv, v string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	widget := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"FT":      types.Name("Tx"),
		"Rect":    onPageRect(),
		"T":       types.StringLiteral("field1"),
	}
	if rv != "" {
		widget["RV"] = types.StringLiteral(rv)
	}
	if v != "" {
		widget["V"] = types.StringLiteral(v)
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["AcroForm"] = types.Dict{"Fields": types.Array{*widgetRef}}
	return writeAndLog(ctx, dst)
}

// withType0Notdef builds a page that shows a single 2-byte code from a
// composite (Type0) font with Identity-H encoding and a CIDFontType2
// descendant. Showing code 0 references CID 0 = glyph 0 = .notdef (UA-10-004,
// ISO 14289-1 §7.21.8 / ISO 14289-2 §8.4.5.9). When cidToGIDStream is true the
// descendant carries a /CIDToGIDMap remapping stream, so glyph 0 cannot be
// confirmed and the check conservatively passes. No real font program is
// embedded: the check inspects the font dictionaries and the shown code only.
func withType0Notdef(dst string, showCode int, cidToGIDStream bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	descriptor := types.Dict{
		"Type":     types.Name("FontDescriptor"),
		"FontName": types.Name("TestFont"),
		"Flags":    types.Integer(4),
	}
	descRef, err := xrt.IndRefForNewObject(descriptor)
	if err != nil {
		return err
	}
	cidFont := types.Dict{
		"Type":     types.Name("Font"),
		"Subtype":  types.Name("CIDFontType2"),
		"BaseFont": types.Name("TestFont"),
		"CIDSystemInfo": types.Dict{
			"Registry":   types.StringLiteral("Adobe"),
			"Ordering":   types.StringLiteral("Identity"),
			"Supplement": types.Integer(0),
		},
		"FontDescriptor": *descRef,
	}
	if cidToGIDStream {
		mapStream := types.StreamDict{Dict: types.Dict{}, Content: []byte{0, 0, 0, 1}}
		if err := mapStream.Encode(); err != nil {
			return err
		}
		mapRef, err := xrt.IndRefForNewObject(mapStream)
		if err != nil {
			return err
		}
		cidFont["CIDToGIDMap"] = *mapRef
	}
	cidRef, err := xrt.IndRefForNewObject(cidFont)
	if err != nil {
		return err
	}
	type0 := types.Dict{
		"Type":            types.Name("Font"),
		"Subtype":         types.Name("Type0"),
		"BaseFont":        types.Name("TestFont"),
		"Encoding":        types.Name("Identity-H"),
		"DescendantFonts": types.Array{*cidRef},
	}
	type0Ref, err := xrt.IndRefForNewObject(type0)
	if err != nil {
		return err
	}

	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Resources"] = types.Dict{"Font": types.Dict{"C0_0": *type0Ref}}
	body := fmt.Sprintf("BT /C0_0 12 Tf <%04X> Tj ET\n", showCode)
	content := types.StreamDict{Dict: types.Dict{}, Content: []byte(body)}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef
	return writeAndLog(ctx, dst)
}

// utf16HexLiteral encodes a Go string as a UTF-16BE PDF text string (with a
// leading byte-order mark) in hex-literal form, so BMP code points outside
// ASCII -- including Private Use Area characters -- round-trip through the
// parser.
func utf16HexLiteral(s string) types.HexLiteral {
	buf := []byte{0xFE, 0xFF}
	for _, r := range s {
		buf = append(buf, byte(r>>8), byte(r))
	}
	return types.NewHexLiteral(buf)
}

// withWidgetName builds a form field widget annotation whose /T (partial field
// name) is name, optionally with a trailing Unicode Private Use Area character
// appended. Exercises UA-01-017 (ISO 14289-2 §8.6): a human-readable text
// string must not use PUA.
func withWidgetName(dst, name string, pua bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	title := name
	if pua {
		title += string(rune(0xE000)) // Unicode Private Use Area code point
	}
	widget := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"FT":      types.Name("Tx"),
		"Rect":    onPageRect(),
		"T":       utf16HexLiteral(title),
	}
	widgetRef, err := xrt.IndRefForNewObject(widget)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*widgetRef}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["AcroForm"] = types.Dict{"Fields": types.Array{*widgetRef}}
	return writeAndLog(ctx, dst)
}

// withSpanLang builds a page whose content stream contains a Span
// marked-content sequence carrying the given alternate-text attribute
// ("ActualText", "Alt" or "E"). langSource controls where (if anywhere) the
// natural language comes from, exercising UA-11-008 (ISO 14289-1 §7.2):
//   - "none":    no /Lang anywhere -> fails
//   - "span":    a /Lang property on the Span itself -> passes
//   - "catalog": a document-wide catalog /Lang -> passes
//   - "struct":  the Span is nested in a tagged P whose structure element
//     carries /Lang, inherited by the Span -> passes
func withSpanLang(dst, attr, langSource string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}

	var body string
	switch langSource {
	case "span":
		body = "/Span <</" + attr + " (x) /Lang (en-US)>> BDC 10 10 20 20 re f EMC\n"
	case "struct":
		body = "/P <</MCID 0>> BDC /Span <</" + attr + " (x)>> BDC 10 10 20 20 re f EMC EMC\n"
		// Minimal structure tree: Document -> P(/Lang) owning MCID 0.
		streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
		streeRef, err := xrt.IndRefForNewObject(streeDict)
		if err != nil {
			return err
		}
		docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
		docRef, err := xrt.IndRefForNewObject(docElem)
		if err != nil {
			return err
		}
		pElem := types.Dict{
			"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef,
			"Pg": pageRef, "Lang": types.StringLiteral("en-US"), "K": types.Integer(0),
		}
		pRef, err := xrt.IndRefForNewObject(pElem)
		if err != nil {
			return err
		}
		docElem["K"] = *pRef
		streeDict["K"] = *docRef
		streeDict["ParentTree"] = types.Dict{"Nums": types.Array{types.Integer(0), *pRef}}
		cat["StructTreeRoot"] = *streeRef
	default: // "none", "catalog"
		body = "/Span <</" + attr + " (x)>> BDC 10 10 20 20 re f EMC\n"
	}
	if langSource == "catalog" {
		cat["Lang"] = types.StringLiteral("en-US")
	}

	content := types.StreamDict{Dict: types.Dict{}, Content: []byte(body)}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef
	return writeAndLog(ctx, dst)
}

// withMCNesting sets a page's content stream to exercise UA-14-010's
// Artifact/tagged marked-content nesting rule (ISO 14289-1 §7.1). mode selects
// the shape: "artifact-in-tagged" opens an Artifact sequence inside a tagged
// Span (fails 7.1-1), "tagged-in-artifact" opens a tagged Note inside an
// Artifact (fails 7.1-2), and "siblings" keeps the two as siblings (passes).
func withMCNesting(dst, mode string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	var body string
	switch mode {
	case "artifact-in-tagged":
		body = "/Span <</MCID 0>> BDC /Artifact <</Type /Pagination>> BDC 10 10 50 50 re f EMC EMC\n"
	case "tagged-in-artifact":
		body = "/Artifact <</Type /Pagination>> BDC /Note <</MCID 0>> BDC 10 10 50 50 re f EMC EMC\n"
	case "siblings":
		body = "/Artifact <</Type /Pagination>> BDC 10 10 50 50 re f EMC /Span <</MCID 0>> BDC 60 10 50 50 re f EMC\n"
	default:
		return fmt.Errorf("withMCNesting: unknown mode %q", mode)
	}

	content := types.StreamDict{Dict: types.Dict{}, Content: []byte(body)}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withMarkedFormXObject builds a page that draws a single form XObject via Do
// refCount times. When hasMCID is true the XObject's content stream carries a
// BDC ... /MCID marked-content sequence, so its content participates in the
// structure tree. UA-30-002 (ISO 14289-1 §7.20) fails when a marked-content
// XObject is drawn more than once; a plain (unmarked) XObject may be drawn any
// number of times.
func withMarkedFormXObject(dst string, hasMCID bool, refCount int) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	inner := "0 0 100 100 re f"
	if hasMCID {
		inner = "/Span <</MCID 0>> BDC 0 0 100 100 re f EMC"
	}
	xobj := types.StreamDict{
		Dict: types.Dict{
			"Type":      types.Name("XObject"),
			"Subtype":   types.Name("Form"),
			"BBox":      types.Array{types.Integer(0), types.Integer(0), types.Integer(100), types.Integer(100)},
			"Resources": types.Dict{},
		},
		Content: []byte("q 1 0 0 1 0 0 cm " + inner + " Q\n"),
	}
	if err := xobj.Encode(); err != nil {
		return err
	}
	xobjRef, err := xrt.IndRefForNewObject(xobj)
	if err != nil {
		return err
	}

	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Resources"] = types.Dict{
		"XObject": types.Dict{"Fm0": *xobjRef},
	}

	var body []byte
	for range refCount {
		body = append(body, []byte("/Fm0 Do\n")...)
	}
	content := types.StreamDict{Dict: types.Dict{}, Content: body}
	if err := content.Encode(); err != nil {
		return err
	}
	contentRef, err := xrt.IndRefForNewObject(content)
	if err != nil {
		return err
	}
	pageDict["Contents"] = *contentRef
	return writeAndLog(ctx, dst)
}

// withXFA adds an /AcroForm with an /XFA array whose config packet declares
// <dynamicRender>. When dynamic is true the value is "required" (a dynamic XFA
// form -- the UA-28-028 failure shape, ISO 14289-1 §7.15); otherwise it is
// "forbidden" (a static XFA form, which passes).
func withXFA(dst string, dynamic bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	render := "forbidden"
	if dynamic {
		render = "required"
	}
	config := types.StreamDict{
		Dict:    types.Dict{},
		Content: []byte("<config><acrobat><acrobat7><dynamicRender>" + render + "</dynamicRender></acrobat7></acrobat></config>"),
	}
	if err := config.Encode(); err != nil {
		return err
	}
	configRef, err := xrt.IndRefForNewObject(config)
	if err != nil {
		return err
	}

	acroForm := types.Dict{
		"Fields": types.Array{},
		"XFA":    types.Array{types.StringLiteral("config"), *configRef},
	}
	acroRef, err := xrt.IndRefForNewObject(acroForm)
	if err != nil {
		return err
	}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["AcroForm"] = *acroRef
	return writeAndLog(ctx, dst)
}

// withZeroSizeWidget builds a Widget annotation whose /Rect has zero width and
// height. When tagged is true the widget is enclosed in a P structure element
// (via OBJR + ParentTree) -- the UA-28-018 failure shape (a zero-size widget in
// logical structure that is not an artifact). When false the widget carries no
// /StructParent and is not in the tree -- the compliant case.
func withZeroSizeWidget(dst string, tagged bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	zeroRect := types.Array{types.Integer(100), types.Integer(100), types.Integer(100), types.Integer(100)}
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"Rect":    zeroRect,
	}

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}

	if tagged {
		annot["StructParent"] = types.Integer(0)
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}

	if tagged {
		elem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef, "Pg": pageRef}
		elemRef, err := xrt.IndRefForNewObject(elem)
		if err != nil {
			return err
		}
		objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
		objrRef, err := xrt.IndRefForNewObject(objr)
		if err != nil {
			return err
		}
		elem["K"] = *objrRef
		docElem["K"] = *elemRef
		parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *elemRef}}
		ptRef, err := xrt.IndRefForNewObject(parentTree)
		if err != nil {
			return err
		}
		streeDict["ParentTree"] = *ptRef
	} else {
		streeDict["ParentTree"] = types.Dict{"Nums": types.Array{}}
	}
	streeDict["K"] = *docRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withFlaggedAnnotation builds a Text annotation carrying the given /F
// annotation flags. When tagged is true the annotation is enclosed in a P
// structure element (via OBJR + ParentTree) -- the UA-28-020 failure shape for
// an invisible/no-view annotation that is not an artifact. When false the
// annotation carries no /StructParent and is not in the tree (the compliant
// case).
func withFlaggedAnnotation(dst string, flags int, tagged bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	annot := types.Dict{
		"Type":     types.Name("Annot"),
		"Subtype":  types.Name("Text"),
		"Rect":     onPageRect(),
		"Contents": types.StringLiteral("comment"),
		"F":        types.Integer(flags),
	}

	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}

	if tagged {
		annot["StructParent"] = types.Integer(0)
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}

	if tagged {
		elem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef, "Pg": pageRef}
		elemRef, err := xrt.IndRefForNewObject(elem)
		if err != nil {
			return err
		}
		objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
		objrRef, err := xrt.IndRefForNewObject(objr)
		if err != nil {
			return err
		}
		elem["K"] = *objrRef
		docElem["K"] = *elemRef
		parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *elemRef}}
		ptRef, err := xrt.IndRefForNewObject(parentTree)
		if err != nil {
			return err
		}
		streeDict["ParentTree"] = *ptRef
	} else {
		streeDict["ParentTree"] = types.Dict{"Nums": types.Array{}}
	}
	streeDict["K"] = *docRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withTitleAgreement writes a PDF that carries BOTH a DocumentInfo
// /Title and an XMP dc:title. The two strings can be equal (the
// UA-06-005 passing case) or distinct (the failing case). Implemented
// as a single pass over the base PDF -- using AddPropertiesFile +
// withMetadataStream sequentially would not stack because each builder
// reads from basePath, not from the previous step's output.
func withTitleAgreement(dst, infoTitle, xmpTitle string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	// DocumentInfo /Title: create an /Info dict and link it from the
	// trailer; that is how pdfcpu materialises subsequent /Title
	// lookups via DocInfo().
	infoDict := types.Dict{"Title": types.StringLiteral(infoTitle)}
	infoRef, err := xrt.IndRefForNewObject(infoDict)
	if err != nil {
		return err
	}
	xrt.Info = infoRef

	// XMP dc:title -- build the same UA1+dc:title packet shape used by
	// the with-xmp-title fixture, but with the requested title text
	// substituted in.
	xmp := `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/"
        xmlns:dc="http://purl.org/dc/elements/1.1/">
      <pdfuaid:part>1</pdfuaid:part>
      <dc:title>
        <rdf:Alt>
          <rdf:li xml:lang="x-default">` + xmpTitle + `</rdf:li>
        </rdf:Alt>
      </dc:title>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	sd := types.StreamDict{
		Dict: types.Dict{
			"Type":    types.Name("Metadata"),
			"Subtype": types.Name("XML"),
		},
		Content: []byte(xmp),
	}
	if err := sd.Encode(); err != nil {
		return err
	}
	mdRef, err := xrt.IndRefForNewObject(sd)
	if err != nil {
		return err
	}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["Metadata"] = *mdRef
	return writeAndLog(ctx, dst)
}

// withTableRow builds a tagged PDF whose structure tree is Document →
// Table → TR → [rowChildTypes...]. UA-15-004 fails when a TR carries a
// non-cell child (anything other than TH or TD); the fixture varies
// only the rowChildTypes vector.
func withTableRow(dst string, rowChildTypes []string) error {
	return withTablePattern(dst, rowChildTypes, nil)
}

// withFENote builds Document → [P (citation), FENote]. The citation and the
// FENote reference each other through /Ref depending on citationRefsNote and
// noteRefsCitation, and the FENote carries /NoteType when noteType != "".
// Used for UA-19-002 fixtures.
func withFENote(dst string, citationRefsNote, noteRefsCitation bool, noteType string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	citation := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef, "Pg": pageRef}
	citationRef, err := xrt.IndRefForNewObject(citation)
	if err != nil {
		return err
	}
	fenote := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("FENote"), "P": *docRef, "Pg": pageRef}
	fenoteRef, err := xrt.IndRefForNewObject(fenote)
	if err != nil {
		return err
	}
	if citationRefsNote {
		citation["Ref"] = types.Array{*fenoteRef}
	}
	if noteRefsCitation {
		fenote["Ref"] = types.Array{*citationRef}
	}
	if noteType != "" {
		fenote["A"] = types.Dict{"O": types.Name("FENote"), "NoteType": types.Name(noteType)}
	}
	docElem["K"] = types.Array{*citationRef, *fenoteRef}
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withRoleMap builds a minimal tagged document (StructTreeRoot → Document)
// whose StructTreeRoot carries the given classic /RoleMap. Used for
// UA-31-009 circular-mapping fixtures.
func withRoleMap(dst string, roleMap map[string]string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	rm := types.Dict{}
	for k, v := range roleMap {
		rm[k] = types.Name(v)
	}
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot"), "RoleMap": rm}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withTOCItem builds Document → TOC → TOCI. When withRef is true the TOCI
// carries a /Ref array pointing at the Document element (a resolvable
// structure element); otherwise no /Ref is present -- the UA-01-012 failure
// path.
func withTOCItem(dst string, withRef bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	tocElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TOC"), "P": *docRef, "Pg": pageRef}
	tocRef, err := xrt.IndRefForNewObject(tocElem)
	if err != nil {
		return err
	}
	tociElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TOCI"), "P": *tocRef, "Pg": pageRef}
	if withRef {
		tociElem["Ref"] = types.Array{*docRef}
	}
	tociRef, err := xrt.IndRefForNewObject(tociElem)
	if err != nil {
		return err
	}
	tocElem["K"] = *tociRef
	docElem["K"] = *tocRef
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withBadTOC builds Document → TOC → [TOCI, P]. The stray P child violates
// UA1:7.2-27 (a TOC may contain only TOC/TOCI/Caption) -- the UA-27-002
// failure fixture.
func withBadTOC(dst string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	tocElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TOC"), "P": *docRef, "Pg": pageRef}
	tocRef, err := xrt.IndRefForNewObject(tocElem)
	if err != nil {
		return err
	}
	tociElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TOCI"), "P": *tocRef, "Pg": pageRef}
	tociRef, err := xrt.IndRefForNewObject(tociElem)
	if err != nil {
		return err
	}
	strayElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *tocRef, "Pg": pageRef}
	strayRef, err := xrt.IndRefForNewObject(strayElem)
	if err != nil {
		return err
	}
	tocElem["K"] = types.Array{*tociRef, *strayRef}
	docElem["K"] = *tocRef
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// gridCell describes one table cell for withTableGrid.
type gridCell struct {
	kind             string   // "TH" or "TD"
	rowSpan, colSpan int      // 0 means the attribute is omitted (default 1)
	scope            string   // /Scope in /A (Row/Column/Both); "" omits it
	id               string   // top-level /ID; "" omits it
	headers          []string // /Headers array in /A; nil omits it
}

// withTableGrid builds Document → Table → TR* → (TH|TD)*, one TR per row and
// the given cells per row. RowSpan/ColSpan, when > 0, are written to the
// cell's /A attribute-owner dict (/O /Table). Used for UA-15-006 table
// regularity fixtures.
func withTableGrid(dst string, rows [][]gridCell) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	tableElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Table"), "P": *docRef, "Pg": pageRef}
	tableRef, err := xrt.IndRefForNewObject(tableElem)
	if err != nil {
		return err
	}
	var trRefs types.Array
	for _, row := range rows {
		trElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TR"), "P": *tableRef, "Pg": pageRef}
		trRef, err := xrt.IndRefForNewObject(trElem)
		if err != nil {
			return err
		}
		var cellRefs types.Array
		for _, cell := range row {
			cd := types.Dict{"Type": types.Name("StructElem"), "S": types.Name(cell.kind), "P": *trRef, "Pg": pageRef}
			if cell.id != "" {
				cd["ID"] = types.StringLiteral(cell.id)
			}
			attr := types.Dict{"O": types.Name("Table")}
			if cell.rowSpan > 0 {
				attr["RowSpan"] = types.Integer(cell.rowSpan)
			}
			if cell.colSpan > 0 {
				attr["ColSpan"] = types.Integer(cell.colSpan)
			}
			if cell.scope != "" {
				attr["Scope"] = types.Name(cell.scope)
			}
			if cell.headers != nil {
				var hs types.Array
				for _, h := range cell.headers {
					hs = append(hs, types.StringLiteral(h))
				}
				attr["Headers"] = hs
			}
			if len(attr) > 1 {
				cd["A"] = attr
			}
			cref, err := xrt.IndRefForNewObject(cd)
			if err != nil {
				return err
			}
			cellRefs = append(cellRefs, *cref)
		}
		trElem["K"] = cellRefs
		trRefs = append(trRefs, *trRef)
	}
	tableElem["K"] = trRefs
	docElem["K"] = *tableRef
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withGroupedTable builds Document → Table → [THead, TBody×bodies, TFoot],
// each row group holding one TR with one cell. When bodies is 0 the TBody is
// omitted, so the Table has a THead and TFoot but no body -- the UA-15-009
// failure fixture (violates UA1:7.2-13/-14). With bodies >= 1 the table is
// well-formed. Used for the table row-group cardinality check.
func withGroupedTable(dst string, bodies int) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	tableElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Table"), "P": *docRef, "Pg": pageRef}
	tableRef, err := xrt.IndRefForNewObject(tableElem)
	if err != nil {
		return err
	}

	// group builds a row group (THead/TBody/TFoot) with one TR holding one
	// cell of the given kind, and returns its indirect reference.
	group := func(groupType, cellKind string) (*types.IndirectRef, error) {
		gElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name(groupType), "P": *tableRef, "Pg": pageRef}
		gRef, err := xrt.IndRefForNewObject(gElem)
		if err != nil {
			return nil, err
		}
		trElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TR"), "P": *gRef, "Pg": pageRef}
		trRef, err := xrt.IndRefForNewObject(trElem)
		if err != nil {
			return nil, err
		}
		cell := types.Dict{"Type": types.Name("StructElem"), "S": types.Name(cellKind), "P": *trRef, "Pg": pageRef}
		cellRef, err := xrt.IndRefForNewObject(cell)
		if err != nil {
			return nil, err
		}
		trElem["K"] = *cellRef
		gElem["K"] = *trRef
		return gRef, nil
	}

	var kids types.Array
	headRef, err := group("THead", "TH")
	if err != nil {
		return err
	}
	kids = append(kids, *headRef)
	for range bodies {
		bodyRef, err := group("TBody", "TD")
		if err != nil {
			return err
		}
		kids = append(kids, *bodyRef)
	}
	footRef, err := group("TFoot", "TD")
	if err != nil {
		return err
	}
	kids = append(kids, *footRef)

	tableElem["K"] = kids
	docElem["K"] = *tableRef
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withCaptionTable builds Document → Table → [childTypes...], where the
// Table's direct children are the given types. Used for UA-01-011: a
// Caption must be the first or last child of its parent.
func withCaptionTable(dst string, childTypes []string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	tableElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Table"), "P": *docRef, "Pg": pageRef}
	tableRef, err := xrt.IndRefForNewObject(tableElem)
	if err != nil {
		return err
	}
	var kids types.Array
	for _, t := range childTypes {
		child := types.Dict{"Type": types.Name("StructElem"), "S": types.Name(t), "P": *tableRef, "Pg": pageRef}
		ref, err := xrt.IndRefForNewObject(child)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	tableElem["K"] = kids
	docElem["K"] = *tableRef
	streeDict["K"] = *docRef
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withTableHeader builds Document → Table → TR → TH. When scope is
// non-empty, the TH carries /A << /O /Table /Scope <scope> >>. With
// scope == "" the /A entry is omitted entirely -- the UA-15-005
// failure pattern.
func withTableHeader(dst, scope string) error {
	var attr types.Dict
	if scope != "" {
		attr = types.Dict{
			"O":     types.Name("Table"),
			"Scope": types.Name(scope),
		}
	}
	return withTablePattern(dst, []string{"TH"}, map[int]types.Dict{0: attr})
}

// withTablePattern is the shared builder for the table-related
// fixtures. It writes Document → Table → TR → [rowChildTypes...] and
// attaches the per-index /A attribute dict from childAttrs (nil entries
// produce a bare StructElem).
func withTablePattern(dst string, rowChildTypes []string, childAttrs map[int]types.Dict) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	tableElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Table"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	tableRef, err := xrt.IndRefForNewObject(tableElem)
	if err != nil {
		return err
	}
	trElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("TR"),
		"P":    *tableRef,
		"Pg":   pageRef,
	}
	trRef, err := xrt.IndRefForNewObject(trElem)
	if err != nil {
		return err
	}

	var kids types.Array
	for i, t := range rowChildTypes {
		child := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name(t),
			"P":    *trRef,
			"Pg":   pageRef,
		}
		if attr, ok := childAttrs[i]; ok && attr != nil {
			child["A"] = attr
		}
		ref, err := xrt.IndRefForNewObject(child)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	trElem["K"] = kids
	tableElem["K"] = *trRef
	docElem["K"] = *tableRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withListItem builds Document → L → LI → [itemChildTypes...]. UA-16-002
// fails when the LI has no LBody among its direct children.
func withListItem(dst string, itemChildTypes []string) error {
	return withListPattern(dst, "", itemChildTypes)
}

// withListNumbering builds Document → L → LI → LBody and sets
// /A << /O /List /ListNumbering <numbering> >> on the L when numbering
// is non-empty. The empty case omits /A -- the UA-16-003 warning
// pattern.
func withListNumbering(dst, numbering string) error {
	return withListPattern(dst, numbering, []string{"LBody"})
}

// withListPattern is the shared builder for the list-related fixtures.
// It writes Document → L → LI → [itemChildTypes...] and, when
// numbering is non-empty, attaches /A on the L with /ListNumbering.
func withListPattern(dst, numbering string, itemChildTypes []string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	parentTree := types.Dict{"Nums": types.Array{}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	docElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("Document"),
		"P":    *streeRef,
	}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	lElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("L"),
		"P":    *docRef,
		"Pg":   pageRef,
	}
	if numbering != "" {
		lElem["A"] = types.Dict{
			"O":             types.Name("List"),
			"ListNumbering": types.Name(numbering),
		}
	}
	lRef, err := xrt.IndRefForNewObject(lElem)
	if err != nil {
		return err
	}
	liElem := types.Dict{
		"Type": types.Name("StructElem"),
		"S":    types.Name("LI"),
		"P":    *lRef,
		"Pg":   pageRef,
	}
	liRef, err := xrt.IndRefForNewObject(liElem)
	if err != nil {
		return err
	}

	var kids types.Array
	for _, t := range itemChildTypes {
		child := types.Dict{
			"Type": types.Name("StructElem"),
			"S":    types.Name(t),
			"P":    *liRef,
			"Pg":   pageRef,
		}
		ref, err := xrt.IndRefForNewObject(child)
		if err != nil {
			return err
		}
		kids = append(kids, *ref)
	}
	liElem["K"] = kids
	lElem["K"] = *liRef
	docElem["K"] = *lRef
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withAnnotation attaches a single annotation dict to the first page's
// /Annots array. Used by the per-subtype wrappers below; the base PDF
// itself carries no annotations, so the resulting fixture exposes
// exactly one annotation to the UA-28 walker.
func withAnnotation(dst string, annot types.Dict) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}
	return writeAndLog(ctx, dst)
}

// withAnnotationInAltElement builds a markup annotation (given subtype, no
// /Contents) enclosed in a P structure element carrying the given /Alt (via
// OBJR + ParentTree). Used for the UA-28-025 passing case where the annotation
// has no /Contents but its enclosing element supplies /Alt.
// withAnnotAltContents builds Document → P(/Alt=alt) → OBJR → annotation, where
// the annotation carries /Contents=contents (omitted when empty). Used by
// UA-28-031 (ISO 14289-2 §8.9.4.2): when both the enclosing element's /Alt and
// the annotation's /Contents are present they shall be identical.
func withAnnotAltContents(dst, alt, contents string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	elem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef, "Pg": pageRef, "Alt": types.StringLiteral(alt)}
	elemRef, err := xrt.IndRefForNewObject(elem)
	if err != nil {
		return err
	}
	annot := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name("Link"),
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
	}
	if contents != "" {
		annot["Contents"] = types.StringLiteral(contents)
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}
	elem["K"] = *objrRef
	docElem["K"] = *elemRef
	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *elemRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef
	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}
	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

func withAnnotationInAltElement(dst, subtype, alt string) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable
	streeDict := types.Dict{"Type": types.Name("StructTreeRoot")}
	streeRef, err := xrt.IndRefForNewObject(streeDict)
	if err != nil {
		return err
	}
	docElem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Document"), "P": *streeRef}
	docRef, err := xrt.IndRefForNewObject(docElem)
	if err != nil {
		return err
	}
	pageRef, err := firstPageRef(xrt)
	if err != nil {
		return err
	}
	elem := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("P"), "P": *docRef, "Pg": pageRef, "Alt": types.StringLiteral(alt)}
	elemRef, err := xrt.IndRefForNewObject(elem)
	if err != nil {
		return err
	}
	annot := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name(subtype),
		"Rect":         onPageRect(),
		"StructParent": types.Integer(0),
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}
	objr := types.Dict{"Type": types.Name("OBJR"), "Obj": *annotRef}
	objrRef, err := xrt.IndRefForNewObject(objr)
	if err != nil {
		return err
	}
	elem["K"] = *objrRef
	docElem["K"] = *elemRef
	parentTree := types.Dict{"Nums": types.Array{types.Integer(0), *elemRef}}
	ptRef, err := xrt.IndRefForNewObject(parentTree)
	if err != nil {
		return err
	}
	streeDict["K"] = *docRef
	streeDict["ParentTree"] = *ptRef

	pageDict, err := xrt.DereferenceDict(pageRef)
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}

	cat, err := xrt.Catalog()
	if err != nil {
		return err
	}
	cat["StructTreeRoot"] = *streeRef
	cat["MarkInfo"] = types.Dict{"Marked": types.Boolean(true)}
	return writeAndLog(ctx, dst)
}

// withFileAttachment builds a FileAttachment annotation whose /FS references a
// file specification dictionary. withAFRel toggles the filespec's
// /AFRelationship entry -- the UA-28-024 hinge (ISO 14289-2 §8.9.2.4.10).
func withFileAttachment(dst string, withAFRel bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	efStream := types.Dict{"Type": types.Name("EmbeddedFile")}
	efRef, err := xrt.IndRefForNewObject(efStream)
	if err != nil {
		return err
	}
	filespec := types.Dict{
		"Type": types.Name("Filespec"),
		"F":    types.StringLiteral("attachment.txt"),
		"UF":   types.StringLiteral("attachment.txt"),
		"EF":   types.Dict{"F": *efRef},
	}
	if withAFRel {
		filespec["AFRelationship"] = types.Name("Data")
	}
	fsRef, err := xrt.IndRefForNewObject(filespec)
	if err != nil {
		return err
	}
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("FileAttachment"),
		"Rect":    onPageRect(),
		"FS":      *fsRef,
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}
	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}
	return writeAndLog(ctx, dst)
}

// withMediaClip builds a Screen annotation whose Rendition action points at a
// media rendition (/S /MR) whose /C is a media clip data dictionary (/S /MCD).
// hasCT toggles the /CT entry; correctAlt toggles a valid default /Alt entry
// (empty culture + non-empty text). Used for the UA-28-022/-023 fixtures.
func withMediaClip(dst string, hasCT, correctAlt bool) error {
	ctx, err := api.ReadContextFile(basePath)
	if err != nil {
		return err
	}
	xrt := ctx.XRefTable

	clip := types.Dict{
		"Type": types.Name("MediaClip"),
		"S":    types.Name("MCD"),
		"N":    types.StringLiteral("clip"),
	}
	if hasCT {
		clip["CT"] = types.StringLiteral("video/mp4")
	}
	if correctAlt {
		clip["Alt"] = types.Array{types.StringLiteral(""), types.StringLiteral("A short demo video")}
	}
	clipRef, err := xrt.IndRefForNewObject(clip)
	if err != nil {
		return err
	}
	rendition := types.Dict{
		"Type": types.Name("Rendition"),
		"S":    types.Name("MR"),
		"C":    *clipRef,
	}
	rendRef, err := xrt.IndRefForNewObject(rendition)
	if err != nil {
		return err
	}
	action := types.Dict{
		"Type": types.Name("Action"),
		"S":    types.Name("Rendition"),
		"R":    *rendRef,
	}
	actionRef, err := xrt.IndRefForNewObject(action)
	if err != nil {
		return err
	}
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Screen"),
		"Rect":    onPageRect(),
		"A":       *actionRef,
	}
	annotRef, err := xrt.IndRefForNewObject(annot)
	if err != nil {
		return err
	}

	pagesRef, err := xrt.Pages()
	if err != nil {
		return err
	}
	pagesDict, err := xrt.DereferenceDict(*pagesRef)
	if err != nil {
		return err
	}
	kids, _ := pagesDict["Kids"].(types.Array)
	pageDict, err := xrt.DereferenceDict(kids[0])
	if err != nil {
		return err
	}
	pageDict["Annots"] = types.Array{*annotRef}
	return writeAndLog(ctx, dst)
}

// onPageRect returns a /Rect that sits well inside the blank base PDF's
// 612×792 MediaBox. Used wherever the geometry does not matter and we
// just need the annotation to count as on-page.
func onPageRect() types.Array {
	return types.Array{types.Integer(100), types.Integer(100), types.Integer(200), types.Integer(200)}
}

// withLinkAnnotation builds a Link annotation. When contents is
// non-empty, the annotation carries /Contents; otherwise the entry is
// omitted -- the UA-28-001 failure pattern.
func withLinkAnnotation(dst, contents string) error {
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Link"),
		"Rect":    onPageRect(),
	}
	if contents != "" {
		annot["Contents"] = types.StringLiteral(contents)
	}
	return withAnnotation(dst, annot)
}

// withWidgetAnnotation builds a Widget (form field) annotation. When
// tooltip is non-empty the widget carries /TU; otherwise no tooltip is
// set anywhere -- the UA-28-003 failure pattern.
func withWidgetAnnotation(dst, tooltip string) error {
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"FT":      types.Name("Tx"),
		"Rect":    onPageRect(),
	}
	if tooltip != "" {
		annot["TU"] = types.StringLiteral(tooltip)
	}
	return withAnnotation(dst, annot)
}

// withTextAnnotation builds a Text (sticky-note) annotation, which is
// structure-tree-eligible. withStructParent toggles /StructParent --
// the UA-28-004 hinge.
func withTextAnnotation(dst string, withStructParent bool) error {
	annot := types.Dict{
		"Type":     types.Name("Annot"),
		"Subtype":  types.Name("Text"),
		"Rect":     onPageRect(),
		"Contents": types.StringLiteral("comment"),
	}
	if withStructParent {
		annot["StructParent"] = types.Integer(0)
	}
	return withAnnotation(dst, annot)
}

// withContentsAnnotation builds an annotation of the given subtype (Ink,
// Screen, 3D, RichMedia) with or without a /Contents entry -- the UA-28-011
// hinge: these subtypes must carry a /Contents alternate description.
func withContentsAnnotation(dst, subtype, contents string) error {
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name(subtype),
		"Rect":    onPageRect(),
	}
	if contents != "" {
		annot["Contents"] = types.StringLiteral(contents)
	}
	return withAnnotation(dst, annot)
}

// withNamedStamp builds a rubber-stamp (Stamp) annotation carrying a /Name
// entry but no /Contents -- the UA-28-021 passing case where the stamp's intent
// is conveyed by its standard icon name.
func withNamedStamp(dst, name string) error {
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Stamp"),
		"Rect":    onPageRect(),
		"Name":    types.Name(name),
	}
	return withAnnotation(dst, annot)
}

// withContentsAnnotationLang builds an annotation with /Contents and its own
// /Lang -- the UA-11-005 passing case (the annotation's language is
// determinable from /Lang).
func withContentsAnnotationLang(dst, subtype, contents, lang string) error {
	annot := types.Dict{
		"Type":     types.Name("Annot"),
		"Subtype":  types.Name(subtype),
		"Rect":     onPageRect(),
		"Contents": types.StringLiteral(contents),
		"Lang":     types.StringLiteral(lang),
	}
	return withAnnotation(dst, annot)
}

// withArtifactAnnotation builds a page-furniture annotation (Watermark,
// PrinterMark, TrapNet). withStructParent toggles /StructParent -- the
// UA-28-006 hinge: artifact subtypes must NOT carry one.
func withArtifactAnnotation(dst, subtype string, withStructParent bool) error {
	annot := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name(subtype),
		"Rect":    onPageRect(),
	}
	if withStructParent {
		annot["StructParent"] = types.Integer(0)
	}
	return withAnnotation(dst, annot)
}

// chdirRepoRoot walks up from the current working directory to find the
// pdfa11y repo root and changes to it. We look for the internal/checks
// directory rather than go.mod, because genfixtures now lives in its own
// sub-module -- a go.mod-based search would stop at cmd/genfixtures
// instead of climbing to the parent.
func chdirRepoRoot() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	for dir := cwd; ; {
		if info, err := os.Stat(filepath.Join(dir, "internal", "checks")); err == nil && info.IsDir() {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("no pdfa11y repo root (internal/checks) above %s", cwd)
		}
		dir = parent
	}
}
