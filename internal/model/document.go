// Package model defines the backend-agnostic API that checks operate on.
// The current implementation lives in internal/pdf and is built on top of
// speedata/pdfdisassembler; a slimmer parser (e.g. for WASM bundle-size
// reasons) can be dropped in by satisfying these interfaces without
// touching the checks.
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

	// StructTreeRootKids returns the structure-element children directly
	// under the /StructTreeRoot /K entry, in document order. The second
	// return is false when the document has no /StructTreeRoot dictionary
	// at all -- distinct from a present-but-childless root, which yields an
	// empty slice and true. Used by UA-01-015 to enforce that the root has
	// a single Document element as its only child (ISO 14289-2 §8.2.5.2);
	// StructTreeRoot() cannot serve this because it silently collapses the
	// kid list to the first Document.
	StructTreeRootKids() ([]StructElement, bool)

	// RoleMap returns the classic /RoleMap on the StructTreeRoot: a
	// mapping from a structure type to the type it is role-mapped to.
	// Empty when the document has no /RoleMap.
	RoleMap() map[string]string

	// Namespaces returns the PDF 2.0 namespaces declared on the
	// StructTreeRoot /Namespaces array, each with its URI and its
	// /RoleMapNS role mapping. Empty when there are no declared
	// namespaces.
	Namespaces() []Namespace

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
	// UA-28 family of checks already resolved (Subtype, Contents,
	// Tooltip via /Parent chain, /StructParent, /F flags). Backends
	// without annotation support return an empty slice with no error.
	Annotations() ([]Annotation, error)

	// OptionalContentGroups returns one snapshot per OCG dictionary
	// listed in the catalog's /OCProperties/OCGs. Returns an empty
	// slice with no error when the document has no /OCProperties at
	// all (the common case for documents without layers).
	OptionalContentGroups() ([]OptionalContentGroup, error)

	// OptionalContentConfigs returns one snapshot per optional-content
	// configuration dictionary reachable from /OCProperties: the default
	// config /D, followed by each entry of the /Configs array. Empty when
	// the document has no /OCProperties. Used by UA-20-003 (/Name required)
	// and UA-20-004 (/AS forbidden).
	OptionalContentConfigs() []OptionalContentConfig

	// AcroFormFields walks /AcroForm/Fields and returns one entry per
	// terminal Widget annotation reachable through the field
	// hierarchy. Abstract fields (no widget anywhere below them) are
	// not surfaced because they carry no UI surface to link to the
	// structure tree. Returns an empty slice with no error when the
	// document has no /AcroForm or no /Fields.
	AcroFormFields() ([]AcroFormField, error)

	// AssociatedFiles enumerates filespecs reachable through /AF
	// arrays on the catalog and on every page. Annotation- and
	// XObject-level /AF arrays are not yet walked (planned). Used by
	// UA-12-001 to verify every /AF entry declares /AFRelationship.
	AssociatedFiles() ([]AssociatedFile, error)

	// EmbeddedFileSpecs enumerates the file specification dictionaries
	// held in the catalog's /Names /EmbeddedFiles name tree, in tree
	// order. Used by UA-12-002 to verify that every embedded file
	// carries a /Desc entry (ISO 14289-2 §8.14.1). Returns an empty
	// slice when the document has no /Names /EmbeddedFiles tree.
	EmbeddedFileSpecs() []EmbeddedFileSpec

	// MediaClips enumerates the media clip data dictionaries reachable
	// from the document's Screen annotations (via their Rendition
	// actions and renditions). Used by UA-28-022 / UA-28-023 to verify
	// the /CT and /Alt entries required by ISO 14289-1 §7.18.6.2.
	// Returns an empty slice when the document has no media clips.
	MediaClips() []MediaClip

	// Encryption reports the document's security handler state in
	// the subset needed by the UA-26 family of checks. For an
	// unencrypted document the zero value (Encrypted=false) is
	// returned; AllowExtractText / AllowAccessibility are then
	// meaningless. For an encrypted document the flags reflect the
	// permission bits actually granted by /P.
	Encryption() EncryptionInfo

	// XFA reports the document's XFA form state: whether an /AcroForm/XFA
	// entry is present, and the value of the <dynamicRender> element in
	// its XFA config packet. Used by UA-28-028 to flag dynamic XFA forms
	// (ISO 14289-1 §7.15).
	XFA() XFAInfo

	// FormXObjects returns one snapshot per form XObject in the document,
	// each flagged with whether it carries a /Ref entry (a reference
	// XObject importing page content from an external file, ISO 32000-1
	// §8.10.4). ISO 14289-1 §7.20 forbids reference XObjects outright, and
	// PDF 2.0 deprecates the feature, so PDF/UA-2 documents shall not use
	// it either. Used by UA-30-001. Returns an empty slice when the
	// document has no form XObjects at all.
	FormXObjects() []FormXObject

	// HumanReadableStrings enumerates the document's text strings that are
	// intended to be read by a human: document information (Title, Author,
	// Subject, Keywords), outline titles, form-field names (T, TU),
	// annotation contents, structure-element alternate text (Alt,
	// ActualText, E, T) and optional-content group names. ISO 14289-2 §8.6
	// forbids Unicode PUA in these strings (they carry no readable meaning).
	// Used by UA-01-019. Returns an empty slice when none are present.
	HumanReadableStrings() []TextString

	// IntraDocDestinations enumerates the destinations that point
	// somewhere inside this document: the /Dest of every outline item
	// and Link annotation, a destination-array /OpenAction, and the /D
	// of every /GoTo action reached from those carriers. Remote
	// destinations (/GoToR) and non-destination actions (e.g. /URI) are
	// not enumerated -- they do not target the current document. Used by
	// UA-27-003 to enforce ISO 14289-2 §8.8 (intra-document destinations
	// shall be structure destinations). Returns an empty slice when the
	// document carries no such destinations.
	IntraDocDestinations() []Destination
}

