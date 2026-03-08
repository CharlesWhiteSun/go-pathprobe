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

type stubHTTPProber struct {
	called bool
	url    string
}

func (s *stubHTTPProber) Probe(ctx context.Context, req netprobe.HTTPProbeRequest) (netprobe.HTTPProbeResult, error) {
	s.called = true
	s.url = req.URL
	return netprobe.HTTPProbeResult{StatusCode: 200, RTT: time.Millisecond}, nil
}

// TestHTTPRunnerUsesURL ensures URL resolution and insecure/timeout propagation.
func TestHTTPRunnerUsesURL(t *testing.T) {
	prober := &stubHTTPProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewHTTPRunner(prober, logger)

	req := Request{Target: TargetWeb, Options: Options{
		Global: GlobalOptions{Timeout: 2 * time.Second, Insecure: true, MTRCount: 1},
		Web:    WebOptions{URL: "https://example.com"},
		Net:    NetworkOptions{Host: "example.com"},
	}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !prober.called || prober.url != "https://example.com" {
		t.Fatalf("expected prober called with url, got %s", prober.url)
	}
}

// TestHTTPRunnerDefaultURL uses host fallback when URL empty.
func TestHTTPRunnerDefaultURL(t *testing.T) {
	prober := &stubHTTPProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewHTTPRunner(prober, logger)

	req := Request{Target: TargetWeb, Options: Options{Global: GlobalOptions{Timeout: time.Second, MTRCount: 1}, Net: NetworkOptions{Host: "my.test"}}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if prober.url != "https://my.test" {
		t.Fatalf("expected derived url, got %s", prober.url)
	}
}

// errHTTPProber is a stub that always returns a configured error.
type errHTTPProber struct{ err error }

func (e *errHTTPProber) Probe(_ context.Context, _ netprobe.HTTPProbeRequest) (netprobe.HTTPProbeResult, error) {
	return netprobe.HTTPProbeResult{}, e.err
}

// TestHTTPRunnerNilProber verifies ErrRunnerNotFound is returned when prober is nil.
func TestHTTPRunnerNilProber(t *testing.T) {
	runner := &HTTPRunner{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := runner.Run(context.Background(), Request{Target: TargetWeb}); !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("expected ErrRunnerNotFound, got %v", err)
	}
}

// TestHTTPRunnerProberError verifies prober errors are propagated to the caller.
func TestHTTPRunnerProberError(t *testing.T) {
	prober := &errHTTPProber{err: errors.New("probe failed")}
	runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := Request{Target: TargetWeb, Options: Options{
		Global: GlobalOptions{Timeout: time.Second},
		Web:    WebOptions{URL: "https://example.com"},
	}}
	if err := runner.Run(context.Background(), req); err == nil {
		t.Fatalf("expected prober error, got nil")
	}
}

// TestHTTPRunnerDefaultFallbackHost verifies that when both URL and host are empty,
// the URL defaults to "https://example.com".
func TestHTTPRunnerDefaultFallbackHost(t *testing.T) {
	prober := &stubHTTPProber{}
	runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := Request{Target: TargetWeb, Options: Options{Global: GlobalOptions{Timeout: time.Second}}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if prober.url != "https://example.com" {
		t.Fatalf("expected fallback url, got %s", prober.url)
	}
}
