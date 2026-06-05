// Package model defines the backend-agnostic API that checks operate on.
// The current implementation lives in internal/pdf and is built on top of
// pdfcpu; a slimmer parser (e.g. for WASM bundle-size reasons) can be
// dropped in by satisfying these interfaces without touching the checks.
package model

// Document is the unit of analysis. Implementations are stateful: they
// own the parsed PDF data structures and resolve indirect references on
// demand. Checks should never depend on a concrete implementation type.
type Document interface {
	// Catalog returns the document catalog dictionary.
	Catalog() (Dict, error)

	// DocInfo returns the standard /Info dictionary entries as a value
	// snapshot. Missing entries are returned as zero values.
	DocInfo() DocInfo

	// DereferenceDict resolves obj (possibly an indirect reference) to
	// a dictionary. Returns an error if obj is missing or not a dict.
	DereferenceDict(obj Object) (Dict, error)

	// DereferenceBoolean resolves obj to a boolean value. Returns an
	// error if obj is missing or not a boolean.
	DereferenceBoolean(obj Object) (bool, error)

	// DecodeStream resolves obj to a stream and returns its fully
	// decoded (filters applied) byte content.
	DecodeStream(obj Object) ([]byte, error)

	// StructTreeRoot returns the first structure element below the
	// catalog's /StructTreeRoot (typically /S = "Document"). Returns
	// nil with no error if the document has no structure tree at all,
	// so callers can short-circuit before walking.
	StructTreeRoot() (StructElement, error)

	// Fonts returns one snapshot per top-level font dictionary in the
	// document. CID descendant fonts (CIDFontType0/2) are folded into
	// their Type0 parent and not surfaced separately. Order is
	// undefined.
	Fonts() ([]Font, error)

	// Pages returns one PageReport per page in document order. Each
	// report carries the facts derived from content-stream walking on
	// that page: which fonts are actually referenced, which MCIDs are
	// declared by marked-content, and which content lives outside any
	// marked-content sequence. Backends without a content-stream
	// walker return an empty slice with no error.
	Pages() ([]PageReport, error)

	// PageCount returns the number of page leaves in the document,
	// resolved from the page tree at load time. Cheap to call
	// repeatedly; does not require content-stream walking.
	PageCount() int

	// Annotations returns one Annotation per /Annots entry across
	// every page, in page order, with the fields needed by the
	// MH-28 family of checks already resolved (Subtype, Contents,
	// Tooltip via /Parent chain, /StructParent, /F flags). Backends
	// without annotation support return an empty slice with no error.
	Annotations() ([]Annotation, error)

	// OptionalContentGroups returns one snapshot per OCG dictionary
	// listed in the catalog's /OCProperties/OCGs. Returns an empty
	// slice with no error when the document has no /OCProperties at
	// all (the common case for documents without layers).
	OptionalContentGroups() ([]OptionalContentGroup, error)

	// AcroFormFields walks /AcroForm/Fields and returns one entry per
	// terminal Widget annotation reachable through the field
	// hierarchy. Abstract fields (no widget anywhere below them) are
	// not surfaced because they carry no UI surface to link to the
	// structure tree. Returns an empty slice with no error when the
	// document has no /AcroForm or no /Fields.
	AcroFormFields() ([]AcroFormField, error)

	// Encryption reports the document's security handler state in
	// the subset needed by the MH-26 family of checks. For an
	// unencrypted document the zero value (Encrypted=false) is
	// returned; AllowExtractText / AllowAccessibility are then
	// meaningless. For an encrypted document the flags reflect the
	// permission bits actually granted by /P.
	Encryption() EncryptionInfo
}

// AcroFormField is a value snapshot of one terminal Widget annotation
// reached through a recursive walk of /AcroForm/Fields. Used by
// MH-28-007 to cross-check that every form field surfaced in the
// interactive layer is also referenced from the structure tree.
type AcroFormField struct {
	// FullName is the field's fully-qualified name: the local /T of
	// every ancestor (and self) joined by ".". Empty when nothing
	// along the chain declared /T.
	FullName string

	// StructParent is the value of /StructParent on the widget. -1
	// when the widget has no /StructParent at all -- the MH-28-007
	// failure pattern.
	StructParent int
}

// OptionalContentGroup is a value snapshot of one OCG (PDF layer)
// dictionary. Currently only /Name is surfaced because that is what
// MH-20-001 needs; other entries (/Intent, /Usage) can be added when
// further OCG checks land.
type OptionalContentGroup struct {
	// Name is the value of /Name on the OCG dictionary. Empty when
	// the entry is missing or not a text string.
	Name string
}

// EncryptionInfo reports a document's encryption-permission state.
// PDF /P is a signed integer whose individual bits grant or deny
// specific user actions; we only surface the bits that affect
// accessibility.
type EncryptionInfo struct {
	// Encrypted is true when the document carries an /Encrypt
	// trailer entry. False on plain (non-encrypted) PDFs.
	Encrypted bool

	// AllowExtractText reflects /P bit 5 (mask 16): permits copying
	// or otherwise extracting text and graphics from the document.
	AllowExtractText bool

	// AllowAccessibility reflects /P bit 10 (mask 512): permits
	// extraction of text and graphics for accessibility purposes,
	// in particular for AT consumption. PDF/UA-1 §7.20 requires
	// this bit on every encrypted conforming document.
	AllowAccessibility bool
}