// AssociatedFile is a value snapshot of one filespec entry reached
// via an /AF array on the catalog, a page, or a structure element.
// Used by UA-12-001 to check that each entry declares
// /AFRelationship, and by UA-17-001 to detect MathML-as-AF on
// Formula structure elements.
type AssociatedFile struct {
	// SourcePath identifies where the /AF array sat. "Catalog" for
	// the document-level /AF; "Page N" (1-based) for a per-page /AF;
	// the structure-tree path (e.g. "/Document/Sect/Formula") when
	// the AF was reached via StructElement.AssociatedFiles().
	SourcePath string

	// Filename is /UF if present (Unicode form), else /F. Empty when
	// neither is set -- such filespecs are unusual but not strictly
	// disallowed.
	Filename string

	// Relationship is the /AFRelationship Name value (e.g. "Source",
	// "Data", "Supplement", "Unspecified"). Empty when the entry is
	// absent -- the UA-12-001 failure pattern.
	Relationship string

	// Subtype is the MIME type carried on the embedded-file stream
	// (filespec /EF/F's /Subtype, PDF-encoded as a Name, e.g.
	// "application/mathml+xml"). Empty when /EF or /Subtype is
	// missing. The PDF '#2F' hex escape of the slash is normalised
	// to "/" so callers can compare against literal MIME strings.
	Subtype string

	// Content is the decoded byte content of the embedded-file
	// stream (filters applied). nil when the filespec has no /EF
	// stream or decoding failed. Eager-loaded at collect time --
	// MathML/LaTeX AFs are small enough (<1KB typical) that the
	// alternative of lazy access would add API friction without
	// material savings. UA-17-004 parses these bytes as XML; other
	// callers should treat them as opaque.
	Content []byte
}

// EmbeddedFileSpec is a value snapshot of one file specification dictionary
// found in the catalog's /Names /EmbeddedFiles name tree. Used by UA-12-002
// to check the /Desc requirement of ISO 14289-2 §8.14.1.
type EmbeddedFileSpec struct {
	// Filename is /UF if present (Unicode form), else /F. Empty when
	// neither is set.
	Filename string

	// HasDesc reports whether the filespec dictionary carries a /Desc
	// entry -- the ISO 14289-2 §8.14.1 requirement. Presence-only: an
	// empty /Desc () still counts as present.
	HasDesc bool

	// HasEF reports whether the filespec carries an /EF entry (an actual
	// embedded-file stream). ISO 14289-1 §7.11 only constrains file
	// specifications that embed a file.
	HasEF bool

	// HasNonEmptyF and HasNonEmptyUF report whether the filespec's /F and
	// /UF keys are present with a non-empty string value. UA-12-003 (ISO
	// 14289-1 §7.11) requires both to be non-empty on an embedded-file
	// filespec.
	HasNonEmptyF  bool
	HasNonEmptyUF bool
}

