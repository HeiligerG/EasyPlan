package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFS embed.FS

// staticPrefix is the path prefix of the static files.
const staticPrefix = "/static/"

// cacheControl is the cache header of the static files. They change only
// with a new version of the binary, an hour of cache is enough to keep the
// pages light without hiding an update for long.
const cacheControl = "public, max-age=3600"

// staticHandler serves the embedded static files.
//
// The files are read from the binary, so the application needs no build step
// and no files next to the executable. The handler serves the files with an
// entity tag, so a browser that already has the file does not download it
// again.
type staticHandler struct {
	files    http.Handler
	etags    map[string]string
	notFound http.Handler
}

// newStaticHandler returns the handler for the embedded static files.
func (s *server) newStaticHandler() (http.Handler, error) {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("web: cannot read the embedded static files: %w", err)
	}

	etags := make(map[string]string)
	entries, err := fs.ReadDir(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("web: cannot read the embedded static files: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := fs.ReadFile(staticFS, "static/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("web: cannot read the embedded static file %q: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(content)
		etags[entry.Name()] = `"` + hex.EncodeToString(sum[:]) + `"`
	}

	return &staticHandler{
		files:    http.StripPrefix(staticPrefix, http.FileServerFS(sub)),
		etags:    etags,
		notFound: http.HandlerFunc(s.notFoundPage),
	}, nil
}

// ServeHTTP serves one of the embedded static files.
func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, staticPrefix)
	etag, ok := h.etags[name]
	if !ok || strings.HasSuffix(name, "/") {
		// The directory itself is not a file and there is nothing to list.
		h.notFound.ServeHTTP(w, r)
		return
	}

	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", cacheControl)

	h.files.ServeHTTP(w, r)
}
