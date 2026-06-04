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