// MediaClip is a value snapshot of one media clip data dictionary
// (/Type /MediaClip, /S /MCD) reached from a Screen annotation's
// Rendition action. Used by the UA-17 media checks to verify the /CT
// (content type) and /Alt (alternate text) entries required by
// ISO 14289-1 §7.18.6.2.
type MediaClip struct {
	// Page is the 1-based page of the Screen annotation the clip was
	// reached from, for locating the finding.
	Page int

	// HasCT reports whether the media clip data dictionary carries a /CT
	// (content type) key. Presence-only: an empty /CT () still counts, as
	// ISO 14289-1 §7.18.6.2 only requires the key to be present.
	HasCT bool

	// HasCorrectAlt reports whether the /Alt array contains a default
	// entry -- a (culture, text) pair whose culture string is empty --
	// with a non-empty text string. The /Alt array alternates culture and
	// text strings; the empty-culture entry is the fallback description
	// that must carry meaningful text.
	HasCorrectAlt bool
}

// XFAInfo describes a document's XFA form, as needed by UA-28-028.
type XFAInfo struct {
	// Present is true when the /AcroForm dictionary carries an /XFA entry.
	Present bool

	// DynamicRender is the value of the <dynamicRender> element in the XFA
	// config packet: "required" for a dynamic form (forbidden by ISO
	// 14289-1 §7.15), "forbidden" for a static form, or "" when no XFA is
	// present or the element is absent.
	DynamicRender string
}

// TextString is a value snapshot of one human-readable text string in the
// document, with a human-facing location label for reporting. Used by
// UA-01-019 (ISO 14289-2 §8.6).
type TextString struct {
	// Location describes where the string was found, e.g. "Info /Title",
	// "form field /T", "structure element /Alt", "outline item /Title".
	Location string

	// Value is the decoded Unicode string value.
	Value string
}

// FormXObject is a value snapshot of a form XObject (/Type /XObject,
// /Subtype /Form). Used by UA-30-001 to flag reference XObjects.
type FormXObject struct {
	// ObjectNumber is the PDF object number of the XObject stream, for
	// locating the finding. 0 when it could not be determined.
	ObjectNumber int

	// IsReference reports whether the XObject carries a /Ref entry, making
	// it a reference XObject (ISO 32000-1 §8.10.4) -- the construct ISO
	// 14289-1 §7.20 forbids.
	IsReference bool

	// HasMarkedContent reports whether the XObject's content stream carries
	// a marked-content sequence with an /MCID property (BDC ... /MCID),
	// meaning its content participates in the structure tree. Used by
	// UA-30-002.
	HasMarkedContent bool

	// RefCount is the number of times this XObject is invoked by a Do
	// operator across all page and form-XObject content streams. A form
	// XObject that carries marked content must be invoked at most once, so
	// its MCIDs map to a single unambiguous "semantic parent" in the
	// structure tree (ISO 14289-1 §7.20 / ISO 32000-1 §14.7.2). Used by
	// UA-30-002.
	RefCount int
}

// Destination is a value snapshot of one intra-document destination
// carrier: an outline item, the catalog /OpenAction, a Link
// annotation, or a /GoTo action. Used by UA-27-003 to enforce ISO
// 14289-2 §8.8 -- a destination whose target lies within the document
// shall be a structure destination (its target a structure element),
// not a page destination.
type Destination struct {
	// Source is a human-readable description of where the destination
	// was found, e.g. `outline item "Chapter 1"`, `Link annotation on
	// page 3`, or `catalog /OpenAction`. Used to locate the finding.
	Source string

	// ViaGoToAction distinguishes the two veraPDF rules that share the
	// §8.8 requirement: false when the destination is carried directly
	// as a /Dest (or a destination-array /OpenAction) -- veraPDF
	// UA2:8.8-1 (PDDestination); true when it is reached through a
	// /GoTo action's /D -- veraPDF UA2:8.8-2 (PDGoToAction).
	ViaGoToAction bool

	// TargetsPage is true only when the destination's target has been
	// positively identified as a page: an integer page index or a first
	// destination-array element that resolves to a /Type /Page object. A
	// page target is never a structure destination, so this is the §8.8
	// violation signal. A target that is a structure element, or that
	// cannot be classified, leaves this false so the check does not
	// report a false positive. A /SD structure-destination entry on the
	// carrier also leaves it false: such a carrier already offers a
	// structure destination and conforms.
	TargetsPage bool
}

