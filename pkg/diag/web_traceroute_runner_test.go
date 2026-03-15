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

// ── helpers ────────────────────────────────────────────────────────────────

// webTracerouteRequest builds a Request with the given WebMode and optional MaxHops.
func webTracerouteRequest(mode WebMode, maxHops int) Request {
	return Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Mode: mode, MaxHops: maxHops},
		},
	}
}

// ── mode filtering ─────────────────────────────────────────────────────────

// TestWebTracerouteRunner_DelegatesOnTracerouteMode verifies delegation when
// the mode is explicitly "traceroute".
func TestWebTracerouteRunner_DelegatesOnTracerouteMode(t *testing.T) {
	inner := &trackingRunner{}
	r := NewWebTracerouteRunner(inner)
	if err := r.Run(context.Background(), webTracerouteRequest(WebModeTraceroute, 0)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.called {
		t.Fatal("inner runner must be called in traceroute mode")
	}
}

// TestWebTracerouteRunner_DelegatesOnLegacyMode verifies delegation when mode
// is "" (all-in-one legacy behaviour).
func TestWebTracerouteRunner_DelegatesOnLegacyMode(t *testing.T) {
	inner := &trackingRunner{}
	r := NewWebTracerouteRunner(inner)
	if err := r.Run(context.Background(), webTracerouteRequest(WebModeAll, 0)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.called {
		t.Fatal("inner runner must be called in legacy (all) mode")
	}
}

// TestWebTracerouteRunner_NoOpOnOtherModes verifies the runner is silent for
// public-ip / dns / http / port modes.
func TestWebTracerouteRunner_NoOpOnOtherModes(t *testing.T) {
	noOpModes := []WebMode{WebModePublicIP, WebModeDNS, WebModeHTTP, WebModePort}
	for _, mode := range noOpModes {
		inner := &trackingRunner{}
		r := NewWebTracerouteRunner(inner)
		if err := r.Run(context.Background(), webTracerouteRequest(mode, 0)); err != nil {
			t.Fatalf("mode=%q: unexpected error %v", mode, err)
		}
		if inner.called {
			t.Fatalf("mode=%q: inner runner must NOT be called", mode)
		}
	}
}

// ── MaxHops bridging ──────────────────────────────────────────────────────

// capturingRunner stores the Request it received, allowing assertions on
// the forwarded options.
type capturingRunner struct {
	capturedReq Request
}

func (r *capturingRunner) Run(_ context.Context, req Request) error {
	r.capturedReq = req
	return nil
}

// TestWebTracerouteRunner_BridgesMaxHopsWhenNetNotSet verifies that
// WebOptions.MaxHops is copied into NetworkOptions.MaxHops when the network
// field is zero.
func TestWebTracerouteRunner_BridgesMaxHopsWhenNetNotSet(t *testing.T) {
	inner := &capturingRunner{}
	r := NewWebTracerouteRunner(inner)

	req := webTracerouteRequest(WebModeTraceroute, 20)
	if err := r.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.capturedReq.Options.Net.MaxHops != 20 {
		t.Fatalf("expected Net.MaxHops=20, got %d", inner.capturedReq.Options.Net.MaxHops)
	}
}

// TestWebTracerouteRunner_DoesNotOverrideExistingNetMaxHops verifies that an
// explicitly set NetworkOptions.MaxHops is preserved and not overwritten by
// the web-level value.
func TestWebTracerouteRunner_DoesNotOverrideExistingNetMaxHops(t *testing.T) {
	inner := &capturingRunner{}
	r := NewWebTracerouteRunner(inner)

	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Mode: WebModeTraceroute, MaxHops: 20},
			Net:    NetworkOptions{MaxHops: 15}, // already set — must win
		},
	}
	if err := r.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.capturedReq.Options.Net.MaxHops != 15 {
		t.Fatalf("expected Net.MaxHops=15 (net wins), got %d", inner.capturedReq.Options.Net.MaxHops)
	}
}

// TestWebTracerouteRunner_ZeroWebMaxHopsNotBridged verifies that when
// WebOptions.MaxHops is zero (not set) the net-level value is left unchanged.
func TestWebTracerouteRunner_ZeroWebMaxHopsNotBridged(t *testing.T) {
	inner := &capturingRunner{}
	r := NewWebTracerouteRunner(inner)

	req := webTracerouteRequest(WebModeTraceroute, 0) // Web.MaxHops = 0
	if err := r.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Net.MaxHops should remain 0; TracerouteRunner applies DefaultMaxHops internally.
	if inner.capturedReq.Options.Net.MaxHops != 0 {
		t.Fatalf("expected Net.MaxHops=0 when web value is zero, got %d", inner.capturedReq.Options.Net.MaxHops)
	}
}

// ── full integration via TracerouteRunner ────────────────────────────────

