// Package version holds the build-time version information injected via ldflags.
// Build with: go build -ldflags "-X go-pathprobe/pkg/version.Version=vX.Y.Z -X go-pathprobe/pkg/version.BuildTime=..."
package version

// Version is injected at build time.  Falls back to "dev" for local builds.
var Version = "dev"

// BuildTime is optionally injected at build time (ISO-8601 UTC string).
var BuildTime = "unknown"