// Annotation is a value snapshot of a single page annotation.
// Subtype-specific fields irrelevant to accessibility checks (Rect,
// Border, AP appearance streams, …) are intentionally not exposed.
type Annotation struct {
	// Page is the 1-based number of the page on which the
	// annotation lives.
	Page int

	// Subtype is the value of /Subtype: "Link", "Widget", "Text",
	// "FreeText", "Highlight", "Stamp", "Popup", "PrinterMark", …
	Subtype string

	// Contents is the value of /Contents, the canonical AT-readable
	// text for the annotation. Decoded through the PDF text-string
	// convention (PDFDocEncoding / UTF-16BE / UTF-8 BOM).
	Contents string

	// Tooltip is the value of /TU resolved with /Parent inheritance
	// (form fields express the tooltip on the field dict, not the
	// individual widget). Empty when no /TU is present anywhere up
	// the chain.
	Tooltip string

	// StructParent is the value of /StructParent (the ParentTree
	// key linking this annotation to its parent structure element).
	// -1 when the annotation has no /StructParent at all.
	StructParent int

	// Hidden is true when bit 2 of /F (Annotation flags) is set:
	// the annotation is suppressed from display and printing. Such
	// annotations are not user-visible and should not be expected
	// to participate in accessibility surfaces.
	Hidden bool

	// NoView is true when bit 5 of /F is set: the annotation is
	// printed but not shown on screen. Treated like Hidden for
	// accessibility purposes (no AT surface).
	NoView bool

	// OffPage is true when the annotation's /Rect lies entirely
	// outside the page's visible box (CropBox if present, else
	// MediaBox). Such annotations cannot be perceived by sighted
	// readers and must therefore also be hidden from assistive
	// technology via the Hidden or NoView flags -- a printed-but-
	// off-page annotation that AT still reads creates a phantom
	// reading-order entry.
	OffPage bool
}

// PageReport bundles the content-stream facts about a single page.
// Fields are independent: a check typically uses one of them.
type PageReport struct {
	// Number is the 1-based page number.
	Number int

	// UsedFonts maps each font resource key referenced by a Tf
	// operator on this page (e.g. "F1") to the resolved Font snapshot
	// for that resource. Resource keys that resolve to no font dict
	// are omitted.
	UsedFonts map[string]Font

	// ContentMCIDs is the set of MCIDs declared by BDC operators on
	// this page (inline /MCID or via /Properties name reference).
	ContentMCIDs map[int]bool

	// StructTreeMCIDs is the set of MCIDs that the structure tree
	// claims for this page (via MCR children — either bare integer
	// K-entries or {Type:MCR, Pg, MCID} dictionaries).
	StructTreeMCIDs map[int]bool

	// UntaggedOps lists the real-content operators that ran while the
	// marked-content stack was empty. "Real content" means text
	// showing (Tj/TJ/'/"), XObject invocation (Do), path painting
	// (S/s/f/F/f*/B/B*/b/b*) and inline-image (EI). The slice is
	// capped at a small implementation-defined number so a single
	// broken page does not flood the report.
	UntaggedOps []UntaggedOp

	// Tabs is the effective value of the page's /Tabs key, resolved
	// with inheritance from /Pages ancestors per ISO 32000-1
	// §7.7.3.4. Empty when no ancestor declared it. Valid PDF/UA
	// value is "S" (Structure order); "R" (Row) and "C" (Column)
	// are legal PDF but disallowed by PDF/UA.
	Tabs string

	// FontCodes maps each font resource key (matching a UsedFonts
	// entry) to the set of character codes actually rendered with
	// that font on this page. Decoding follows the active font's
	// subtype:
	//
	//   - Simple fonts (Type1, TrueType, MMType1, Type3): each byte
	//     of the Tj/TJ string is one code in [0, 255].
	//   - Composite fonts (Type0) with Identity-H/V: each pair of
	//     bytes is one 16-bit CID. Other CMaps are approximated as
	//     Identity-H -- a proper implementation would walk the
	//     encoded CMap, which is a future refinement.
	FontCodes map[string]map[uint32]bool
}

// UntaggedOp is one occurrence of a real-content operator running
// outside any marked-content sequence.
type UntaggedOp struct {
	// Operator is the keyword (e.g. "Tj", "Do", "f").
	Operator string
	// Offset is the byte position in the (joined) content stream
	// where the operator was emitted. Useful for diagnostics.
	Offset int64
}

// Object is an opaque handle to a PDF object obtained from a Document
// or one of its derived values. Only the Document that produced an
// Object can interpret it; passing an Object from one backend to a
// different backend is a programming error.
type Object any

// Dict represents a PDF dictionary for read-only access by checks.
type Dict interface {
	// Find returns the value for key and whether the key was present.
	Find(key string) (Object, bool)
}

