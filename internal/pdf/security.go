// Security and optional-content document features.
package pdf

import "github.com/speedata/pdfa11y/internal/model"

// OptionalContentGroups walks the catalog's /OCProperties/OCGs array
// and returns one OptionalContentGroup snapshot per entry. Returns an
// empty slice when /OCProperties or /OCGs is absent (the common case
// on documents without layers).
func (d *document) OptionalContentGroups() ([]model.OptionalContentGroup, error) {
	cat, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	props, ok := cat.Dict("OCProperties")
	if !ok {
		return nil, nil
	}
	ocgsObj, ok := props.Get("OCGs")
	if !ok {
		return nil, nil
	}
	arr, err := d.r.ResolveArray(ocgsObj)
	if err != nil {
		return nil, err
	}
	out := make([]model.OptionalContentGroup, 0, len(arr))
	for _, item := range arr {
		ocg, err := d.r.ResolveDict(item)
		if err != nil || ocg == nil {
			// Skip unresolvable entries; the file is malformed but
			// we still want to evaluate the remaining OCGs.
			out = append(out, model.OptionalContentGroup{})
			continue
		}
		name, _ := ocg.String("Name")
		out = append(out, model.OptionalContentGroup{Name: name})
	}
	return out, nil
}

// Encryption returns the document's encryption-permission state.
// pdfdisassembler reads the trailer /Encrypt during Open() and
// exposes the trailer through Trailer(); from there /P (a signed
// integer of permission bits) drives the per-bit accessors below.
func (d *document) Encryption() model.EncryptionInfo {
	tr := d.r.Trailer()
	if tr == nil {
		return model.EncryptionInfo{}
	}
	encObj, ok := tr.Get("Encrypt")
	if !ok {
		return model.EncryptionInfo{}
	}
	enc, err := d.r.ResolveDict(encObj)
	if err != nil || enc == nil {
		// Trailer claims encryption but the dict cannot be read;
		// be conservative and report Encrypted=true with no
		// permissions, so the check fires.
		return model.EncryptionInfo{Encrypted: true}
	}
	p, _ := enc.Int("P")
	return model.EncryptionInfo{
		Encrypted:          true,
		AllowExtractText:   permissionBit(p, 5),
		AllowAccessibility: permissionBit(p, 10),
	}
}

// permissionBit reads a single PDF /P permission bit. PDF numbers
// bits 1-based starting from the low end, so bit N corresponds to
// mask (1 << (N-1)). PDF stores /P as a 32-bit signed integer with
// the high reserved bits set to 1; bitwise AND against the mask
// still works correctly on the signed value.
func permissionBit(p int64, bit int) bool {
	mask := int64(1) << (bit - 1)
	return p&mask != 0
}