// AcroFormField is a value snapshot of one terminal Widget annotation
// reached through a recursive walk of /AcroForm/Fields. Used by
// UA-28-007 to cross-check that every form field surfaced in the
// interactive layer is also referenced from the structure tree.
type AcroFormField struct {
	// FullName is the field's fully-qualified name: the local /T of
	// every ancestor (and self) joined by ".". Empty when nothing
	// along the chain declared /T.
	FullName string

	// StructParent is the value of /StructParent on the widget. -1
	// when the widget has no /StructParent at all -- the UA-28-007
	// failure pattern.
	StructParent int
}

// OptionalContentGroup is a value snapshot of one OCG (PDF layer)
// dictionary. Currently only /Name is surfaced because that is what
// UA-20-001 needs; other entries (/Intent, /Usage) can be added when
// further OCG checks land.
type OptionalContentGroup struct {
	// Name is the value of /Name on the OCG dictionary. Empty when
	// the entry is missing or not a text string.
	Name string
}

// OptionalContentConfig is a value snapshot of one optional-content
// configuration dictionary (the /D default config or an entry of /Configs).
// Used by the UA-20 config checks (ISO 14289-1 §7.10).
type OptionalContentConfig struct {
	// Source identifies the config's origin: "D" for the default config,
	// or "Configs[i]" (0-based) for an entry of the /Configs array.
	Source string

	// HasName reports whether the config carries a non-empty /Name text
	// string (ISO 14289-1 §7.10 requires it).
	HasName bool

	// HasAS reports whether the config carries an /AS (usage application)
	// entry, which ISO 14289-1 §7.10 forbids.
	HasAS bool
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

	// Name is the value of /Name, a PDF Name giving the annotation's
	// appearance identifier -- for a rubber-stamp annotation the stamp
	// icon name (e.g. "Approved"). Empty when /Name is absent. UA-28-021
	// uses it as the alternative to /Contents for stamp annotations.
	Name string

	// HasFileSpec reports whether the annotation carries a /FS file
	// specification dictionary (a file attachment annotation references
	// the embedded file this way). UA-28-024 uses it with
	// FileSpecHasAFRelationship.
	HasFileSpec bool

	// FileSpecHasAFRelationship reports whether the /FS filespec includes
	// an /AFRelationship entry. ISO 14289-2 §8.9.2.4.10 requires it on the
	// filespec referenced by a file attachment annotation. Meaningful only
	// when HasFileSpec is true.
	FileSpecHasAFRelationship bool

	// Tooltip is the value of /TU resolved with /Parent inheritance
	// (form fields express the tooltip on the field dict, not the
	// individual widget). Empty when no /TU is present anywhere up
	// the chain.
	Tooltip string

	// FieldTooltip is the /TU read at the form-field level that owns this
	// widget: the widget dict itself when it is a merged field-widget (carries
	// /FT), otherwise the nearest /Parent that carries /FT (the terminal
	// field). Unlike Tooltip it does NOT count a /TU sitting on a kid widget
	// that is not itself a field, matching the field-scoped semantics of ISO
	// 14289-1 §7.18.1 (veraPDF UA1:7.18.1-3, UA-28-026). Empty for non-widget
	// annotations or when the owning field has no /TU.
	FieldTooltip string

	// Lang is the annotation's own natural-language tag, resolved from
	// /Lang on the annotation itself or inherited up the /Parent (form
	// field) chain. Empty when none is found. See also StructLang.
	Lang string

	// StructLang is the effective /Lang of the structure element that
	// owns this annotation, reached via /StructParent through the
	// StructTreeRoot /ParentTree and then resolved up the element's /P
	// ancestor chain. Empty when the annotation is not in the structure
	// tree or no ancestor declares /Lang. UA-11-005 treats an annotation's
	// language as determinable when Lang, StructLang or the catalog /Lang
	// is set.
	StructLang string

	// StructType is the standard structure type (role-map resolved) of the
	// structure element that owns this annotation, reached via /StructParent
	// through the StructTreeRoot /ParentTree. Empty when the annotation is
	// not in the structure tree. UA-28-013 uses it to verify that a Link
	// annotation is enclosed in a Link structure element (ISO 14289-1
	// §7.18.5).
	StructType string

	// StructAlt is the /Alt entry of the structure element that owns this
	// annotation (the immediate enclosing element reached via /StructParent
	// through the /ParentTree). Empty when the annotation is not in the
	// structure tree or the enclosing element has no non-empty /Alt.
	// UA-28-025 treats an annotation as described when it has /Contents or
	// its enclosing element carries /Alt (ISO 14289-1 §7.18.1).
	StructAlt string

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

	// Invisible is true when bit 1 of /F (value 1) is set: a
	// non-standard annotation type without a handler is not displayed.
	// ISO 14289-2 §8.9.2.2 requires such an annotation to be an artifact
	// if it is invisible; UA-28-020 uses this with StructType.
	Invisible bool

	// ToggleNoView is true when bit 9 of /F (value 256) is set: the
	// NoView state is inverted for certain events, so a NoView annotation
	// with ToggleNoView set can still become visible. ISO 14289-2
	// §8.9.2.2 exempts such annotations from the NoView-artifact rule.
	ToggleNoView bool

	// OffPage is true when the annotation's /Rect lies entirely
	// outside the page's visible box (CropBox if present, else
	// MediaBox). Such annotations cannot be perceived by sighted
	// readers and must therefore also be hidden from assistive
	// technology via the Hidden or NoView flags -- a printed-but-
	// off-page annotation that AT still reads creates a phantom
	// reading-order entry.
	OffPage bool

	// ZeroSize is true when the annotation carries a /Rect whose width
	// and height are both zero. Distinct from a missing/unparsable /Rect,
	// which leaves ZeroSize false ("we don't know"). ISO 14289-2
	// §8.9.2.4.13 requires a zero-size Widget to be an artifact; UA-28-018
	// uses this together with StructType.
	ZeroSize bool

	// HasAA is true when the annotation carries an /AA additional-actions
	// dictionary. ISO 14289-2 §8.10.2.3 requires a widget with an /AA to
	// provide a /Contents description; UA-28-029 uses this.
	HasAA bool

	// IsFieldWidget is true when the annotation is a Widget that belongs
	// to the interactive form: it carries /FT itself, or a /Parent up its
	// chain does (the terminal form field). Widgets that are pure
	// appearance children with no field ancestry are not field widgets.
	IsFieldWidget bool

	// IsTextField is true when the owning form field is a text field
	// (/FT resolves to /Tx up the /Parent chain). ISO 14289-2 §8.10.3.3
	// (UA-28-033) constrains the /RV and /V of text fields specifically.
	IsTextField bool

	// FieldValue is the form field's /V value (the field's plain-text
	// value), resolved at the /FT-bearing field level. Empty when absent.
	FieldValue string

	// RichTextValue is the form field's /RV rich-text value (an XHTML
	// fragment per ISO 32000-1 §12.7.3.4), resolved at the field level. A
	// /RV given as a stream is decoded. Empty when absent. ISO 14289-2
	// §8.10.3.3 requires that when /RV is present, /V is present and the two
	// are textually equivalent; UA-28-033 uses this with FieldValue.
	RichTextValue string

	// RichContents is the annotation's /RC rich-text contents (an XHTML
	// fragment per ISO 32000-1 §12.5.6.2, on markup annotations), decoded
	// when given as a stream. Empty when absent. ISO 14289-2 §8.9.2.3
	// requires that when both /RC and /Contents are present they are
	// textually equivalent; UA-28-034 uses this with Contents.
	RichContents string

	// StructHasLbl reports whether the structure element that owns this
	// annotation (reached via /StructParent) has a Lbl structure element
	// among its direct children -- the label of a widget. ISO 14289-2
	// §8.10.2.3 (UA-28-035) requires a field widget with no such label and
	// no /Contents to be flagged. False when the annotation is not in the
	// structure tree or its owning element has no Lbl child.
	StructHasLbl bool

	// StructOwnLang is the /Lang declared directly on the structure element
	// that owns this annotation (via /StructParent), WITHOUT inheriting from
	// /P ancestors -- distinct from StructLang, which does inherit. ISO
	// 14289-1 §7.2 (UA-11-009) treats a form field's /TU language as
	// determinable only when the field's own structure element carries /Lang.
	// Empty when the annotation is not in the structure tree or its owning
	// element has no own /Lang.
	StructOwnLang string
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

	// MCIDBoxes maps each MCID seen on this page to a bounding box
	// in PDF user-space units (origin at the bottom-left of the page,
	// Y growing up). The box is the axis-aligned hull of the text
	// matrix start positions of every Tj/TJ/'/" emitted while that
	// MCID was on the marked-content stack.
	//
	// Glyph widths and CTM scale/rotation are ignored: the box is a
	// pragmatic approximation suitable for reading-order heuristics,
	// not a pixel-perfect rendering of the content. Pages with no
	// text-showing operators have an empty map.
	MCIDBoxes map[int]Rect

	// MCIDText maps each MCID seen on this page to the Unicode text
	// shown while that MCID was the innermost (tightest-enclosing)
	// marked-content tag. Decoded via the active font's /ToUnicode
	// CMap; glyph codes with no ToUnicode entry are dropped, so fonts
	// without /ToUnicode contribute empty or partial text. Unlike
	// MCIDBoxes, text is attributed to the innermost MCID only (not to
	// every ancestor on the stack), so a Span nested inside a P does
	// not duplicate its text into the P's own entry.
	MCIDText map[int]string

	// MCNestingViolations lists marked-content sequences on this page that
	// nest an Artifact and structure-tagged content inside one another,
	// which PDF/UA-1 §7.1 forbids in either direction (an Artifact inside
	// tagged content, or tagged content inside an Artifact). Empty on a
	// well-formed page. Used by UA-14-010.
	MCNestingViolations []MCNestingViolation

	// SpanLangViolations lists Span marked-content sequences on this page
	// that carry an /ActualText, /Alt or /E property whose natural language
	// cannot be determined (no /Lang on the Span, no inherited Lang from an
	// enclosing sequence or structure element, and no catalog /Lang). ISO
	// 14289-1 §7.2 (veraPDF UA1:7.2-30/-31/-32). Empty on a well-formed
	// page. Used by UA-11-008.
	SpanLangViolations []SpanLangViolation

	// PUAContentMCIDs is the set of MCIDs on this page whose decoded real
	// content (via /ToUnicode) contains a Unicode Private Use Area code
	// point and that no /ActualText or /Alt covers -- neither on the
	// marked-content sequence nor on the owning structure element. These are
	// the UA-01-020 violations (ISO 14289-2 §8.4.3). Empty when every PUA
	// sequence is covered or none is present.
	PUAContentMCIDs map[int]bool

	// HadPUARealContent reports whether any real content on this page decoded
	// (via /ToUnicode) to a Unicode Private Use Area code point at all,
	// before applying /ActualText or /Alt exemptions. Lets UA-01-020
	// distinguish "PUA present but properly covered" (pass) from "no PUA
	// content at all" (not applicable).
	HadPUARealContent bool
}

