//go:build !nostatic

package main

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFS embed.FS

// getStaticFS returns the embedded static filesystem.
// Returns the filesystem and true if static files are available.
func getStaticFS() (fs.FS, bool) {
	// Check if static directory has content
	entries, err := fs.ReadDir(staticFS, "static")
	if err != nil || len(entries) == 0 {
		return nil, false
	}
	return staticFS, true
}

// staticRoot returns the subdirectory name in the embedded FS
func staticRoot() string {
	return "static"
}
