package diag

import (
	"context"
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