// SpanLangViolation records one Span marked-content attribute whose natural
// language is undeterminable (ISO 14289-1 §7.2). Used by UA-11-008.
type SpanLangViolation struct {
	// Attribute is the property whose language cannot be determined:
	// "ActualText", "Alt" or "E".
	Attribute string

	// Offset is the byte offset of the offending BDC operator in the
	// decoded content stream, for locating the finding.
	Offset int64
}

// MCNestingViolation records one illegal nesting of Artifact and
// structure-tagged marked content in a page content stream (ISO 14289-1
// §7.1). Used by UA-14-010.
type MCNestingViolation struct {
	// ArtifactInsideTagged is true when an Artifact marked-content sequence
	// was opened while a structure-tagged sequence was already on the stack
	// (veraPDF UA1:7.1-1). False for the opposite direction: a tagged
	// sequence opened inside an Artifact (veraPDF UA1:7.1-2).
	ArtifactInsideTagged bool

	// InnerTag is the tag name of the inner (offending) marked-content
	// sequence -- "Artifact" for the 7.1-1 direction, or the structure tag
	// (e.g. "Note", "Span") for the 7.1-2 direction.
	InnerTag string

	// Offset is the byte offset of the offending BDC/BMC operator in the
	// decoded content stream, for locating the finding.
	Offset int64
}

