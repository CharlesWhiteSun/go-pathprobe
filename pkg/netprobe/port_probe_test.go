package netprobe

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProber struct {
	attempts []ProbeAttempt
	idx      int
}

func (f *fakeProber) ProbeOnce(ctx context.Context, host string, port int) ProbeAttempt {
	defer func() { f.idx++ }()
	if f.idx >= len(f.attempts) {
		return ProbeAttempt{Err: errors.New("out of range")}
	}
	return f.attempts[f.idx]
}

// TestProbePortsStats verifies loss/RTT stats and port grouping.
func TestProbePortsStats(t *testing.T) {
	attempts := []ProbeAttempt{
		{RTT: 10 * time.Millisecond, Success: true},
		{RTT: 20 * time.Millisecond, Success: true},
		{RTT: 0, Success: false, Err: errors.New("timeout")},
	}
	fp := &fakeProber{attempts: attempts}
	res, err := ProbePorts(context.Background(), "example.com", []int{443}, len(attempts), fp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected one port result")
	}
	stats := res[0].Stats
	if stats.Sent != 3 || stats.Received != 2 {
		t.Fatalf("unexpected sent/received: %d/%d", stats.Sent, stats.Received)
	}
	if stats.LossPct != 33.333333333333336 { // 1/3 loss
		t.Fatalf("unexpected loss pct: %v", stats.LossPct)
	}
	if stats.MinRTT != 10*time.Millisecond || stats.MaxRTT != 20*time.Millisecond {
		t.Fatalf("unexpected rtt range: min %v max %v", stats.MinRTT, stats.MaxRTT)
	}
	if stats.AvgRTT != 15*time.Millisecond {
		t.Fatalf("unexpected avg rtt: %v", stats.AvgRTT)
	}
}

// TestProbePortsAttemptsZero validates attempts guard.
func TestProbePortsAttemptsZero(t *testing.T) {
	_, err := ProbePorts(context.Background(), "example.com", []int{443}, 0, &fakeProber{})
	if err == nil {
		t.Fatalf("expected error for zero attempts")
	}
}
