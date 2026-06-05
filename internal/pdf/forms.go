package pdf

import (
	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// AcroFormFields walks /AcroForm/Fields recursively and returns one
// AcroFormField per terminal Widget annotation reached. Fields that
// expose no widget kid (purely abstract field dicts) are skipped:
// without a widget there is nothing on a page to link from the
// structure tree, so the cross-check does not apply.
func (d *document) AcroFormFields() ([]model.AcroFormField, error) {
	cat, err := d.r.Catalog()
	if err != nil {
		return nil, err
	}
	af, ok := cat.Dict("AcroForm")
	if !ok {
		return nil, nil
	}
	fieldsObj, ok := af.Get("Fields")
	if !ok {
		return nil, nil
	}
	arr, err := d.r.ResolveArray(fieldsObj)
	if err != nil {
		return nil, err
	}
	var out []model.AcroFormField
	for _, item := range arr {
		d.walkAcroField(item, "", &out)
	}
	return out, nil
}

// walkAcroField recurses through one /AcroForm field-tree node.
// Subtype=Widget marks a terminal node and is emitted; /Kids drives
// the recursion. Self+Kids both being processed (verschmolzene
// widget+field with radio-group kids) is intentional -- such nodes
// are reportable themselves and as the root of their kids.
func (d *document) walkAcroField(obj pdd.Object, prefix string, out *[]model.AcroFormField) {
	fd, err := d.r.ResolveDict(obj)
	if err != nil || fd == nil {
		return
	}
	name := prefix
	if t, ok := fd.String("T"); ok && t != "" {
		if name != "" {
			name = name + "." + t
		} else {
			name = t
		}
	}
	if subtype, ok := fd.Name("Subtype"); ok && subtype == "Widget" {
		sp := -1
		if v, ok := fd.Int("StructParent"); ok {
			sp = int(v)
		}
		*out = append(*out, model.AcroFormField{
			FullName:     name,
			StructParent: sp,
		})
	}
	kidsObj, ok := fd.Get("Kids")
	if !ok {
		return
	}
	kids, err := d.r.ResolveArray(kidsObj)
	if err != nil {
		return
	}
	for _, kid := range kids {
		d.walkAcroField(kid, name, out)
	}
}
