package cli

import (
	"io"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
)

// dummyCmd returns a bare cobra.Command suitable for flag-registration tests.
func dummyCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "test",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}
}

// TestTargetFlagRegistrars_ContainsExpectedTargets verifies that the four
// protocol targets that carry extra CLI flags are all present in DefaultRegistrars.
func TestTargetFlagRegistrars_ContainsExpectedTargets(t *testing.T) {
	registrars := DefaultRegistrars()
	withRegistrar := []diag.Target{
		diag.TargetWeb,
		diag.TargetSMTP,
		diag.TargetFTP,
		diag.TargetSFTP,
	}
	for _, target := range withRegistrar {
		if _, ok := registrars[target]; !ok {
			t.Errorf("DefaultRegistrars() missing entry for %s", target)
		}
	}
}

// TestIMAPAndPOPHaveNoRegistrar verifies that IMAP and POP are intentionally
// absent from DefaultRegistrars (they use only shared network flags).
func TestIMAPAndPOPHaveNoRegistrar(t *testing.T) {
	registrars := DefaultRegistrars()
	for _, target := range []diag.Target{diag.TargetIMAP, diag.TargetPOP} {
		if _, ok := registrars[target]; ok {
			t.Errorf("unexpected registrar for %s; only shared network flags should apply", target)
		}
	}
}

// TestRegisterWebFlags_RegistersExpectedFlags verifies that RegisterWebFlags
// attaches the three web-specific flags to the command.
func TestRegisterWebFlags_RegistersExpectedFlags(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	RegisterWebFlags(cmd, opts)

	for _, name := range []string{"dns-domain", "dns-type", "http-url"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("RegisterWebFlags did not register flag --%s", name)
		}
	}
}

// TestRegisterWebFlags_SetsDefaultDomains verifies the initial default value
// injected into opts.Web.Domains before flag binding.
func TestRegisterWebFlags_SetsDefaultDomains(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	RegisterWebFlags(cmd, opts)

	if len(opts.Web.Domains) == 0 {
		t.Error("RegisterWebFlags must set a default domain before flag registration")
	}
}

// TestRegisterWebFlags_PreparerParsesRecordTypes verifies that the returned
// OptionsPreparer converts the default type strings into typed RecordType values.
func TestRegisterWebFlags_PreparerParsesRecordTypes(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	preparer := RegisterWebFlags(cmd, opts)

	if preparer == nil {
		t.Fatal("RegisterWebFlags must return a non-nil OptionsPreparer")
	}
	if err := preparer(opts); err != nil {
		t.Fatalf("preparer returned unexpected error: %v", err)
	}
	if len(opts.Web.Types) == 0 {
		t.Error("expected opts.Web.Types to be populated after running the preparer")
	}
}

// TestRegisterWebFlags_PreparerRejectsInvalidType verifies that an invalid
// record-type string causes the preparer to return an error.
func TestRegisterWebFlags_PreparerRejectsInvalidType(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	preparer := RegisterWebFlags(cmd, opts)

	// Simulate user passing --dns-type=INVALID by overwriting the flag value.
	if err := cmd.Flags().Set("dns-type", "NOTATYPE"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := preparer(opts); err == nil {
		t.Error("expected preparer to return error for invalid record type, got nil")
	}
}

// TestRegisterSMTPFlags_RegistersExpectedFlags verifies that all SMTP flags are registered.
func TestRegisterSMTPFlags_RegistersExpectedFlags(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	preparer := RegisterSMTPFlags(cmd, opts)

	if preparer != nil {
		t.Error("RegisterSMTPFlags should return nil preparer (flags bind directly)")
	}
	for _, name := range []string{
		"smtp-domain", "smtp-user", "smtp-pass", "smtp-from", "smtp-to",
		"smtp-ssl", "smtp-starttls", "smtp-auth-methods", "smtp-mx-all",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("registerSMTPFlags did not register flag --%s", name)
		}
	}
}

// TestRegisterFTPFlags_RegistersExpectedFlags verifies that all FTP flags are registered.
func TestRegisterFTPFlags_RegistersExpectedFlags(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	preparer := RegisterFTPFlags(cmd, opts)

	if preparer != nil {
		t.Error("RegisterFTPFlags should return nil preparer")
	}
	for _, name := range []string{"ftp-user", "ftp-pass", "ftp-ssl", "ftp-auth-tls", "ftp-list"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("registerFTPFlags did not register flag --%s", name)
		}
	}
}

// TestRegisterSFTPFlags_RegistersExpectedFlags verifies that all SFTP flags are registered.
func TestRegisterSFTPFlags_RegistersExpectedFlags(t *testing.T) {
	cmd := dummyCmd()
	opts := &diag.Options{}
	preparer := RegisterSFTPFlags(cmd, opts)

	if preparer != nil {
		t.Error("RegisterSFTPFlags should return nil preparer")
	}
	for _, name := range []string{"sftp-user", "sftp-pass", "sftp-ls"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("registerSFTPFlags did not register flag --%s", name)
		}
	}
}

// TestNewTargetCommand_NoRegistrarTargetsHaveOnlyNetFlags verifies that targets
// without a registrar (IMAP, POP) still receive the shared network flags.
func TestNewTargetCommand_NoRegistrarTargetsHaveOnlyNetFlags(t *testing.T) {
	runner := &recordingRunner{}
	dispatcher := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetIMAP: runner,
		diag.TargetPOP:  runner,
	})
	globalOpts := diag.GlobalOptions{MTRCount: diag.DefaultMTRCount, Timeout: 5e9}
	for _, target := range []diag.Target{diag.TargetIMAP, diag.TargetPOP} {
		silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		cmd := newTargetCommand(target, &globalOpts, dispatcher, DefaultRegistrars(), silentLogger)
		if cmd.Flags().Lookup("target-host") == nil {
			t.Errorf("%s command missing --target-host flag", target)
		}
		if cmd.Flags().Lookup("port") == nil {
			t.Errorf("%s command missing --port flag", target)
		}
		// Protocol-specific flags must NOT be present.
		if cmd.Flags().Lookup("smtp-domain") != nil {
			t.Errorf("%s command must not have --smtp-domain flag", target)
		}
	}
}
