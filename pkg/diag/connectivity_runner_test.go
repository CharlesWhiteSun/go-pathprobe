package diag

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"go-pathprobe/pkg/netprobe"
)

type stubPortProber struct {
	calls []int
}

func (s *stubPortProber) ProbeOnce(ctx context.Context, host string, port int) netprobe.ProbeAttempt {
	s.calls = append(s.calls, port)
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
