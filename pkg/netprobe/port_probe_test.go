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

// TestProbePortsNegativeAttempts validates negative attempts guard.
func TestProbePortsNegativeAttempts(t *testing.T) {
	_, err := ProbePorts(context.Background(), "example.com", []int{443}, -1, &fakeProber{})
	if err == nil {
		t.Fatalf("expected error for negative attempts")
	}
}

// TestProbePortsNilProber validates nil prober guard.
func TestProbePortsNilProber(t *testing.T) {
	_, err := ProbePorts(context.Background(), "example.com", []int{443}, 1, nil)
	if err == nil {
		t.Fatalf("expected error for nil prober")
	}
}

// TestProbePortsEmptyPorts verifies empty port list returns an empty result slice.
func TestProbePortsEmptyPorts(t *testing.T) {
	res, err := ProbePorts(context.Background(), "example.com", []int{}, 1, &fakeProber{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected empty results, got %d", len(res))
	}
}

// TestProbePortsAllFailures verifies 100% loss and zero RTTs when every attempt fails.
func TestProbePortsAllFailures(t *testing.T) {
	fp := &fakeProber{attempts: []ProbeAttempt{
		{Success: false, Err: errors.New("refused")},
		{Success: false, Err: errors.New("refused")},
	}}
	res, err := ProbePorts(context.Background(), "example.com", []int{80}, 2, fp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	stats := res[0].Stats
	if stats.LossPct != 100.0 {
		t.Fatalf("expected 100%% loss, got %v", stats.LossPct)
	}
	if stats.MinRTT != 0 || stats.MaxRTT != 0 || stats.AvgRTT != 0 {
		t.Fatalf("expected zero RTTs on all-failure, got min=%v max=%v avg=%v", stats.MinRTT, stats.MaxRTT, stats.AvgRTT)
	}
	if stats.LastErrStr == "" {
		t.Fatalf("expected LastErrStr to be set")
	}
}

// TestProbePortsSingleSuccess verifies min==max==avg RTT when there is exactly one success.
func TestProbePortsSingleSuccess(t *testing.T) {
	fp := &fakeProber{attempts: []ProbeAttempt{
		{RTT: 5 * time.Millisecond, Success: true},
	}}
	res, err := ProbePorts(context.Background(), "example.com", []int{443}, 1, fp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	stats := res[0].Stats
	if stats.MinRTT != 5*time.Millisecond || stats.MaxRTT != 5*time.Millisecond || stats.AvgRTT != 5*time.Millisecond {
		t.Fatalf("expected equal RTTs, got min=%v max=%v avg=%v", stats.MinRTT, stats.MaxRTT, stats.AvgRTT)
	}
	if stats.LossPct != 0 {
		t.Fatalf("expected 0%% loss, got %v", stats.LossPct)
	}
}

// TestProbePortsMultiplePorts verifies each port receives independent stats.
func TestProbePortsMultiplePorts(t *testing.T) {
	fp := &fakeProber{attempts: []ProbeAttempt{
		{RTT: 10 * time.Millisecond, Success: true},          // port 80
		{RTT: 0, Success: false, Err: errors.New("refused")}, // port 443
	}}
	res, err := ProbePorts(context.Background(), "example.com", []int{80, 443}, 1, fp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 port results, got %d", len(res))
	}
	if res[0].Port != 80 || res[0].Stats.LossPct != 0 {
		t.Fatalf("expected port 80 with 0%% loss, got port=%d loss=%v", res[0].Port, res[0].Stats.LossPct)
	}
	if res[1].Port != 443 || res[1].Stats.LossPct != 100 {
		t.Fatalf("expected port 443 with 100%% loss, got port=%d loss=%v", res[1].Port, res[1].Stats.LossPct)
	}
}

// TestProbePortsContextCancel verifies ProbePorts returns ctx.Err() when context is cancelled.
func TestProbePortsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ProbePorts(ctx, "example.com", []int{80, 443}, 3, &fakeProber{})
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
}
