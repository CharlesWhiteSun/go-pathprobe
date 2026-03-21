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

// ── stub TracerouteProber ────────────────────────────────────────────────────

// stubTracerouteProber is a hermetic TracerouteProber used in tests.
// It records the arguments it received and returns the configured result.
type stubTracerouteProber struct {
	result         netprobe.RouteResult
	err            error
	calledHost     string
	calledMaxHops  int
	calledAttempts int
}

func (s *stubTracerouteProber) Trace(_ context.Context, host string, maxHops, attemptsPerHop int, onHop netprobe.HopEmitter) (netprobe.RouteResult, error) {
	s.calledHost = host
	s.calledMaxHops = maxHops
	s.calledAttempts = attemptsPerHop
	if onHop != nil {
		for _, hop := range s.result.Hops {
			onHop(hop)
		}
	}
	return s.result, s.err
}

// build3HopResult returns a canonical 3-hop RouteResult for reuse across tests.
func build3HopResult() netprobe.RouteResult {
	return netprobe.RouteResult{
		Hops: []netprobe.HopResult{
			{TTL: 1, IP: "192.168.1.1", Hostname: "gw.local",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 3, AvgRTT: 2 * time.Millisecond}},
			{TTL: 2, IP: "", Hostname: "",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 0, LossPct: 100}}, // timed-out
			{TTL: 3, IP: "93.184.216.34", Hostname: "example.com",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 3, AvgRTT: 50 * time.Millisecond}},
		},
	}
}

// newTestLogger returns a logger that discards output, suitable for unit tests.
func newTRTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── constructor ──────────────────────────────────────────────────────────────

func TestNewTracerouteRunner(t *testing.T) {
	prober := &stubTracerouteProber{}
	logger := newTRTestLogger()
	runner := NewTracerouteRunner(prober, logger)
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if runner.Prober != prober {
		t.Fatal("expected Prober to be assigned")
	}
}

// ── nil prober guard ─────────────────────────────────────────────────────────

func TestTracerouteRunner_NilProber(t *testing.T) {
	runner := &TracerouteRunner{Logger: newTRTestLogger()}
	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
	}
	err := runner.Run(context.Background(), req)
	if !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("expected ErrRunnerNotFound, got %v", err)
	}
}

// ── basic execution ──────────────────────────────────────────────────────────

func TestTracerouteRunner_ExecutesTrace(t *testing.T) {
	prober := &stubTracerouteProber{result: build3HopResult()}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 3, Timeout: time.Second},
			Net:    NetworkOptions{Host: "example.com", MaxHops: 10},
		},
		Report: &DiagReport{Target: TargetWeb, Host: "example.com"},
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prober.calledHost != "example.com" {
		t.Fatalf("expected host 'example.com', got %q", prober.calledHost)
	}
	if prober.calledMaxHops != 10 {
		t.Fatalf("expected maxHops=10, got %d", prober.calledMaxHops)
	}
	if prober.calledAttempts != 3 {
		t.Fatalf("expected attemptsPerHop=3, got %d", prober.calledAttempts)
	}
}

// ── default fallbacks ─────────────────────────────────────────────────────────

func TestTracerouteRunner_DefaultHost(t *testing.T) {
	prober := &stubTracerouteProber{}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
		// Net.Host intentionally empty
	}
	_ = runner.Run(context.Background(), req)

	if prober.calledHost != "example.com" {
		t.Fatalf("expected default host 'example.com', got %q", prober.calledHost)
	}
}

func TestTracerouteRunner_DefaultMaxHops(t *testing.T) {
	prober := &stubTracerouteProber{}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
		// Net.MaxHops = 0 → should use DefaultMaxHops
	}
	_ = runner.Run(context.Background(), req)

	if prober.calledMaxHops != DefaultMaxHops {
		t.Fatalf("expected maxHops=%d, got %d", DefaultMaxHops, prober.calledMaxHops)
	}
}

func TestTracerouteRunner_DefaultMTRCount(t *testing.T) {
	prober := &stubTracerouteProber{}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 0, Timeout: time.Second}},
		// MTRCount = 0 → should use DefaultMTRCount
	}
	_ = runner.Run(context.Background(), req)

	if prober.calledAttempts != DefaultMTRCount {
		t.Fatalf("expected attemptsPerHop=%d, got %d", DefaultMTRCount, prober.calledAttempts)
	}
}

