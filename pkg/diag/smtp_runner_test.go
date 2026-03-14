package diag

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"go-pathprobe/pkg/netprobe"
)

type stubSMTPProber struct {
	called   bool
	lastHost string
	lastPort int
}

func (s *stubSMTPProber) Probe(ctx context.Context, req netprobe.SMTPProbeRequest) (netprobe.SMTPProbeResult, error) {
	s.called = true
	s.lastHost = req.Host
	s.lastPort = req.Port
	return netprobe.SMTPProbeResult{Banner: "220 test"}, nil
}

type failingResolver struct{}

func (f *failingResolver) Lookup(ctx context.Context, name string, rtype netprobe.RecordType) (netprobe.DNSAnswer, error) {
	return netprobe.DNSAnswer{}, errors.New("fail")
}

type mxResolver struct{}

func (m *mxResolver) Lookup(ctx context.Context, name string, rtype netprobe.RecordType) (netprobe.DNSAnswer, error) {
	return netprobe.DNSAnswer{Values: []string{"mx.test:10"}}, nil
}

// TestSMTPRunnerUsesMXFallback ensures MX resolution is used when host empty.
func TestSMTPRunnerUsesMXFallback(t *testing.T) {
	prober := &stubSMTPProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewSMTPRunner(prober, &mxResolver{}, logger)

	req := Request{Target: TargetSMTP, Options: Options{Global: GlobalOptions{Timeout: time.Second, MTRCount: 1}, SMTP: SMTPOptions{Domain: "example.com"}}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !prober.called || prober.lastHost != "mx.test" {
		t.Fatalf("expected mx host used, got %s", prober.lastHost)
	}
	if prober.lastPort != 25 {
		t.Fatalf("expected default port 25, got %d", prober.lastPort)
	}
}

// TestSMTPRunnerWithExplicitHost uses provided host/port.
func TestSMTPRunnerWithExplicitHost(t *testing.T) {
	prober := &stubSMTPProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewSMTPRunner(prober, &failingResolver{}, logger)

	req := Request{Target: TargetSMTP, Options: Options{Global: GlobalOptions{Timeout: time.Second, MTRCount: 1}, Net: NetworkOptions{Host: "mail.test", Ports: []int{587}}, SMTP: SMTPOptions{Domain: "example.com", StartTLS: true}}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if prober.lastHost != "mail.test" || prober.lastPort != 587 {
		t.Fatalf("expected explicit host/port used")
	}
}

// TestSMTPRunnerNilProber verifies ErrRunnerNotFound is returned when the prober is nil.
func TestSMTPRunnerNilProber(t *testing.T) {
	runner := &SMTPRunner{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	err := runner.Run(context.Background(), Request{Target: TargetSMTP})
	if !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("expected ErrRunnerNotFound, got %v", err)
	}
}

// multiCaptureProber records every host it is asked to probe.
type multiCaptureProber struct {
	hosts []string
}

func (m *multiCaptureProber) Probe(_ context.Context, req netprobe.SMTPProbeRequest) (netprobe.SMTPProbeResult, error) {
	m.hosts = append(m.hosts, req.Host)
	return netprobe.SMTPProbeResult{}, nil
}

// mxMultiResolver returns two MX records to simulate a domain with multiple mail servers.
type mxMultiResolver struct{}

func (r *mxMultiResolver) Lookup(_ context.Context, _ string, _ netprobe.RecordType) (netprobe.DNSAnswer, error) {
	return netprobe.DNSAnswer{Values: []string{"mx1.test:10", "mx2.test:20"}}, nil
}

// TestSMTPRunnerMXProbeAll verifies all MX records are probed when MXProbeAll is enabled.
func TestSMTPRunnerMXProbeAll(t *testing.T) {
	prober := &multiCaptureProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewSMTPRunner(prober, &mxMultiResolver{}, logger)

	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			SMTP:   SMTPOptions{Domain: "example.com", MXProbeAll: true},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prober.hosts) != 2 || prober.hosts[0] != "mx1.test" || prober.hosts[1] != "mx2.test" {
		t.Fatalf("expected both MX hosts probed in order, got %v", prober.hosts)
	}
}

// authCaptureProber captures the auth methods forwarded in the probe request.
type authCaptureProber struct {
	methods []string
}

func (a *authCaptureProber) Probe(_ context.Context, req netprobe.SMTPProbeRequest) (netprobe.SMTPProbeResult, error) {
	a.methods = req.AuthMethods
	return netprobe.SMTPProbeResult{}, nil
}

// smtpFullCaptureProber captures the full SMTPProbeRequest for assertion.
type smtpFullCaptureProber struct {
	last netprobe.SMTPProbeRequest
}

func (p *smtpFullCaptureProber) Probe(_ context.Context, req netprobe.SMTPProbeRequest) (netprobe.SMTPProbeResult, error) {
	p.last = req
	return netprobe.SMTPProbeResult{}, nil
}

// TestIsValidSMTPMode verifies mode validation.
func TestIsValidSMTPMode(t *testing.T) {
	for _, tc := range []struct {
		m    SMTPMode
		want bool
	}{
		{"", true},
		{"handshake", true},
		{"auth", true},
		{"send", true},
		{"invalid", false},
	} {
		if got := IsValidSMTPMode(tc.m); got != tc.want {
			t.Errorf("IsValidSMTPMode(%q) = %v, want %v", tc.m, got, tc.want)
		}
	}
}

// TestSMTPHandshakeModeStripsCredentials verifies that handshake mode clears auth/from/to.
func TestSMTPHandshakeModeStripsCredentials(t *testing.T) {
	prober := &smtpFullCaptureProber{}
	runner := NewSMTPRunner(prober, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "mail.test", Ports: []int{25}},
			SMTP: SMTPOptions{
				Mode:     SMTPModeHandshake,
				Username: "user",
				Password: "secret",
				From:     "from@test",
				To:       []string{"to@test"},
				Domain:   "example.com",
			},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prober.last.Username != "" || prober.last.Password != "" {
		t.Errorf("handshake mode must clear credentials, got user=%q pass=%q", prober.last.Username, prober.last.Password)
	}
	if prober.last.From != "" || len(prober.last.To) != 0 {
		t.Errorf("handshake mode must clear from/to, got from=%q to=%v", prober.last.From, prober.last.To)
	}
}

// TestSMTPAuthModeStripsFromTo verifies that auth mode clears from/to but preserves credentials.
func TestSMTPAuthModeStripsFromTo(t *testing.T) {
	prober := &smtpFullCaptureProber{}
	runner := NewSMTPRunner(prober, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "mail.test", Ports: []int{587}},
			SMTP: SMTPOptions{
				Mode:     SMTPModeAuth,
				Username: "user",
				Password: "secret",
				From:     "from@test",
				To:       []string{"to@test"},
			},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prober.last.Username != "user" || prober.last.Password != "secret" {
		t.Errorf("auth mode must preserve credentials, got user=%q pass=%q", prober.last.Username, prober.last.Password)
	}
	if prober.last.From != "" || len(prober.last.To) != 0 {
		t.Errorf("auth mode must clear from/to, got from=%q to=%v", prober.last.From, prober.last.To)
	}
}

// TestSMTPSendModePreservesAll verifies that send mode passes all fields through.
func TestSMTPSendModePreservesAll(t *testing.T) {
	prober := &smtpFullCaptureProber{}
	runner := NewSMTPRunner(prober, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "mail.test", Ports: []int{587}},
			SMTP: SMTPOptions{
				Mode:     SMTPModeSend,
				Username: "user",
				Password: "secret",
				From:     "from@test",
				To:       []string{"to@test"},
			},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prober.last.Username != "user" || prober.last.Password != "secret" {
		t.Errorf("send mode must preserve credentials, got user=%q pass=%q", prober.last.Username, prober.last.Password)
	}
	if prober.last.From != "from@test" || len(prober.last.To) != 1 || prober.last.To[0] != "to@test" {
		t.Errorf("send mode must preserve from/to, got from=%q to=%v", prober.last.From, prober.last.To)
	}
}

// TestSMTPRunnerAuthMethodsPropagated verifies AuthMethods from SMTPOptions flow into the prober.
func TestSMTPRunnerAuthMethodsPropagated(t *testing.T) {
	prober := &authCaptureProber{}
	runner := NewSMTPRunner(prober, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Net:    NetworkOptions{Host: "mail.test", Ports: []int{587}},
			SMTP:   SMTPOptions{AuthMethods: []string{"XOAUTH2", "PLAIN"}},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prober.methods) != 2 || prober.methods[0] != "XOAUTH2" || prober.methods[1] != "PLAIN" {
		t.Fatalf("expected [XOAUTH2,PLAIN], got %v", prober.methods)
	}
}
