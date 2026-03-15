package netprobe

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ── stub prober ──────────────────────────────────────────────────────────────

// stubProber is a TracerouteProber that returns a pre-crafted RouteResult
// without any real network I/O.  Used to keep tests hermetic.
type stubProber struct {
	result RouteResult
	err    error
}

func (s *stubProber) Trace(_ context.Context, _ string, _, _ int) (RouteResult, error) {
	return s.result, s.err
}

// compile-time check: stubProber satisfies TracerouteProber
var _ TracerouteProber = (*stubProber)(nil)
var _ TracerouteProber = (*ICMPTracerouteProber)(nil)
var _ TracerouteProber = (*TCPTracerouteProber)(nil)

// ── RouteResult / HopResult helpers ─────────────────────────────────────────

func TestRouteResult_Empty(t *testing.T) {
	r := RouteResult{}
	if len(r.Hops) != 0 {
		t.Fatalf("expected empty hops, got %d", len(r.Hops))
	}
}

func TestHopResult_TimedOut(t *testing.T) {
	hop := HopResult{TTL: 3, IP: "", Hostname: ""}
	if hop.IP != "" {
		t.Fatalf("timed-out hop should have empty IP, got %q", hop.IP)
	}
	if hop.Hostname != "" {
		t.Fatalf("timed-out hop should have empty Hostname, got %q", hop.Hostname)
	}
}

// ── ProbeStats calculation (reuses computeStats) ────────────────────────────

func TestComputeStats_AllSuccess(t *testing.T) {
	attempts := []ProbeAttempt{
		{RTT: 10 * time.Millisecond, Success: true},
		{RTT: 20 * time.Millisecond, Success: true},
		{RTT: 30 * time.Millisecond, Success: true},
	}
	s := computeStats(attempts)

	assertDuration(t, "MinRTT", s.MinRTT, 10*time.Millisecond)
	assertDuration(t, "MaxRTT", s.MaxRTT, 30*time.Millisecond)
	assertDuration(t, "AvgRTT", s.AvgRTT, 20*time.Millisecond)
	if s.LossPct != 0 {
		t.Fatalf("expected 0%% loss, got %.2f%%", s.LossPct)
	}
	if s.Sent != 3 {
		t.Fatalf("expected Sent=3, got %d", s.Sent)
	}
	if s.Received != 3 {
		t.Fatalf("expected Received=3, got %d", s.Received)
	}
}

func TestComputeStats_AllFailed(t *testing.T) {
	attempts := []ProbeAttempt{
		{Success: false, Err: errors.New("timeout")},
		{Success: false, Err: errors.New("timeout")},
	}
	s := computeStats(attempts)

	if s.LossPct != 100 {
		t.Fatalf("expected 100%% loss, got %.2f%%", s.LossPct)
	}
	if s.Received != 0 {
		t.Fatalf("expected Received=0, got %d", s.Received)
	}
	// Min/Avg/Max should be zero when no successes.
	if s.MinRTT != 0 {
		t.Fatalf("expected MinRTT=0 on all-failed, got %v", s.MinRTT)
	}
}

func TestComputeStats_PartialSuccess(t *testing.T) {
	attempts := []ProbeAttempt{
		{RTT: 100 * time.Millisecond, Success: true},
		{Success: false},
		{RTT: 200 * time.Millisecond, Success: true},
		{Success: false},
	}
	s := computeStats(attempts)

	if s.Sent != 4 {
		t.Fatalf("expected Sent=4, got %d", s.Sent)
	}
	if s.Received != 2 {
		t.Fatalf("expected Received=2, got %d", s.Received)
	}
	if s.LossPct != 50 {
		t.Fatalf("expected 50%% loss, got %.2f%%", s.LossPct)
	}
	assertDuration(t, "AvgRTT", s.AvgRTT, 150*time.Millisecond)
}

func TestComputeStats_Empty(t *testing.T) {
	s := computeStats(nil)
	if s.Sent != 0 || s.Received != 0 || s.LossPct != 0 {
		t.Fatalf("empty attempts should yield zero stats, got %+v", s)
	}
}

// ── stubProber integration ───────────────────────────────────────────────────

