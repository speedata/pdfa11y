// genfixtures is a build-tool sub-module: pdfa11y itself does not
// depend on pdfcpu (the runtime parser is pdfdisassembler), but
// regenerating the test fixtures requires PDF *writing* which
// pdfdisassembler does not provide. Keeping this in a separate Go
// module ensures pdfa11y's top-level go.mod stays parser-clean.
//
// Run from this directory:
//   go run .
module github.com/speedata/pdfa11y/cmd/genfixtures

go 1.25.0

require github.com/pdfcpu/pdfcpu v0.12.1

require (
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/pkcs7 v0.2.2 // indirect
	github.com/hhrutter/tiff v1.0.3 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/image v0.39.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
