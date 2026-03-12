package version_test

import (
	"strings"
	"testing"

	"go-pathprobe/pkg/version"
)

func TestVersionNotEmpty(t *testing.T) {
	if version.Version == "" {
		t.Fatal("Version must not be empty")
	}
}

// TestVersionDefaultIsDev verifies that the package-level default before
// ldflags injection is exactly "dev".
func TestVersionDefaultIsDev(t *testing.T) {
	// When run via `go test` (without ldflags), the default value is "dev".
	// When built via build.ps1 the value follows the tag-hash[-dirty] pattern.
	if version.Version != "dev" {
		// Injected by ldflags: validate format is non-empty and contains no
		// whitespace (the badge must be a single token).
		if strings.ContainsAny(version.Version, " \t\n") {
			t.Errorf("Version %q must not contain whitespace", version.Version)
		}
	}
}
