// Package server — internal unit tests for unexported helper functions.
package server

import (
	"testing"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/netprobe"
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
