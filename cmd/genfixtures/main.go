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
	if err := withMetadataStream("internal/checks/metadata/testdata/with-pdfua-id.pdf",
		xmpUA1); err != nil {
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
	if err := withUntaggedContent("internal/checks/structure/testdata/untagged-content.pdf"); err != nil {
		return err
	}
	if err := withFigure("internal/checks/graphics/testdata/figure-with-alt.pdf",
		"Sunset over the mountains"); err != nil {
		return err
	}
	if err := withFigure("internal/checks/graphics/testdata/figure-no-alt.pdf",
		""); err != nil {
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

	// Annotation fixtures: one passing + one failing pair per MH-28 check.
	if err := withLinkAnnotation("internal/checks/annotations/testdata/link-with-contents.pdf",
		"Project documentation"); err != nil {
		return err
	}
	if err := withLinkAnnotation("internal/checks/annotations/testdata/link-no-contents.pdf",
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
	if err := withOffPageAnnotation("internal/checks/annotations/testdata/offpage-hidden.pdf",
		true); err != nil {
		return err
	}
	if err := withOffPageAnnotation("internal/checks/annotations/testdata/offpage-visible.pdf",
		false); err != nil {
		return err
	}

	// MH-15-004 / MH-15-005: Table row child types and TH /Scope.
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

	// MH-16-002 / MH-16-003: List item children and /ListNumbering.
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

	// MH-14-006: heading style consistency (no /H + /H<n> mix).
	if err := withMixedHeadings("internal/checks/headings/testdata/heading-style-hn-only.pdf",
		[]string{"H1", "H2"}); err != nil {
		return err
	}
	if err := withMixedHeadings("internal/checks/headings/testdata/heading-style-mixed.pdf",
		[]string{"H", "H1"}); err != nil {
		return err
	}

	// MH-17-001: Formula has /Alt or /ActualText.
	if err := withFormula("internal/checks/graphics/testdata/formula-with-alt.pdf",
		"Pythagorean theorem"); err != nil {
		return err
	}
	if err := withFormula("internal/checks/graphics/testdata/formula-no-alt.pdf",
		""); err != nil {
		return err
	}

	// MH-06-005: DocumentInfo /Title and XMP dc:title agree.
	if err := withTitleAgreement("internal/checks/metadata/testdata/title-agreement-ok.pdf",
		"Sample", "Sample"); err != nil {
		return err
	}
	if err := withTitleAgreement("internal/checks/metadata/testdata/title-agreement-mismatch.pdf",
		"Sample", "Different"); err != nil {
		return err
	}

	// MH-20-001: OCG /Name.
	if err := withOCG("internal/checks/optionalcontent/testdata/ocg-with-name.pdf",
		"Drawing geometry"); err != nil {
		return err
	}
	if err := withOCG("internal/checks/optionalcontent/testdata/ocg-no-name.pdf",
		""); err != nil {
		return err
	}

	// MH-27-001: /Outlines on > 21-page documents.
	if err := withOutlines("internal/checks/navigation/testdata/outlines-present.pdf",
		22, true); err != nil {
		return err
	}
	if err := withOutlines("internal/checks/navigation/testdata/outlines-missing.pdf",
		22, false); err != nil {
		return err
	}

	// MH-08-001: page /Tabs = S.
	if err := withTabs("internal/checks/taborder/testdata/tabs-s.pdf",
		"S"); err != nil {
		return err
	}
	if err := withTabs("internal/checks/taborder/testdata/tabs-r.pdf",
		"R"); err != nil {
		return err
	}

	// MH-26-001: encryption permits accessibility extraction.
	if err := withPermissions("internal/checks/security/testdata/encrypted-allow-access.pdf",
		true); err != nil {
		return err
	}
	if err := withPermissions("internal/checks/security/testdata/encrypted-block-access.pdf",
		false); err != nil {
		return err
	}

	// MH-28-007: AcroForm widget linked from the structure tree.
	if err := withAcroFormField("internal/checks/annotations/testdata/acroform-linked.pdf",
		true); err != nil {
		return err
	}
	if err := withAcroFormField("internal/checks/annotations/testdata/acroform-orphan.pdf",
		false); err != nil {
		return err
	}

	// MH-11-002: per-element /Lang coverage when Catalog /Lang is absent.
	if err := withLangCoverage("internal/checks/language/testdata/lang-coverage-ok.pdf",
		"en-US"); err != nil {
		return err
	}
	if err := withLangCoverage("internal/checks/language/testdata/lang-coverage-missing.pdf",
		""); err != nil {
		return err
	}

	// MH-19-001: Note carries /ID; /Ref targets resolve to /ID-bearing
	// structure elements.
	if err := withNote("internal/checks/notes/testdata/note-with-id.pdf",
		"n1"); err != nil {
		return err
	}
	if err := withNote("internal/checks/notes/testdata/note-no-id.pdf",
		""); err != nil {
		return err
	}
	if err := withReferenceToBareP("internal/checks/notes/testdata/ref-unresolved.pdf"); err != nil {
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
		cmap := types.StreamDict{
			Dict: types.Dict{},
			Content: []byte("/CIDInit /ProcSet findresource begin\n" +
				"12 dict begin\nbegincmap\nendcmap\nend\nend\n"),
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

// withRoleMappedHeading writes a tagged PDF that uses a custom structure
// type "MyHeading" role-mapped to H1, followed by H3. With role-map
// resolution working, the structure should be read as H1 -> H3 and
// trigger MH-14-003 (level skip). Without resolution the custom type
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
// fixtures for MH-14-003.
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

// withFigure derives a tagged PDF whose structure tree contains a single
// Figure StructElem. If alt is non-empty, the Figure carries an /Alt
// entry; otherwise /Alt is omitted, producing a known-failing fixture
// for MH-13-004. All other fields match the minimal scaffolding from
// withStructTree.
func withFigure(dst, alt string) error {
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
	if alt != "" {
		figElem["Alt"] = types.StringLiteral(alt)
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

// withNote builds Document → Note. When id is non-empty the Note
// carries /ID; otherwise the entry is omitted -- the MH-19-001
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

// withReferenceToBareP builds Document → [Note(/ID="n1"), P, Reference].
// The Reference's /Ref array points at the P (which carries no /ID),
// so MH-19-001's half B emits a finding while the Note itself passes.
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
// (MH-11-002 passes); empty lang omits the entry and the P inherits
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
// widget carries /StructParent (the MH-28-007 passing case);
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
// MH-20-001 failure pattern.
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
// MH-27-001 threshold (21). When withTree is true, the catalog carries
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
// page-count threshold MH-27-001 enforces.
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

// withTabs sets /Tabs on the first page to tabs ("S", "R", "C").
// MH-08-001 passes only on "S".
func withTabs(dst, tabs string) error {
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
	pageDict["Tabs"] = types.Name(tabs)
	return writeAndLog(ctx, dst)
}

// withPermissions writes the base PDF, then re-encrypts it with
// pdfcpu's standard security handler. allowAccess toggles the PDF/UA
// accessibility bit (/P bit 10). When false, every permission is
// cleared so the resulting fixture exercises the MH-26-001 failure
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
// MH-14-006 to assemble both the pure-/H<n> fixture and the mixed
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

// withFormula derives a tagged PDF whose structure tree contains a
// single Formula StructElem. Non-empty alt produces an /Alt entry;
// empty alt omits both /Alt and /ActualText -- the MH-17-001 failure
// pattern. Mirrors withFigure for the Figure check.
func withFormula(dst, alt string) error {
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
	if alt != "" {
		formula["Alt"] = types.StringLiteral(alt)
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

// withTitleAgreement writes a PDF that carries BOTH a DocumentInfo
// /Title and an XMP dc:title. The two strings can be equal (the
// MH-06-005 passing case) or distinct (the failing case). Implemented
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
// Table → TR → [rowChildTypes...]. MH-15-004 fails when a TR carries a
// non-cell child (anything other than TH or TD); the fixture varies
// only the rowChildTypes vector.
func withTableRow(dst string, rowChildTypes []string) error {
	return withTablePattern(dst, rowChildTypes, nil)
}

// withTableHeader builds Document → Table → TR → TH. When scope is
// non-empty, the TH carries /A << /O /Table /Scope <scope> >>. With
// scope == "" the /A entry is omitted entirely -- the MH-15-005
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

// withListItem builds Document → L → LI → [itemChildTypes...]. MH-16-002
// fails when the LI has no LBody among its direct children.
func withListItem(dst string, itemChildTypes []string) error {
	return withListPattern(dst, "", itemChildTypes)
}

// withListNumbering builds Document → L → LI → LBody and sets
// /A << /O /List /ListNumbering <numbering> >> on the L when numbering
// is non-empty. The empty case omits /A -- the MH-16-003 warning
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
// exactly one annotation to the MH-28 walker.
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

// onPageRect returns a /Rect that sits well inside the blank base PDF's
// 612×792 MediaBox. Used wherever the geometry does not matter and we
// just need the annotation to count as on-page.
func onPageRect() types.Array {
	return types.Array{types.Integer(100), types.Integer(100), types.Integer(200), types.Integer(200)}
}

// withLinkAnnotation builds a Link annotation. When contents is
// non-empty, the annotation carries /Contents; otherwise the entry is
// omitted -- the MH-28-001 failure pattern.
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
// set anywhere -- the MH-28-003 failure pattern.
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
// the MH-28-004 hinge.
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

// withArtifactAnnotation builds a page-furniture annotation (Watermark,
// PrinterMark, TrapNet). withStructParent toggles /StructParent -- the
// MH-28-006 hinge: artifact subtypes must NOT carry one.
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

// withOffPageAnnotation builds a Text annotation whose /Rect sits well
// outside the 612×792 MediaBox. hidden toggles the /F Hidden bit (2);
// without it MH-28-008 fires. /StructParent is set so MH-28-004 stays
// silent and the fixture isolates the off-page concern.
func withOffPageAnnotation(dst string, hidden bool) error {
	annot := types.Dict{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name("Text"),
		"Rect":         types.Array{types.Integer(2000), types.Integer(2000), types.Integer(2100), types.Integer(2100)},
		"Contents":     types.StringLiteral("off-page"),
		"StructParent": types.Integer(0),
	}
	if hidden {
		annot["F"] = types.Integer(2)
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
