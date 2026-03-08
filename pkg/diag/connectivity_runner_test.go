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

type stubPortProber struct {
	calls    []int
	lastHost string
}

func (s *stubPortProber) ProbeOnce(ctx context.Context, host string, port int) netprobe.ProbeAttempt {
	s.calls = append(s.calls, port)
	s.lastHost = host
	return netprobe.ProbeAttempt{Success: true, RTT: time.Millisecond}
}

// TestConnectivityRunnerExecutesPortList ensures runner honors ports and host selection.
func TestConnectivityRunnerExecutesPortList(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prober := &stubPortProber{}
	runner := NewConnectivityRunner(prober, logger)

	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 2, Timeout: time.Second},
			Net:    NetworkOptions{Host: "smtp.example.com", Ports: []int{25, 587}},
		},
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(prober.calls) != 4 { // 2 ports * 2 attempts
		t.Fatalf("expected 4 calls, got %d", len(prober.calls))
	}
}

// TestConnectivityRunnerDefaultPorts uses target defaults when ports not provided.
func TestConnectivityRunnerDefaultPorts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prober := &stubPortProber{}
	runner := NewConnectivityRunner(prober, logger)

	req := Request{
		Target:  TargetSFTP,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(prober.calls) != 1 || prober.calls[0] != 22 {
		t.Fatalf("expected default port 22, got %v", prober.calls)
	}
}

// TestConnectivityRunnerNilProber verifies ErrRunnerNotFound is returned when prober is nil.
func TestConnectivityRunnerNilProber(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewConnectivityRunner(nil, logger)
	req := Request{
		Target:  TargetSMTP,
		Options: Options{Global: GlobalOptions{MTRCount: 1}},
	}
	if err := runner.Run(context.Background(), req); !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("expected ErrRunnerNotFound, got %v", err)
	}
}

// TestConnectivityRunnerDefaultHost verifies an empty host defaults to "example.com".
func TestConnectivityRunnerDefaultHost(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prober := &stubPortProber{}
	runner := NewConnectivityRunner(prober, logger)
	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1},
			Net:    NetworkOptions{Ports: []int{25}},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if prober.lastHost != "example.com" {
		t.Fatalf("expected default host %q, got %q", "example.com", prober.lastHost)
	}
}

// TestConnectivityRunnerContextCancel verifies a cancelled context propagates an error.
func TestConnectivityRunnerContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewConnectivityRunner(&stubPortProber{}, logger)
	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1},
			Net:    NetworkOptions{Host: "smtp.example.com", Ports: []int{25}},
		},
	}
	if err := runner.Run(ctx, req); err == nil {
		t.Fatalf("expected context error, got nil")
	}
}
