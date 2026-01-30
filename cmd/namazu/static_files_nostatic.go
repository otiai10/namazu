//go:build nostatic

package main

import "io/fs"

// getStaticFS returns nil when built without static files.
func getStaticFS() (fs.FS, bool) {
	return nil, false
}

// staticRoot returns empty string when built without static files
func staticRoot() string {
	return ""
}
