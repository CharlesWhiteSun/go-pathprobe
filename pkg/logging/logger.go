package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds a logger with a mutable level guard suitable for CLI toggling.
func NewLogger(level slog.Level) (*slog.Logger, *slog.LevelVar) {
	var levelVar slog.LevelVar
	levelVar.Set(level)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: &levelVar,
	})
	return slog.New(handler), &levelVar
}

// ParseLevel converts string inputs to slog.Level values.
func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", value)
	}
}
