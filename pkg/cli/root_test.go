package cli

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
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
// The report is written to a temporary directory so that no artefacts are left
// inside the source tree after the test run.
func TestReportFlagPropagates(t *testing.T) {
	runner := &recordingRunner{}
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{diag.TargetWeb: runner})
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	reportPath := filepath.Join(t.TempDir(), "report.html")
	if _, err := executeCommand(cmd, "diag", "web", "--report", reportPath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if runner.lastRequest.Options.Global.Report != reportPath {
		t.Fatalf("expected report path %q, got %q", reportPath, runner.lastRequest.Options.Global.Report)
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

// ── Default-serve / browser-open tests ───────────────────────────────────

// TestRootCommandHasDefaultServe verifies that the root command has a RunE
// set so that running the binary without a subcommand starts the server
// instead of printing help.
func TestRootCommandHasDefaultServe(t *testing.T) {
	dispatcher := diag.NewDispatcher(nil)
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	cmd := NewRootCommand(dispatcher, logger, levelVar)

	if cmd.RunE == nil {
		t.Fatal("root command must have RunE set so that bare invocation starts the server")
	}
}

// TestServeCommandFlags verifies that the serve subcommand exposes the expected
// flags with their documented defaults.
func TestServeCommandFlags(t *testing.T) {
	dispatcher := diag.NewDispatcher(nil)
	logger, levelVar := logging.NewLogger(slog.LevelInfo)
	rootCmd := NewRootCommand(dispatcher, logger, levelVar)

	serveCmd, _, err := rootCmd.Find([]string{"serve"})
	if err != nil || serveCmd == nil {
		t.Fatalf("expected 'serve' subcommand to be registered: %v", err)
	}

	// --open flag must exist and default to true.
	openFlag := serveCmd.Flags().Lookup("open")
	if openFlag == nil {
		t.Fatal("serve command must have --open flag")
	}
	if openFlag.DefValue != "true" {
		t.Errorf("--open default must be true, got %q", openFlag.DefValue)
	}

	// --addr flag must exist.
	if serveCmd.Flags().Lookup("addr") == nil {
		t.Fatal("serve command must have --addr flag")
	}

	// --write-timeout flag must exist.
	if serveCmd.Flags().Lookup("write-timeout") == nil {
		t.Fatal("serve command must have --write-timeout flag")
	}
}

// TestServerURL verifies the URL derivation helper for various listen addresses.
func TestServerURL(t *testing.T) {
	cases := []struct {
		addr string
		want string
	}{
		{":8080", "http://localhost:8080"},
		{"0.0.0.0:9090", "http://localhost:9090"},
		{"[::]:8080", "http://localhost:8080"},
		{"127.0.0.1:8888", "http://127.0.0.1:8888"},
		{"192.168.1.5:3000", "http://192.168.1.5:3000"},
	}
	for _, tc := range cases {
		got := serverURL(tc.addr)
		if got != tc.want {
			t.Errorf("serverURL(%q) = %q, want %q", tc.addr, got, tc.want)
		}
	}
}

// TestServeOpenerCalledOnStart verifies that the injected opener function is
// invoked when the serve command is started with --open=true (the default).
// The test replaces the real server with a context-cancelled run to avoid
// binding a real TCP port.
func TestServeOpenerCalledOnStart(t *testing.T) {
	var opened []string
	recorder := func(url string) error {
		opened = append(opened, url)
		return nil
	}

	dispatcher := diag.NewDispatcher(nil)
	logger, _ := logging.NewLogger(slog.LevelInfo)
	cmd := newServeCommand(dispatcher, &diag.GlobalOptions{MTRCount: 5, Timeout: 5 * 1e9}, logger, recorder)

	// Cancel the context immediately so the server exits without binding.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--open=true"})
	// srv.Start() returns ErrServerClosed for a cancelled context – that is fine.
	_ = cmd.Execute()

	// Allow the browser goroutine (300 ms sleep) to fire.
	time.Sleep(500 * time.Millisecond)

	if len(opened) == 0 {
		t.Fatal("expected opener to be called at least once when --open=true")
	}
	if opened[0] != "http://localhost:8080" {
		t.Errorf("expected opener called with http://localhost:8080, got %q", opened[0])
	}
}

// TestServeOpenerSkippedWhenOpenFalse verifies that the browser is NOT opened
// when the user passes --open=false.
func TestServeOpenerSkippedWhenOpenFalse(t *testing.T) {
	var opened []string
	recorder := func(url string) error {
		opened = append(opened, url)
		return nil
	}

	dispatcher := diag.NewDispatcher(nil)
	logger, _ := logging.NewLogger(slog.LevelInfo)
	cmd := newServeCommand(dispatcher, &diag.GlobalOptions{MTRCount: 5, Timeout: 5 * 1e9}, logger, recorder)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--open=false"})
	_ = cmd.Execute()

	time.Sleep(500 * time.Millisecond)

	if len(opened) != 0 {
		t.Fatalf("expected opener NOT to be called when --open=false, but got calls: %v", opened)
	}
}
