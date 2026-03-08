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