// stubTracerouteProberForWeb is a lightweight stub that records invocation
// metadata and returns a fixed 2-hop RouteResult.
type stubTracerouteProberForWeb struct {
	calledMaxHops int
	called        bool
}

func (s *stubTracerouteProberForWeb) Trace(_ context.Context, _ string, maxHops, _ int) (netprobe.RouteResult, error) {
	s.called = true
	s.calledMaxHops = maxHops
	return netprobe.RouteResult{
		Hops: []netprobe.HopResult{
			{TTL: 1, IP: "192.168.1.1", Stats: netprobe.ProbeStats{Sent: 1, Received: 1}},
			{TTL: 2, IP: "93.184.216.34", Stats: netprobe.ProbeStats{Sent: 1, Received: 1}},
		},
	}, nil
}

// TestWebTracerouteRunner_SetsRouteOnReport verifies the full chain:
// WebTracerouteRunner → TracerouteRunner → DiagReport.SetRoute.
func TestWebTracerouteRunner_SetsRouteOnReport(t *testing.T) {
	prober := &stubTracerouteProberForWeb{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inner := NewTracerouteRunner(prober, logger)
	wrapper := NewWebTracerouteRunner(inner)

	report := &DiagReport{Target: TargetWeb, Host: "example.com"}
	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Mode: WebModeTraceroute},
			Net:    NetworkOptions{Host: "example.com"},
		},
		Report: report,
	}

	if err := wrapper.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Route == nil {
		t.Fatal("expected DiagReport.Route to be set after WebTracerouteRunner.Run")
	}
	if len(report.Route.Hops) != 2 {
		t.Fatalf("expected 2 hops in report, got %d", len(report.Route.Hops))
	}
}

// TestWebTracerouteRunner_EmitCountEqualsHopCount verifies that the number of
// hop-level "traceroute" Emit events equals the number of hops returned.
// (1 header emit + N hop emits total = N+1, but we assert on the hop emits only.)
func TestWebTracerouteRunner_EmitCountEqualsHopCount(t *testing.T) {
	prober := &stubTracerouteProberForWeb{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inner := NewTracerouteRunner(prober, logger)
	wrapper := NewWebTracerouteRunner(inner)

	var hopEmits int
	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Mode: WebModeTraceroute},
			Net:    NetworkOptions{Host: "example.com"},
		},
		Hook: func(e ProgressEvent) {
			if e.Stage == "traceroute" {
				hopEmits++
			}
		},
	}

	if err := wrapper.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 header emit + 2 hop emits = 3 total.
	const wantTotal = 3
	if hopEmits != wantTotal {
		t.Fatalf("expected %d traceroute emits (header+hops), got %d", wantTotal, hopEmits)
	}
}

// TestWebTracerouteRunner_NoOpDoesNotEmit verifies that no progress events are
// emitted when the mode does not match traceroute.
func TestWebTracerouteRunner_NoOpDoesNotEmit(t *testing.T) {
	prober := &stubTracerouteProberForWeb{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inner := NewTracerouteRunner(prober, logger)
	wrapper := NewWebTracerouteRunner(inner)

	var emits int
	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Mode: WebModePort}, // not traceroute
		},
		Hook: func(_ ProgressEvent) { emits++ },
	}

	if err := wrapper.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emits != 0 {
		t.Fatalf("expected 0 emits in no-op mode, got %d", emits)
	}
}

// TestWebTracerouteRunner_PropagatesProberError verifies that an error from the
// inner TracerouteRunner surfaces through the wrapper.
func TestWebTracerouteRunner_PropagatesProberError(t *testing.T) {
	want := errors.New("probe failed")
	errRunner := &errorRunner{err: want}
	wrapper := NewWebTracerouteRunner(errRunner)

	req := webTracerouteRequest(WebModeTraceroute, 0)
	err := wrapper.Run(context.Background(), req)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// errorRunner always returns a configurable error.
type errorRunner struct{ err error }

func (r *errorRunner) Run(_ context.Context, _ Request) error { return r.err }

// TestWebTracerouteRunner_MaxHopsBridgedToInner verifies that the bridged
// MaxHops value reaches the prober via TracerouteRunner.
func TestWebTracerouteRunner_MaxHopsBridgedToInner(t *testing.T) {
	prober := &stubTracerouteProberForWeb{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inner := NewTracerouteRunner(prober, logger)
	wrapper := NewWebTracerouteRunner(inner)

	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Mode: WebModeTraceroute, MaxHops: 12},
			Net:    NetworkOptions{Host: "example.com"},
			// Net.MaxHops intentionally 0 so the bridge fires.
		},
	}

	if err := wrapper.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prober.calledMaxHops != 12 {
		t.Fatalf("expected prober called with maxHops=12, got %d", prober.calledMaxHops)
	}
}