func TestStubProber_ReturnsError(t *testing.T) {
	want := errors.New("network unreachable")
	p := &stubProber{err: want}

	_, err := p.Trace(context.Background(), "example.com", 10, 3)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestStubProber_ReturnsResult(t *testing.T) {
	hops := []HopResult{
		{TTL: 1, IP: "192.168.1.1", Stats: computeStats([]ProbeAttempt{{RTT: 5 * time.Millisecond, Success: true}})},
		{TTL: 2, IP: "10.0.0.1", Stats: computeStats([]ProbeAttempt{{RTT: 10 * time.Millisecond, Success: true}})},
		{TTL: 3, IP: "93.184.216.34", Stats: computeStats([]ProbeAttempt{{RTT: 50 * time.Millisecond, Success: true}})},
	}
	p := &stubProber{result: RouteResult{Hops: hops}}

	result, err := p.Trace(context.Background(), "example.com", 30, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Hops) != 3 {
		t.Fatalf("expected 3 hops, got %d", len(result.Hops))
	}
	for i, hop := range result.Hops {
		if hop.TTL != i+1 {
			t.Errorf("hop %d: expected TTL=%d, got %d", i, i+1, hop.TTL)
		}
	}
}

// ── context cancellation ─────────────────────────────────────────────────────

func TestStubProber_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	p := &stubProber{result: RouteResult{}, err: ctx.Err()}
	_, err := p.Trace(ctx, "example.com", 30, 3)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// ── ICMPTracerouteProber config ───────────────────────────────────────────────

func TestICMPProber_DefaultHopTimeout(t *testing.T) {
	p := &ICMPTracerouteProber{}
	if got := p.hopTimeoutFor(); got != hopTimeout {
		t.Fatalf("expected default hopTimeout %v, got %v", hopTimeout, got)
	}
}

func TestICMPProber_CustomHopTimeout(t *testing.T) {
	d := 500 * time.Millisecond
	p := &ICMPTracerouteProber{HopTimeout: d}
	if got := p.hopTimeoutFor(); got != d {
		t.Fatalf("expected %v, got %v", d, got)
	}
}

func TestICMPProber_ReverseLookupDefault(t *testing.T) {
	p := &ICMPTracerouteProber{}
	if !p.reverseLookupEnabled() {
		t.Fatal("expected reverse lookup enabled by default")
	}
}

func TestICMPProber_ReverseLookupDisabled(t *testing.T) {
	v := false
	p := &ICMPTracerouteProber{ReverseLookup: &v}
	if p.reverseLookupEnabled() {
		t.Fatal("expected reverse lookup disabled")
	}
}

// ── TCPTracerouteProber config ────────────────────────────────────────────────

func TestTCPProber_DefaultPort(t *testing.T) {
	p := &TCPTracerouteProber{}
	if got := p.remotePort(); got != 80 {
		t.Fatalf("expected default port 80, got %d", got)
	}
}

func TestTCPProber_CustomPort(t *testing.T) {
	p := &TCPTracerouteProber{RemotePort: 443}
	if got := p.remotePort(); got != 443 {
		t.Fatalf("expected port 443, got %d", got)
	}
}

func TestTCPProber_InvalidPortFallback(t *testing.T) {
	p := &TCPTracerouteProber{RemotePort: 99999}
	if got := p.remotePort(); got != 80 {
		t.Fatalf("out-of-range port should fall back to 80, got %d", got)
	}
}

// ── validation: Trace rejects bad arguments ──────────────────────────────────

func TestICMPProber_InvalidMaxHops(t *testing.T) {
	p := &ICMPTracerouteProber{}
	_, err := p.Trace(context.Background(), "127.0.0.1", 0, 1)
	if err == nil {
		t.Fatal("expected error for maxHops=0")
	}
}

func TestICMPProber_InvalidAttempts(t *testing.T) {
	p := &ICMPTracerouteProber{}
	_, err := p.Trace(context.Background(), "127.0.0.1", 1, 0)
	if err == nil {
		t.Fatal("expected error for attemptsPerHop=0")
	}
}

func TestTCPProber_InvalidMaxHops(t *testing.T) {
	p := &TCPTracerouteProber{}
	_, err := p.Trace(context.Background(), "127.0.0.1", 0, 1)
	if err == nil {
		t.Fatal("expected error for maxHops=0")
	}
}

func TestTCPProber_InvalidAttempts(t *testing.T) {
	p := &TCPTracerouteProber{}
	_, err := p.Trace(context.Background(), "127.0.0.1", 1, 0)
	if err == nil {
		t.Fatal("expected error for attemptsPerHop=0")
	}
}

// ── reverseLookup (unit, does real DNS but is best-effort) ──────────────────

func TestReverseLookup_Loopback(t *testing.T) {
	// 127.0.0.1 may or may not resolve; either outcome is acceptable.
	// The important thing is that the function does not panic or block forever.
	done := make(chan struct{})
	go func() {
		_ = reverseLookup("127.0.0.1")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(reverseHostnameTimeout + 200*time.Millisecond):
		t.Fatal("reverseLookup exceeded expected timeout + buffer")
	}
}

// ── icmpID uniqueness ─────────────────────────────────────────────────────────

func TestICMPID_Range(t *testing.T) {
	id := icmpID()
	if id < 0 || id > 0xFFFF {
		t.Fatalf("icmpID out of 16-bit range: %d", id)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func assertDuration(t *testing.T, name string, got, want time.Duration) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %v, got %v", name, want, got)
	}
}
