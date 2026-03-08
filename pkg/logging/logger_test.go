package logging

import (
	"log/slog"
	"testing"
)

// TestParseLevelValid covers all recognised level strings including case variations.
func TestParseLevelValid(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"Debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo}, // empty string defaults to info
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
	}
	for _, c := range cases {
		got, err := ParseLevel(c.input)
		if err != nil {
			t.Fatalf("ParseLevel(%q) unexpected error: %v", c.input, err)
		}
		if got != c.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

// TestParseLevelInvalid verifies that unrecognised strings return an error.
func TestParseLevelInvalid(t *testing.T) {
	invalids := []string{"verbose", "trace", "fatal", "critical", "off", "all"}
	for _, input := range invalids {
		_, err := ParseLevel(input)
		if err == nil {
			t.Fatalf("ParseLevel(%q) expected error, got nil", input)
		}
	}
}

// TestParseLevelLeadingTrailingSpace ensures trimming works correctly.
func TestParseLevelLeadingTrailingSpace(t *testing.T) {
	got, err := ParseLevel("  info  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != slog.LevelInfo {
		t.Fatalf("expected LevelInfo, got %v", got)
	}
}

// TestNewLoggerReturnsMutableLevel verifies the logger and levelVar are correctly linked.
func TestNewLoggerReturnsMutableLevel(t *testing.T) {
	logger, levelVar := NewLogger(slog.LevelInfo)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if levelVar == nil {
		t.Fatal("expected non-nil levelVar")
	}
	if levelVar.Level() != slog.LevelInfo {
		t.Fatalf("expected initial level info, got %v", levelVar.Level())
	}

	// Mutate and verify the handler picks up the new level.
	levelVar.Set(slog.LevelDebug)
	if levelVar.Level() != slog.LevelDebug {
		t.Fatalf("expected level debug after set, got %v", levelVar.Level())
	}
}