// Rect is an axis-aligned rectangle in PDF user-space units. The PDF
// coordinate system places the origin at the bottom-left of the page
// with Y growing upwards, so visual top-to-bottom reading order
// corresponds to MaxY descending.
type Rect struct {
	MinX, MinY, MaxX, MaxY float64
}

// Empty reports whether r has never been extended — i.e. it is the
// zero value. A degenerate one-point rect (MinX == MaxX, MinY == MaxY)
// is intentionally NOT empty: the walker records glyph start positions
// without computing widths, so single-point rects are the normal
// representation of "one Tj at this position".
func (r Rect) Empty() bool {
	return r == Rect{}
}

// Extend grows r to cover (x, y). The first point on a zero-value Rect
// initialises both min and max corners to that point.
func (r Rect) Extend(x, y float64) Rect {
	if r == (Rect{}) {
		return Rect{MinX: x, MinY: y, MaxX: x, MaxY: y}
	}
	if x < r.MinX {
		r.MinX = x
	}
	if x > r.MaxX {
		r.MaxX = x
	}
	if y < r.MinY {
		r.MinY = y
	}
	if y > r.MaxY {
		r.MaxY = y
	}
	return r
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

// Namespace is a PDF 2.0 structure namespace declared on the StructTreeRoot
// /Namespaces array (ISO 32000-2 §14.8.6).
type Namespace struct {
	// URI is the namespace's /NS string (e.g. "http://iso.org/pdf2/ssn").
	URI string

	// RoleMapNS maps a structure type to its role-map targets within this
	// namespace's /RoleMapNS. An entry that gives only a type name (no
	// namespace reference) targets the same namespace, so its RoleTarget
	// carries this namespace's URI.
	RoleMapNS map[string][]RoleTarget
}

// RoleTarget is one target of a namespace role mapping: the type mapped to,
// and the URI of the namespace that type belongs to.
type RoleTarget struct {
	Type         string
	NamespaceURI string
}

// Dict represents a PDF dictionary for read-only access by checks.
type Dict interface {
	// Find returns the value for key and whether the key was present.
	Find(key string) (Object, bool)

	// String returns the decoded text-string value for key (PDF string
	// literal or name), following indirect references, and whether a
	// string value was found. Non-string entries and absent keys yield
	// ("", false). Used e.g. to read the Catalog /Lang tag.
	String(key string) (string, bool)
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
	Encoding               string
	HasEncodingDifferences bool
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
	// /ToUnicode CMap's begincodespacerange (1 or 2). It records the
	// extraction CMap's own codespace and is independent of how the
	// content stream is tokenised (see EncodingCodeBytes). Zero when
	// no codespace was declared.
	ToUnicodeCodeBytes int

	// EncodingCodeBytes is the number of bytes per code in the
	// content stream, as fixed by the font's /Encoding: 1 for simple
	// fonts, and for Type0 the /Encoding CMap's codespace width
	// (2 for Identity-H/V and the predefined Adobe CMaps; an embedded
	// CMap declares its own). This -- not ToUnicodeCodeBytes -- is
	// what the walker uses to split Tj/TJ bytes into codes. The two
	// are independent: a font may render two-byte codes yet ship a
	// /ToUnicode whose codespace is declared one-byte.
	EncodingCodeBytes int

	// CIDSubtype is the /Subtype of the descendant CIDFont for
	// Type0 composite fonts: "CIDFontType0" (Adobe CFF source) or
	// "CIDFontType2" (TrueType source). Empty when the font is not
	// Type0 or has no resolvable descendant.
	CIDSubtype string

	// CIDToGIDMap reflects the descendant CIDFont's /CIDToGIDMap
	// entry. Values:
	//   - "Identity" — name set to /Identity (or the entry omitted,
	//     for which ISO 32000-1 §9.7.4.2 defines Identity as the
	//     default)
	//   - "Stream"   — a stream is present, mapping CIDs to GIDs
	//   - other      — a non-Identity Name was set (the UA-31-001
	//     failure pattern)
	// Empty when the font is not a CIDFontType2 descendant.
	CIDToGIDMap string
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

	// AttrPresent reports whether the top-level attribute exists on
	// the element dictionary as a string value (after dereferencing).
	// Distinct from Attr() != "" because PDF/UA-1 §7.5 treats an
	// /ActualText present-but-empty entry on Formula as a valid
	// "render as silence" representation, while a missing entry is
	// not. Callers that only care about a non-empty value should
	// continue to use Attr().
	AttrPresent(name string) bool

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

	// AttributeInt returns an integer-valued structure attribute from the
	// /A attribute-owner dictionary (or /C ClassMap), following the same
	// resolution as Attribute. Used for /RowSpan and /ColSpan on table
	// cells. Returns (0, false) when the attribute is absent or not an
	// integer; callers should treat "not present" as the PDF default
	// span of 1.
	AttributeInt(name string) (int, bool)

	// HasParent reports whether the element dictionary carries a /P
	// (parent) entry. ISO 14289-2 §8.2.1 (via ISO 32000-2 Table 323)
	// requires /P on every structure element; UA-01-014 uses this to flag
	// elements that omit it. Presence-only: the target is not resolved, so
	// a /P pointing at a missing object still counts as present here.
	HasParent() bool

	// ID returns the element's /ID entry (a byte string identifier used as
	// the target of /Headers references and table-header association) as a
	// string, or "" when absent.
	ID() string

	// HeaderIDs returns the element's /Headers structure attribute -- the
	// list of /ID values of the header cells that pertain to this cell
	// (ISO 32000-2 §14.8.5.7) -- read from the /A attribute-owner dict.
	// Returns nil when /Headers is absent (distinct from an empty array,
	// which yields a non-nil empty slice).
	HeaderIDs() []string

	// Page returns the 1-based page number this structure element
	// belongs to, resolved from its /Pg entry. Returns 0 when the
	// element carries no /Pg (some authors only set it on leaves,
	// others omit it entirely) or when the referenced page is not
	// found in the page tree.
	Page() int

	// Refs returns the StructElements referenced by /Ref (an array
	// of indirect references on the element dict). Entries that do
	// not resolve to a structure element are dropped: this signature
	// cannot distinguish "no /Ref" from "broken /Ref". UA-19-001 uses
	// it to follow Note/Reference cross-links.
	Refs() []StructElement

	// Same reports whether other denotes the same structure element (the
	// same underlying PDF object). Used to match /Ref cross-links, e.g.
	// checking that a FENote references back the citation that references
	// it.
	Same(other StructElement) bool

	// AssociatedFiles returns the filespecs reached through this
	// element's own /AF array (typically a per-Formula AF carrying
	// MathML or LaTeX source per ISO 14289-2 §8.2.5.29.1). Returns
	// nil when the element has no /AF. Entries that fail to resolve
	// to a dictionary are silently dropped.
	AssociatedFiles() []AssociatedFile

	// Namespace returns the URI string of the namespace this element
	// belongs to, as declared by its /NS entry (or inherited through
	// the /P chain). Returns "" when no /NS attribute can be resolved
	// on the element or any ancestor -- in tagged PDF this means the
	// element falls into the default/PDF namespace. UA-17-005 uses
	// this to verify that a 'math' struct child of Formula is
	// actually declared in the W3C MathML namespace, not just named
	// "math" by coincidence (the ISO 32000-2 §14.8.6.3 contract).
	Namespace() string

	// Text returns the Unicode text drawn directly under this element's
	// own marked content (its integer / MCR K-children), concatenated in
	// K order. Text belonging to child structure elements is excluded --
	// those carry their own Text(). The source is the page content stream
	// decoded via /ToUnicode, so it reflects the glyphs actually shown,
	// independent of any /ActualText or /Alt override. Returns "" for a
	// grouping element with no direct marked-content text, or when the
	// text cannot be decoded.
	Text() string

	// Content returns the element's content in /K order: an interleaved
	// sequence of text runs (drawn under the element's own marked content)
	// and child structure elements. This preserves the reading order that
	// Children() and Text() each lose on their own -- e.g. a paragraph
	// whose prose wraps several inline Formula children. Text runs that
	// decode to empty are omitted; OBJR children are skipped.
	Content() []ContentItem

	// EnclosedWidgetCount returns the number of widget annotations this
	// element directly encloses: the /K children that are OBJR
	// references whose /Obj resolves to a /Subtype /Widget annotation.
	// Children() and Content() drop OBJR entries, so this is the only
	// accessor that sees them. Used by UA-28-027 to enforce ISO 14289-2
	// §8.10.1 (a Form structure element shall enclose at most one widget
	// annotation).
	EnclosedWidgetCount() int

	// EnclosesSignatureWidget reports whether this element directly encloses
	// a signature-field widget: an OBJR /K child whose annotation is a
	// /Subtype /Widget belonging to a form field whose /FT resolves to /Sig
	// (own or inherited up the /Parent chain). Used by UA-28-036 to find the
	// Form element of a signature so its graphic appearance can be checked
	// for /Alt (ISO 14289-2 §8.10.3.5).
	EnclosesSignatureWidget() bool

	// EnclosedLinkTargets returns the destination "target key" of each
	// Link annotation this element directly encloses via an OBJR /K
	// child. The key canonically identifies where the link points -- its
	// structure destination (/SD, preferred), explicit destination, or
	// URI -- so a caller can tell whether two enclosed links target the
	// same location. Links with no determinable target are omitted. Used
	// by UA-28-030 to enforce ISO 14289-2 §8.2.5.20 (link annotations
	// targeting different locations shall be in separate Link/Reference
	// elements).
	EnclosedLinkTargets() []string
}

// ContentItem is one entry in a structure element's content sequence (see
// StructElement.Content). Exactly one field is set: Text for a run of
// decoded marked-content text, or Element for a child structure element.
type ContentItem struct {
	Text    string
	Element StructElement
}
