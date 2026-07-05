package pdf

import (
	"regexp"

	pdd "github.com/speedata/pdfdisassembler"

	"github.com/speedata/pdfa11y/internal/model"
)

// dynamicRenderRE extracts the value of the XFA config packet's
// <dynamicRender> element, e.g. <dynamicRender>required</dynamicRender>.
// The element name may be followed by whitespace before '>', as some XFA
// writers emit (`<dynamicRender >required</dynamicRender >`).
var dynamicRenderRE = regexp.MustCompile(`<dynamicRender\s*>\s*([A-Za-z]+)`)

// XFA resolves /AcroForm/XFA and, when present, extracts the
// <dynamicRender> value from its config packet.
func (d *document) XFA() model.XFAInfo {
	cat, err := d.r.Catalog()
	if err != nil || cat == nil {
		return model.XFAInfo{}
	}
	af, ok := cat.Dict("AcroForm")
	if !ok {
		return model.XFAInfo{}
	}
	xfaObj, ok := af.Get("XFA")
	if !ok {
		return model.XFAInfo{}
	}
	return model.XFAInfo{
		Present:       true,
		DynamicRender: d.xfaDynamicRender(xfaObj),
	}
}

// xfaDynamicRender decodes the XFA packet(s) and returns the
// <dynamicRender> value. /XFA is either a single stream (the whole XDP) or
// an array of alternating (packet-name, stream) entries; when the array
// names a "config" packet, only that one is inspected, otherwise every
// packet is scanned. Returns "" when the element is not found.
func (d *document) xfaDynamicRender(xfaObj pdd.Object) string {
	resolved, err := d.r.Resolve(xfaObj)
	if err != nil {
		return ""
	}
	var streams []pdd.Object
	switch v := resolved.(type) {
	case pdd.Array:
		var config []pdd.Object
		for i := 0; i+1 < len(v); i += 2 {
			if name, ok := v[i].(pdd.String); ok && string(name) == "config" {
				config = append(config, v[i+1])
			}
			streams = append(streams, v[i+1])
		}
		if len(config) > 0 {
			streams = config
		}
	default:
		streams = []pdd.Object{xfaObj}
	}
	for _, s := range streams {
		data, err := d.DecodeStream(s)
		if err != nil {
			continue
		}
		if m := dynamicRenderRE.FindSubmatch(data); m != nil {
			return string(m[1])
		}
	}
	return ""
}
