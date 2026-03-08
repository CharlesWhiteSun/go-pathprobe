package version_test

import (
	"testing"

	"go-pathprobe/pkg/version"
)

func TestVersionNotEmpty(t *testing.T) {
	if version.Version == "" {
		t.Fatal("Version must not be empty")
	}
}

func TestBuildTimeNotEmpty(t *testing.T) {
	if version.BuildTime == "" {
		t.Fatal("BuildTime must not be empty")
	}
}
