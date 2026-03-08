package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web
var webFS embed.FS

// newStaticHandler returns an http.Handler that serves the embedded web UI
// from the pkg/server/web directory.  Content-Type, ETag, and Last-Modified
// headers are handled automatically by the standard library.
// Panic on startup (not at request time) if the embedded FS is malformed — this
// is caught at build/test time, never in production.
func newStaticHandler() http.Handler {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic("embedded web FS is missing 'web' subdirectory: " + err.Error())
	}
	return http.FileServerFS(sub)
}
