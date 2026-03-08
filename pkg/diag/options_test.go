package diag

import (
	"log/slog"
	"testing"
	"time"
)

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

func TestDefaultLogLevel(t *testing.T) {
	opts := GlobalOptions{MTRCount: 1, Timeout: time.Second, LogLevel: slog.LevelInfo}
	if opts.LogLevel != slog.LevelInfo {
		t.Fatalf("expected info level, got %v", opts.LogLevel)
	}
}

func TestTimeoutValidation(t *testing.T) {
	opts := GlobalOptions{MTRCount: 1, Timeout: 0}
	if err := opts.Validate(); err == nil {
		t.Fatalf("expected error for zero timeout")
	}
}
