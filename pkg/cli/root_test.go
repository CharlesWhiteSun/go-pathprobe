package cli

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
)

type recordingRunner struct {
	calls       int
	lastRequest diag.Request
}

func (r *recordingRunner) Run(_ context.Context, req diag.Request) error {
	r.calls++
	r.lastRequest = req
	return nil
}

func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// TestDiagWebRunsRegisteredRunner checks CLI flag propagation to runner and log level update.
func TestDiagWebRunsRegisteredRunner(t *testing.T) {
	runner := &recordingRunner{}
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetWeb: runner})

	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	if _, err := executeCommand(cmd, "diag", "web", "--json", "--mtr-count", "3", "--log-level", "debug", "--timeout", "750ms", "--insecure"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if runner.calls != 1 {
		t.Fatalf("expected runner to be called once, got %d", runner.calls)
	}

	if !runner.lastRequest.Options.Global.JSON {
		t.Fatalf("expected JSON flag to propagate to options")
	}

	if runner.lastRequest.Options.Global.MTRCount != 3 {
		t.Fatalf("expected mtr-count=3, got %d", runner.lastRequest.Options.Global.MTRCount)
	}

	if runner.lastRequest.Options.Global.Timeout != 750*time.Millisecond {
		t.Fatalf("expected timeout 750ms, got %v", runner.lastRequest.Options.Global.Timeout)
	}

	if !runner.lastRequest.Options.Global.Insecure {
		t.Fatalf("expected insecure flag to propagate")
	}

	if levelVar.Level() != slog.LevelDebug {
		t.Fatalf("expected log level debug, got %v", levelVar.Level())
	}
}

// TestInvalidMTRCountFailsValidation ensures global validation blocks invalid traceroute counts.
func TestInvalidMTRCountFailsValidation(t *testing.T) {
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetSMTP: &recordingRunner{}})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	if _, err := executeCommand(cmd, "diag", "smtp", "--mtr-count", "0"); err == nil {
		t.Fatalf("expected validation error for mtr-count=0")
	}
}

// TestInvalidTimeoutFailsValidation guards against zero timeout input on CLI.
func TestInvalidTimeoutFailsValidation(t *testing.T) {
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetFTP: &recordingRunner{}})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	if _, err := executeCommand(cmd, "diag", "ftp", "--timeout", "0"); err == nil {
		t.Fatalf("expected validation error for timeout=0")
	}
}

func TestAllTargetsHaveSubcommands(t *testing.T) {
	dispatcher := diag.NewDispatcher(nil)
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	diagCmd, _, err := cmd.Find([]string{"diag"})
	if err != nil {
		t.Fatalf("expected to find diag command: %v", err)
	}

	for _, target := range diag.AllTargets {
		if c, _, err := diagCmd.Find([]string{target.String()}); err != nil || c == nil {
			t.Fatalf("expected subcommand for target %s", target)
		}
	}
}
