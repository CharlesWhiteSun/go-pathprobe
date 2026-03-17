// Package server — internal unit tests for unexported helper functions.
package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/netprobe"
	"go-pathprobe/pkg/store"
)

// ---- isValidTarget -------------------------------------------------------

func TestIsValidTarget_KnownTargets(t *testing.T) {
	for _, target := range diag.AllTargets {
		if !isValidTarget(target) {
			t.Errorf("isValidTarget(%q) = false, want true", target)
		}
	}
}

func TestIsValidTarget_UnknownTarget(t *testing.T) {
	if isValidTarget(diag.Target("unknown")) {
		t.Error("isValidTarget(\"unknown\") = true, want false")
	}
}

func TestIsValidTarget_EmptyString(t *testing.T) {
	if isValidTarget(diag.Target("")) {
		t.Error("isValidTarget(\"\") = true, want false")
	}
}

// ---- parseDiagTimeout ---------------------------------------------------

func TestParseDiagTimeout_EmptyUsesDefault(t *testing.T) {
	if got := parseDiagTimeout(""); got != defaultDiagTimeout {
		t.Errorf("parseDiagTimeout(\"\") = %v, want %v", got, defaultDiagTimeout)
	}
}

func TestParseDiagTimeout_ValidDuration(t *testing.T) {
	want := 10 * time.Second
	if got := parseDiagTimeout("10s"); got != want {
		t.Errorf("parseDiagTimeout(\"10s\") = %v, want %v", got, want)
	}
}

func TestParseDiagTimeout_InvalidStringFallsBack(t *testing.T) {
	if got := parseDiagTimeout("not-a-duration"); got != defaultDiagTimeout {
		t.Errorf("parseDiagTimeout(invalid) = %v, want %v", got, defaultDiagTimeout)
	}
}

func TestParseDiagTimeout_ZeroFallsBack(t *testing.T) {
	if got := parseDiagTimeout("0s"); got != defaultDiagTimeout {
		t.Errorf("parseDiagTimeout(\"0s\") = %v, want %v", got, defaultDiagTimeout)
	}
}

func TestParseDiagTimeout_NegativeFallsBack(t *testing.T) {
	if got := parseDiagTimeout("-5s"); got != defaultDiagTimeout {
		t.Errorf("parseDiagTimeout(\"-5s\") = %v, want %v", got, defaultDiagTimeout)
	}
}

// ---- buildOptions -------------------------------------------------------

