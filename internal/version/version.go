// Package version provides build-time version information.
package version

// CommitHash is the git commit hash of the build.
// It is set at build time via -ldflags:
//
//	go build -ldflags "-X github.com/otiai10/namazu/internal/version.CommitHash=abc1234"
var CommitHash = "unknown"
