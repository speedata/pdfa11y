// serve.go is a minimal dev HTTP server for the WASM frontend.
// Run from the web/ directory:
//
//	go run serve.go
//
// It sets the application/wasm MIME type explicitly (file extensions
// alone are not always honoured by the stdlib MIME table) and serves
// the current directory on :8080.
package main

import (
	"log"
	"mime"
	"net/http"
)

func main() {
	mime.AddExtensionType(".wasm", "application/wasm")
	addr := ":8080"
	log.Printf("pdfa11y dev server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, http.FileServer(http.Dir("."))); err != nil {
		log.Fatal(err)
	}
}
