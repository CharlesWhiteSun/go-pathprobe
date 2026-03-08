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

	if _, err := executeCommand(cmd, "diag", "web", "--json", "--mtr-count", "3", "--log-level", "debug", "--timeout", "750ms", "--insecure", "--target-host", "example.com", "--port", "443", "--dns-domain", "example.com", "--dns-type", "A,MX", "--http-url", "https://example.com"); err != nil {
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

	if runner.lastRequest.Options.Net.Host != "example.com" {
		t.Fatalf("expected target host propagated, got %s", runner.lastRequest.Options.Net.Host)
	}
	if len(runner.lastRequest.Options.Net.Ports) != 1 || runner.lastRequest.Options.Net.Ports[0] != 443 {
		t.Fatalf("expected port 443 propagated")
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

// TestSMTPFlagPropagation ensures SMTP flags bind into options.
func TestSMTPFlagPropagation(t *testing.T) {
	runner := &recordingRunner{}
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetSMTP: runner})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	args := []string{"diag", "smtp", "--smtp-domain", "example.com", "--smtp-user", "user", "--smtp-pass", "pass", "--smtp-from", "from@test", "--smtp-to", "rcpt@test", "--smtp-starttls", "--smtp-ssl", "--port", "587", "--target-host", "mx.test"}
	if _, err := executeCommand(cmd, args...); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected runner called")
	}
	if runner.lastRequest.Options.SMTP.Domain != "example.com" || runner.lastRequest.Options.SMTP.Username != "user" {
		t.Fatalf("expected smtp options propagated")
	}
	if !runner.lastRequest.Options.SMTP.UseTLS || !runner.lastRequest.Options.SMTP.StartTLS {
		t.Fatalf("expected tls/starttls flags set")
	}
	if runner.lastRequest.Options.Net.Host != "mx.test" || runner.lastRequest.Options.Net.Ports[0] != 587 {
		t.Fatalf("expected host/port applied")
	}
}

// TestTargetHostDefault ensures default host/port apply when user omits flags.
func TestTargetHostDefault(t *testing.T) {
	runner := &recordingRunner{}
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetSFTP: runner})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	if _, err := executeCommand(cmd, "diag", "sftp"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected runner called once")
	}
	if runner.lastRequest.Options.Net.Host == "" {
		t.Fatalf("expected default host to be set")
	}
	if len(runner.lastRequest.Options.Net.Ports) == 0 || runner.lastRequest.Options.Net.Ports[0] != 22 {
		t.Fatalf("expected default port 22, got %v", runner.lastRequest.Options.Net.Ports)
	}
}

// TestReportFlagPropagates verifies --report value is passed through to GlobalOptions.
func TestReportFlagPropagates(t *testing.T) {
	runner := &recordingRunner{}
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetWeb: runner})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	if _, err := executeCommand(cmd, "diag", "web", "--report", "output.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if runner.lastRequest.Options.Global.Report != "output.json" {
		t.Fatalf("expected report path 'output.json', got %q", runner.lastRequest.Options.Global.Report)
	}
}

// TestDispatcherRunnerNotFoundReturnsError verifies that a target with no runner returns an error.
func TestDispatcherRunnerNotFoundReturnsError(t *testing.T) {
	dispatcher := diag.NewDispatcher(nil) // no runners registered
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	_, err := executeCommand(cmd, "diag", "imap")
	if err == nil {
		t.Fatal("expected error when no runner registered for target 'imap'")
	}
}

// TestInvalidLogLevelReturnsError verifies that unknown log level values are rejected before dispatch.
func TestInvalidLogLevelReturnsError(t *testing.T) {
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetWeb: &recordingRunner{}})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	_, err := executeCommand(cmd, "--log-level", "verbose", "diag", "web")
	if err == nil {
		t.Fatal("expected error for unknown log level 'verbose'")
	}
}
