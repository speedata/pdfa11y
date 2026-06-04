// The web/ subdirectory holds the WASM frontend assets and a tiny dev
// HTTP server. It is a separate Go module so the top-level go.mod is
// not polluted by HTTP-server-only dependencies (currently none, but
// kept in case the dev server grows).
//
// Build the WASM binary from the repository root:
//
//	GOOS=js GOARCH=wasm go build -o web/pdfa11y.wasm ./cmd/pdfa11ywasm
//
// Start the dev server:
//
//	cd web && go run serve.go
//
// Then open http://localhost:8080
module github.com/speedata/pdfa11y/web

go 1.23
