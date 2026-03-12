// Package version holds the build-time version information injected via ldflags.
// Build with:
//
//	go build -ldflags "-X go-pathprobe/pkg/version.Version=v1.2.3-abc123"
//
// The Version string follows the format: <tag>-<6-char-git-hash>[-dirty]
// where "-dirty" indicates uncommitted changes or unpushed commits.
package version

// Version is injected at build time.  Falls back to "dev" for local builds.
var Version = "dev"
