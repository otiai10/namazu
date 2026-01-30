package api

import (
	"io/fs"
	"net/http"
	"strings"
)

// StaticFileServer serves static files from an embedded filesystem with SPA fallback.
// It returns index.html for any non-file request to support client-side routing.
type StaticFileServer struct {
	subFS      fs.FS // Sub-filesystem starting at fileRoot
	fileServer http.Handler
}

// NewStaticFileServer creates a new static file server.
// The staticFS parameter should be an embed.FS or similar filesystem.
// The fileRoot is the subdirectory within fs (e.g., "static" for //go:embed static).
// If fileRoot is empty, staticFS is used directly.
func NewStaticFileServer(staticFS fs.FS, fileRoot string) *StaticFileServer {
	var subFS fs.FS
	if fileRoot != "" {
		var err error
		subFS, err = fs.Sub(staticFS, fileRoot)
		if err != nil {
			// Fallback to original fs if Sub fails
			subFS = staticFS
		}
	} else {
		subFS = staticFS
	}

	return &StaticFileServer{
		subFS:      subFS,
		fileServer: http.FileServer(http.FS(subFS)),
	}
}

// ServeHTTP implements http.Handler.
// It serves static files and falls back to index.html for SPA routing.
func (s *StaticFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Check if file exists
	if _, err := fs.Stat(s.subFS, path); err == nil {
		// File exists, serve it
		s.fileServer.ServeHTTP(w, r)
		return
	}

	// File doesn't exist, serve index.html for SPA routing
	content, err := fs.ReadFile(s.subFS, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// WithStaticFiles wraps an API router with static file serving.
// API routes (starting with /api or /health) are handled by the apiHandler,
// all other routes fall through to the static file server.
func WithStaticFiles(apiHandler http.Handler, staticServer *StaticFileServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// API routes
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/health") {
			apiHandler.ServeHTTP(w, r)
			return
		}

		// Static files
		staticServer.ServeHTTP(w, r)
	})
}
