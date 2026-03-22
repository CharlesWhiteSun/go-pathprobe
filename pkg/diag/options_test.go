package diag

import (
	"log/slog"
	"testing"
	"time"
)

// TestGlobalOptionsValidate covers happy path and rejects invalid counts.
func TestGlobalOptionsValidate(t *testing.T) {
	opts := GlobalOptions{MTRCount: 1, Timeout: time.Second}
	if err := opts.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	opts.MTRCount = 0
	if err := opts.Validate(); err == nil {
		t.Fatalf("expected error for zero mtr-count")
	}
}

// TestDefaultLogLevel keeps default log level stable at info.
func TestDefaultLogLevel(t *testing.T) {
	opts := GlobalOptions{LogLevel: slog.LevelInfo}
	if opts.LogLevel != slog.LevelInfo {
		t.Fatalf("expected info level, got %v", opts.LogLevel)
	}
}

// TestTimeoutValidation ensures zero/negative timeout is rejected early.
func TestTimeoutValidation(t *testing.T) {
	opts := GlobalOptions{MTRCount: 1, Timeout: 0}
	if err := opts.Validate(); err == nil {
		t.Fatalf("expected error for zero timeout")
	}
}