// DocInfo bundles the standard DocumentInfo dictionary entries. Add
// fields here as new checks need them rather than exposing the raw
// /Info dict.
type DocInfo struct {
	Title string
}

// Font is a value snapshot of a single PDF font dictionary, exposing
// the bits needed by PDF/UA conformance checks. New fields can be added
// as more font-related checks land (e.g. CIDToGIDMap, Differences).
type Font struct {
	// Subtype is the value of /Subtype: "Type0", "Type1", "TrueType",
	// "Type3", "MMType1". Composite font internals (CIDFontType0/2)
	// are not surfaced here.
	Subtype string
	// BaseFont is the value of /BaseFont, e.g. "Helvetica" or
	// "ABCDEF+TimesNewRoman" for subset embeddings. Empty for fonts
	// that omit /BaseFont (uncommon and malformed).
	BaseFont string
	// Embedded reports whether a font program is present. For Type0
	// this checks the descendant font's FontDescriptor; for Type3 it
	// is always true (Type3 glyphs live inline in the PDF).
	Embedded bool
	// HasToUnicode reports whether the font dict carries a /ToUnicode
	// CMap. Reporters keep this as a diagnostic alongside the looser
	// HasUnicodeMapping; the actual PDF/UA acceptability lives in
	// HasUnicodeMapping.
	HasToUnicode bool
	// Encoding is the value of /Encoding when it is a Name (e.g.
	// "WinAnsiEncoding", "MacRomanEncoding"). When /Encoding is a
	// dictionary, this holds the dictionary's /BaseEncoding (if a
	// Name) and HasEncodingDifferences reports whether /Differences
	// is also present.
	Encoding                string
	HasEncodingDifferences  bool
	// HasUnicodeMapping reports whether the font satisfies PDF/UA-1
	// §7.21.3.1: a /ToUnicode CMap, or a predefined encoding that
	// implies the Unicode mapping (WinAnsi/MacRoman/MacExpert on
	// non-symbolic simple fonts without a /Differences override).
	HasUnicodeMapping bool
	// IsSymbolic mirrors /FontDescriptor/Flags bit 3 -- the font
	// uses an encoding outside the standard PDF encodings. For
	// symbolic fonts the WinAnsi/MacRoman shortcut does NOT yield
	// a deterministic Unicode mapping: the glyph at byte 0x41 is
	// some symbol, not the letter "A".
	IsSymbolic bool

	// ToUnicodeMappings is the parsed content of the /ToUnicode
	// CMap when one is present: source code → Unicode replacement.
	// nil when the font has no /ToUnicode stream. Empty (zero
	// length) when the CMap is present but declares only a
	// codespace range with no bfchar / bfrange entries -- the F01
	// failure pattern.
	ToUnicodeMappings map[uint32]string

	// ToUnicodeCodeBytes is the byte width declared by the
	// /ToUnicode CMap's begincodespacerange (1 or 2). Used by the
	// content-stream walker to split Tj/TJ bytes into the right
	// number of codes per glyph. Zero when no codespace was
	// declared; the walker then falls back to 2 for Type0 fonts
	// and 1 for everything else.
	ToUnicodeCodeBytes int
}

// StructElement is a single node in the structure tree. Implementations
// resolve indirect references and skip non-structure children (MCR,
// OBJR, integer MCID) so checks see a clean tree of structure types.
//
// Role mapping (custom types declared via /RoleMap on StructTreeRoot)
// is not yet applied; Type returns the raw /S value. Resolving role
// maps is a planned refinement.
type StructElement interface {
	// Type is the value of /S, e.g. "Document", "Sect", "P", "H1",
	// "Figure", "Table". Empty if /S is absent (rare and malformed).
	Type() string

	// Children returns the StructElement children of this node, with
	// indirect references resolved. MCR / OBJR / integer entries in
	// /K are filtered out -- checks that need them must walk the raw
	// PDF object themselves.
	Children() []StructElement

	// Attr returns a top-level attribute (Alt, ActualText, E, Lang …)
	// as a string. Indirect refs are dereferenced; non-string and
	// missing values yield the empty string.
	Attr(name string) string

	// Attribute returns a value from the structure element's /A
	// attribute owner dictionary. PDF/UA structure attributes such as
	// /ListNumbering (on L), /Scope (on TH), /RowSpan, /ColSpan,
	// /Headers and friends live there rather than on the top-level
	// element dict.
	//
	// /A may be either a single attribute-owner dictionary or an array
	// of them; both shapes are searched in order. The returned value
	// is the attribute's PDF Name as a string ("Decimal", "Row",
	// "Both", …). Numeric and other non-name attribute values are not
	// returned by this accessor and yield "". Add a typed accessor
	// per data shape when a check needs it.
	Attribute(name string) string

	// Page returns the 1-based page number this structure element
	// belongs to, resolved from its /Pg entry. Returns 0 when the
	// element carries no /Pg (some authors only set it on leaves,
	// others omit it entirely) or when the referenced page is not
	// found in the page tree.
	Page() int
}