// ── report integration ────────────────────────────────────────────────────────

func TestTracerouteRunner_SetsRoute(t *testing.T) {
	route := build3HopResult()
	prober := &stubTracerouteProber{result: route}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	report := &DiagReport{Target: TargetWeb, Host: "example.com"}
	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 3, Timeout: time.Second}},
		Report:  report,
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Route == nil {
		t.Fatal("expected Route to be set on DiagReport")
	}
	if len(report.Route.Hops) != 3 {
		t.Fatalf("expected 3 hops in report, got %d", len(report.Route.Hops))
	}
}

// TestTracerouteRunner_NilReport verifies that a nil Report does not cause a panic.
func TestTracerouteRunner_NilReport(t *testing.T) {
	prober := &stubTracerouteProber{result: build3HopResult()}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
		Report:  nil, // explicitly nil
	}

	// Must not panic.
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error with nil report: %v", err)
	}
}

// ── progress emission ─────────────────────────────────────────────────────────

func TestTracerouteRunner_EmitsProgress(t *testing.T) {
	prober := &stubTracerouteProber{result: build3HopResult()}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	var events []ProgressEvent
	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
		Hook: func(e ProgressEvent) {
			events = append(events, e)
		},
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect 1 header emit + 3 hop emits = 4 total.
	if len(events) != 4 {
		t.Fatalf("expected 4 progress events (header + 3 hops), got %d", len(events))
	}

	// First event must be the header with stage "traceroute".
	if events[0].Stage != "traceroute" {
		t.Errorf("event 0: expected stage 'traceroute', got %q", events[0].Stage)
	}

	// Remaining 3 events must have stage "traceroute-hop" with non-nil Hop.
	for i, e := range events[1:] {
		if e.Stage != "traceroute-hop" {
			t.Errorf("event %d: expected stage 'traceroute-hop', got %q", i+1, e.Stage)
		}
		if e.Hop == nil {
			t.Errorf("event %d: expected non-nil Hop field for traceroute-hop event", i+1)
		}
	}
}

// TestTracerouteRunner_TimedOutHopEmitsMark verifies that a timed-out hop emits
// a "traceroute-hop" event with an empty IP field.
func TestTracerouteRunner_TimedOutHopEmitsMark(t *testing.T) {
	prober := &stubTracerouteProber{result: build3HopResult()}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	var events []ProgressEvent
	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
		Hook: func(e ProgressEvent) {
			events = append(events, e)
		},
	}
	_ = runner.Run(context.Background(), req)

	// The second hop (TTL=2) is timed-out.  Its traceroute-hop event must have
	// an empty IP and the message must still contain "???".
	found := false
	for _, ev := range events {
		if ev.Stage == "traceroute-hop" && ev.Hop != nil && ev.Hop.IP == "" {
			if !containsStr(ev.Message, "???") {
				t.Errorf("timed-out hop message should contain '???', got: %q", ev.Message)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a traceroute-hop event with empty IP (timed-out hop), got events: %+v", events)
	}
}

// ── error propagation ─────────────────────────────────────────────────────────

func TestTracerouteRunner_ProberError(t *testing.T) {
	want := errors.New("raw socket unavailable")
	prober := &stubTracerouteProber{err: want}
	runner := NewTracerouteRunner(prober, newTRTestLogger())

	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
	}
	err := runner.Run(context.Background(), req)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// ── MultiRunner composition ───────────────────────────────────────────────────

func TestTracerouteRunner_InMultiRunner(t *testing.T) {
	prober := &stubTracerouteProber{result: build3HopResult()}
	trRunner := NewTracerouteRunner(prober, newTRTestLogger())

	multi := NewMultiRunner(trRunner)
	report := &DiagReport{Target: TargetWeb, Host: "traceroute.test"}
	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 2, Timeout: time.Second}},
		Report:  report,
	}

	if err := multi.Run(context.Background(), req); err != nil {
		t.Fatalf("MultiRunner.Run: %v", err)
	}
	if report.Route == nil {
		t.Fatal("expected Route set via MultiRunner")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// containsStr reports whether s contains substr.
func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