func TestBuildOptions_DefaultsWhenEmpty(t *testing.T) {
	opts, err := buildOptions(ReqOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Global.MTRCount != diag.DefaultMTRCount {
		t.Errorf("MTRCount = %d, want %d", opts.Global.MTRCount, diag.DefaultMTRCount)
	}
	if opts.Global.Timeout != defaultDiagTimeout {
		t.Errorf("Timeout = %v, want %v", opts.Global.Timeout, defaultDiagTimeout)
	}
	if opts.Global.Insecure {
		t.Error("Insecure must default to false")
	}
}

func TestBuildOptions_ExplicitValuesArePreserved(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		MTRCount: 10,
		Timeout:  "15s",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Global.MTRCount != 10 {
		t.Errorf("MTRCount = %d, want 10", opts.Global.MTRCount)
	}
	if opts.Global.Timeout != 15*time.Second {
		t.Errorf("Timeout = %v, want 15s", opts.Global.Timeout)
	}
	if !opts.Global.Insecure {
		t.Error("Insecure must be true")
	}
}

func TestBuildOptions_WebFieldsMapped(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		Web: &ReqWeb{
			Domains: []string{"example.com"},
			Types:   []string{"A", "MX"},
			URL:     "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts.Web.Domains) != 1 || opts.Web.Domains[0] != "example.com" {
		t.Errorf("Web.Domains = %v, want [example.com]", opts.Web.Domains)
	}
	if opts.Web.URL != "https://example.com" {
		t.Errorf("Web.URL = %q", opts.Web.URL)
	}
	if len(opts.Web.Types) != 2 {
		t.Errorf("Web.Types length = %d, want 2", len(opts.Web.Types))
	}
}

func TestBuildOptions_WebInvalidTypesReturnsError(t *testing.T) {
	_, err := buildOptions(ReqOptions{
		Web: &ReqWeb{Types: []string{"TXT"}},
	})
	if err == nil {
		t.Error("expected error for unsupported DNS type, got nil")
	}
}

func TestBuildOptions_NetFieldsMapped(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		Net: &ReqNet{Host: "10.0.0.1", Ports: []int{22, 443}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Net.Host != "10.0.0.1" {
		t.Errorf("Net.Host = %q, want %q", opts.Net.Host, "10.0.0.1")
	}
	if len(opts.Net.Ports) != 2 {
		t.Errorf("Net.Ports length = %d, want 2", len(opts.Net.Ports))
	}
}

func TestBuildOptions_SMTPFieldsMapped(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		SMTP: &ReqSMTP{
			Domain:   "mail.example.com",
			Username: "user",
			Password: "pass",
			StartTLS: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.SMTP.Domain != "mail.example.com" {
		t.Errorf("SMTP.Domain = %q", opts.SMTP.Domain)
	}
	if !opts.SMTP.StartTLS {
		t.Error("SMTP.StartTLS must be true")
	}
}

func TestBuildOptions_FTPFieldsMapped(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		FTP: &ReqFTP{Username: "ftpuser", AuthTLS: true, RunLIST: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.FTP.Username != "ftpuser" {
		t.Errorf("FTP.Username = %q", opts.FTP.Username)
	}
	if !opts.FTP.AuthTLS {
		t.Error("FTP.AuthTLS must be true")
	}
}

func TestBuildOptions_SFTPPrivateKeyNotExposed(t *testing.T) {
	// Ensure that even if an attacker crafts a request with SFTP options,
	// PrivateKey is never populated from the API model.
	opts, err := buildOptions(ReqOptions{
		SFTP: &ReqSFTP{Username: "deploy", RunLS: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.SFTP.PrivateKey != nil {
		t.Error("SFTP.PrivateKey must always be nil when set via the HTTP API")
	}
	if opts.SFTP.Username != "deploy" {
		t.Errorf("SFTP.Username = %q, want %q", opts.SFTP.Username, "deploy")
	}
}

func TestBuildOptions_WebTypesDefaultWhenOmitted(t *testing.T) {
	// When Web is set but Types is nil, Types should remain nil (runner uses defaults).
	opts, err := buildOptions(ReqOptions{
		Web: &ReqWeb{Domains: []string{"example.com"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts.Web.Types) != 0 {
		t.Errorf("Web.Types should be empty when not specified, got %v", opts.Web.Types)
	}
}

// Compile-time assertion: netprobe.ParseRecordTypes is used correctly.
var _ = netprobe.ParseRecordTypes

// ---- resolveLocator (diagPipeline) --------------------------------------

func newTestPipeline(loc geo.Locator) diagPipeline {
	return diagPipeline{
		dispatcher: diag.NewDispatcher(nil),
		locator:    loc,
		store:      store.NewMemoryStore(1),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// TestDiagPipeline_ResolveLocator_FalseReturnsConfigured verifies that when
// DisableGeo is false the pipeline's own locator is returned unchanged.
func TestDiagPipeline_ResolveLocator_FalseReturnsConfigured(t *testing.T) {
	spy := geo.NoopLocator{}
	p := newTestPipeline(spy)
	got := p.resolveLocator(false)
	if got != spy {
		t.Errorf("resolveLocator(false) returned %T, want configured NoopLocator", got)
	}
}

// TestDiagPipeline_ResolveLocator_TrueReturnsNoop verifies that setting
// DisableGeo = true always yields a NoopLocator regardless of the configured
// locator.
func TestDiagPipeline_ResolveLocator_TrueReturnsNoop(t *testing.T) {
	p := newTestPipeline(geo.NoopLocator{})
	got := p.resolveLocator(true)
	if _, ok := got.(geo.NoopLocator); !ok {
		t.Errorf("resolveLocator(true) returned %T, want NoopLocator", got)
	}
}

// ── Phase 4: MaxHops mapping ──────────────────────────────────────────────

// TestBuildOptions_WebMaxHopsMapped verifies that ReqWeb.MaxHops is forwarded
// to diag.WebOptions.MaxHops via buildOptions, ensuring the API field reaches
// the WebTracerouteRunner without the caller having to populate NetworkOptions.
func TestBuildOptions_WebMaxHopsMapped(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		Web: &ReqWeb{Mode: "traceroute", MaxHops: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Web.MaxHops != 20 {
		t.Errorf("Web.MaxHops = %d, want 20", opts.Web.MaxHops)
	}
}

// TestBuildOptions_WebMaxHopsZeroIsPassedThrough verifies that a zero value
// (meaning "use server default") is preserved rather than silently replacing
// with some arbitrary default at the HTTP layer.
func TestBuildOptions_WebMaxHopsZeroIsPassedThrough(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		Web: &ReqWeb{Mode: "traceroute", MaxHops: 0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Web.MaxHops != 0 {
		t.Errorf("Web.MaxHops = %d, want 0 (zero triggers DefaultMaxHops in runner)", opts.Web.MaxHops)
	}
}

// TestBuildOptions_WebTracerouteModeValid verifies that "traceroute" is
// accepted as a valid WebMode by IsValidWebMode so buildOptions does not
// reject it as an unknown string.
func TestBuildOptions_WebTracerouteModeValid(t *testing.T) {
	opts, err := buildOptions(ReqOptions{
		Web: &ReqWeb{Mode: "traceroute"},
	})
	if err != nil {
		t.Fatalf("buildOptions with mode=traceroute must not return an error, got: %v", err)
	}
	if string(opts.Web.Mode) != "traceroute" {
		t.Errorf("Web.Mode = %q, want %q", opts.Web.Mode, "traceroute")
	}
}

// ── tracerouteMinTimeout ─────────────────────────────────────────────────

// TestTracerouteMinTimeout_NonTracerouteMode verifies that a non-traceroute
// request returns 0, meaning "no override needed".
func TestTracerouteMinTimeout_NonTracerouteMode(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeHTTP
	if got := tracerouteMinTimeout(opts); got != 0 {
		t.Errorf("tracerouteMinTimeout(http) = %v, want 0", got)
	}
}

// TestTracerouteMinTimeout_EmptyModeIsNotTraceroute verifies that the legacy
// "all-in-one" mode (empty string) does NOT trigger the traceroute floor,
// because legacy mode does not run a full traceroute.
func TestTracerouteMinTimeout_EmptyModeIsNotTraceroute(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeAll
	if got := tracerouteMinTimeout(opts); got != 0 {
		t.Errorf("tracerouteMinTimeout('') = %v, want 0", got)
	}
}

// TestTracerouteMinTimeout_DefaultSettings verifies the computed minimum when
// maxHops and mtrCount are both zero (runner defaults apply).
// Expected: DefaultMaxHops(30) × DefaultMTRCount(5) × 2 s + 15 s = 315 s.
func TestTracerouteMinTimeout_DefaultSettings(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeTraceroute
	// opts.Web.MaxHops = 0 → defaults to diag.DefaultMaxHops (30)
	// opts.Global.MTRCount = 0 → defaults to diag.DefaultMTRCount (5)
	want := time.Duration(diag.DefaultMaxHops*diag.DefaultMTRCount*2)*time.Second + 15*time.Second
	if got := tracerouteMinTimeout(opts); got != want {
		t.Errorf("tracerouteMinTimeout(default) = %v, want %v", got, want)
	}
}

// TestTracerouteMinTimeout_CustomMaxHops verifies that a custom maxHops value
// produces a proportionally larger minimum.
func TestTracerouteMinTimeout_CustomMaxHops(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeTraceroute
	opts.Web.MaxHops = 10
	opts.Global.MTRCount = 3
	// 10 × 3 × 2 + 15 = 75 s
	want := 75 * time.Second
	if got := tracerouteMinTimeout(opts); got != want {
		t.Errorf("tracerouteMinTimeout(10hops,3mtr) = %v, want %v", got, want)
	}
}

// ── ensureTracerouteTimeout ──────────────────────────────────────────────

// TestEnsureTracerouteTimeout_ExtendsTooShortTimeout verifies that a timeout
// shorter than the computed minimum is replaced by the minimum.
func TestEnsureTracerouteTimeout_ExtendsTooShortTimeout(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeTraceroute
	opts.Web.MaxHops = 10
	opts.Global.MTRCount = 3
	// min = 75 s; providing 30 s should be bumped to 75 s.
	got := ensureTracerouteTimeout(30*time.Second, opts)
	if got != 75*time.Second {
		t.Errorf("ensureTracerouteTimeout(30s) = %v, want 75s", got)
	}
}

// TestEnsureTracerouteTimeout_KeepsSufficientTimeout verifies that a timeout
// already above the minimum is returned unchanged.
func TestEnsureTracerouteTimeout_KeepsSufficientTimeout(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeTraceroute
	opts.Web.MaxHops = 10
	opts.Global.MTRCount = 3
	// min = 75 s; a 120 s timeout should be kept as-is.
	got := ensureTracerouteTimeout(120*time.Second, opts)
	if got != 120*time.Second {
		t.Errorf("ensureTracerouteTimeout(120s) = %v, want 120s unchanged", got)
	}
}

// TestEnsureTracerouteTimeout_NoopForNonTraceroute verifies that non-traceroute
// modes are never modified, even when a very short timeout is given.
func TestEnsureTracerouteTimeout_NoopForNonTraceroute(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeHTTP
	got := ensureTracerouteTimeout(1*time.Second, opts)
	if got != 1*time.Second {
		t.Errorf("ensureTracerouteTimeout(http 1s) = %v, want 1s unchanged", got)
	}
}

// ── fmtDiagError ────────────────────────────────────────────────────────

// TestFmtDiagError_Nil verifies that a nil error returns an empty string.
func TestFmtDiagError_Nil(t *testing.T) {
	if got := fmtDiagError(nil, diag.Options{}); got != "" {
		t.Errorf("fmtDiagError(nil) = %q, want empty string", got)
	}
}

// TestFmtDiagError_DeadlineExceeded_TracerouteMode verifies that the
// context.DeadlineExceeded error is rewritten into a traceroute-specific
// human-readable message that mentions "timed out".
func TestFmtDiagError_DeadlineExceeded_TracerouteMode(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeTraceroute
	opts.Web.MaxHops = 20
	got := fmtDiagError(context.DeadlineExceeded, opts)
	if !strings.Contains(got, "timed out") {
		t.Errorf("fmtDiagError(DeadlineExceeded,traceroute) = %q; must contain 'timed out'", got)
	}
	if strings.Contains(got, "context deadline exceeded") {
		t.Errorf("fmtDiagError must NOT expose raw Go error text 'context deadline exceeded', got: %q", got)
	}
}

// TestFmtDiagError_DeadlineExceeded_OtherMode verifies that the generic
// timeout message is returned for non-traceroute modes.
func TestFmtDiagError_DeadlineExceeded_OtherMode(t *testing.T) {
	opts := diag.Options{}
	opts.Web.Mode = diag.WebModeHTTP
	got := fmtDiagError(context.DeadlineExceeded, opts)
	if !strings.Contains(got, "timed out") {
		t.Errorf("fmtDiagError(DeadlineExceeded,http) = %q; must contain 'timed out'", got)
	}
	if strings.Contains(got, "context deadline exceeded") {
		t.Errorf("fmtDiagError must NOT expose raw Go error text, got: %q", got)
	}
}

// TestFmtDiagError_OtherError verifies that an unrecognised error is
// wrapped with the "diagnostic error: " prefix for frontend parsing.
func TestFmtDiagError_OtherError(t *testing.T) {
	err := errors.New("connection refused")
	got := fmtDiagError(err, diag.Options{})
	if !strings.Contains(got, "connection refused") {
		t.Errorf("fmtDiagError(connection refused) = %q; must contain original message", got)
	}
	if !strings.HasPrefix(got, "diagnostic error:") {
		t.Errorf("fmtDiagError(other) = %q; must start with 'diagnostic error:'", got)
	}
}
