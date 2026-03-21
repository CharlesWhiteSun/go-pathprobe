package diag

// Progress-hook unit tests live in the internal test package (package diag) so
// they can access unexported runner constructors and stub prober types that are
// already defined in connectivity_runner_test.go.

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

// ── ProgressEvent / Request.Emit ─────────────────────────────────────────

// TestEmit_NilHookDoesNotPanic verifies that calling Emit or Emitf on a Request
// with a nil Hook is completely safe.
func TestEmit_NilHookDoesNotPanic(t *testing.T) {
	req := Request{} // Hook is nil
	req.Emit("stage", "message")
	req.Emitf("stage", "format %s", "arg")
}

// TestEmit_CallsHookWithCorrectEvent verifies that a non-nil Hook receives
// exactly the event that was emitted.
func TestEmit_CallsHookWithCorrectEvent(t *testing.T) {
	var got []ProgressEvent
	req := Request{
		Hook: func(ev ProgressEvent) { got = append(got, ev) },
	}

	req.Emit("test_stage", "hello world")

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Stage != "test_stage" {
		t.Errorf("Stage = %q, want %q", got[0].Stage, "test_stage")
	}
	if got[0].Message != "hello world" {
		t.Errorf("Message = %q, want %q", got[0].Message, "hello world")
	}
}

// TestEmitf_FormatsMessage verifies that Emitf formats the message via fmt.
func TestEmitf_FormatsMessage(t *testing.T) {
	var got []ProgressEvent
	req := Request{
		Hook: func(ev ProgressEvent) { got = append(got, ev) },
	}

	req.Emitf("net", "Probing %d port(s) on %s", 3, "example.com")

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	want := "Probing 3 port(s) on example.com"
	if got[0].Message != want {
		t.Errorf("Message = %q, want %q", got[0].Message, want)
	}
}

// TestEmit_MultipleCallsPreservesOrder verifies all events are received in
// emission order.
func TestEmit_MultipleCallsPreservesOrder(t *testing.T) {
	stages := []string{"a", "b", "c"}
	var got []string
	req := Request{
		Hook: func(ev ProgressEvent) { got = append(got, ev.Stage) },
	}

	for _, s := range stages {
		req.Emit(s, "msg")
	}

	if len(got) != len(stages) {
		t.Fatalf("expected %d events, got %d", len(stages), len(got))
	}
	for i, s := range stages {
		if got[i] != s {
			t.Errorf("event[%d].Stage = %q, want %q", i, got[i], s)
		}
	}
}

// ── ConnectivityRunner emits progress ─────────────────────────────────────

// TestConnectivityRunner_EmitsNetworkStage verifies that ConnectivityRunner
// emits at least one "network" progress event when a hook is set.
func TestConnectivityRunner_EmitsNetworkStage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prober := &stubPortProber{}
	runner := NewConnectivityRunner(prober, logger)

	var stages []string
	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Net:    NetworkOptions{Host: "example.com", Ports: []int{443}},
		},
		Report: &DiagReport{},
		Hook:   func(ev ProgressEvent) { stages = append(stages, ev.Stage) },
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(stages) == 0 {
		t.Fatal("expected at least one progress event, got none")
	}
	found := false
	for _, s := range stages {
		if s == "network" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no 'network' stage emitted; got: %v", stages)
	}
}

// TestConnectivityRunner_EmitsPortResultStage verifies that a "port_result"
// event is emitted for each probed port.
func TestConnectivityRunner_EmitsPortResultStage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prober := &stubPortProber{}
	runner := NewConnectivityRunner(prober, logger)

	var stages []string
	req := Request{
		Target: TargetSMTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Net:    NetworkOptions{Host: "mail.example.com", Ports: []int{25, 587}},
		},
		Report: &DiagReport{},
		Hook:   func(ev ProgressEvent) { stages = append(stages, ev.Stage) },
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	portResultCount := 0
	for _, s := range stages {
		if s == "port_result" {
			portResultCount++
		}
	}
	// Two ports probed → two port_result events.
	if portResultCount != 2 {
		t.Errorf("port_result events = %d, want 2; all stages: %v", portResultCount, stages)
	}
}

// TestConnectivityRunner_NoHookStillWorks verifies the runner works normally
// when no hook is provided (backward-compatibility).
func TestConnectivityRunner_NoHookStillWorks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prober := &stubPortProber{}
	runner := NewConnectivityRunner(prober, logger)

	req := Request{
		Target: TargetSFTP,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Net:    NetworkOptions{Host: "sftp.example.com", Ports: []int{22}},
		},
		Report: &DiagReport{},
		// Hook intentionally omitted.
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("Run without hook failed: %v", err)
	}
}

// ── HopProgressData fields ───────────────────────────────────────────────

// TestHopProgressData_ZeroValue verifies that a zero-value HopProgressData
// has empty/zero fields and does not panic when accessed.
func TestHopProgressData_ZeroValue(t *testing.T) {
	var h HopProgressData
	if h.TTL != 0 {
		t.Errorf("zero TTL: want 0, got %d", h.TTL)
	}
	if h.IP != "" {
		t.Errorf("zero IP: want empty string, got %q", h.IP)
	}
	if h.Hostname != "" {
		t.Errorf("zero Hostname: want empty string, got %q", h.Hostname)
	}
	if h.LossPct != 0 {
		t.Errorf("zero LossPct: want 0, got %f", h.LossPct)
	}
}

// TestHopProgressData_RoundTrip verifies that HopProgressData fields survive
// being embedded in a ProgressEvent and retrieved again.
func TestHopProgressData_RoundTrip(t *testing.T) {
	hop := &HopProgressData{
		TTL:      7,
		MaxHops:  30,
		IP:       "10.0.0.1",
		Hostname: "router.example.com",
		AvgRTT:   "5.20ms",
		LossPct:  0.0,
		Sent:     5,
		Received: 5,
	}
	ev := ProgressEvent{Stage: "traceroute-hop", Message: "hop info", Hop: hop}

	if ev.Hop == nil {
		t.Fatal("expected non-nil Hop after assignment")
	}
	if ev.Hop.TTL != 7 {
		t.Errorf("TTL = %d, want 7", ev.Hop.TTL)
	}
	if ev.Hop.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want 10.0.0.1", ev.Hop.IP)
	}
}

// ── EmitEvent ─────────────────────────────────────────────────────────────

// TestEmitEvent_WithHopData verifies that EmitEvent passes a ProgressEvent
// with a non-nil Hop field through the hook unchanged.
func TestEmitEvent_WithHopData(t *testing.T) {
	var got []ProgressEvent
	req := Request{
		Hook: func(ev ProgressEvent) { got = append(got, ev) },
	}

	hop := &HopProgressData{TTL: 3, IP: "192.168.0.1", LossPct: 50.0}
	req.EmitEvent(ProgressEvent{Stage: "traceroute-hop", Message: "hop 3", Hop: hop})

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Hop == nil {
		t.Fatal("expected non-nil Hop in received event")
	}
	if got[0].Hop.TTL != 3 {
		t.Errorf("Hop.TTL = %d, want 3", got[0].Hop.TTL)
	}
	if got[0].Hop.LossPct != 50.0 {
		t.Errorf("Hop.LossPct = %f, want 50.0", got[0].Hop.LossPct)
	}
}

// TestEmitEvent_NilHookDoesNotPanic verifies EmitEvent is no-op with nil hook.
func TestEmitEvent_NilHookDoesNotPanic(t *testing.T) {
	req := Request{} // Hook is nil
	hop := &HopProgressData{TTL: 1}
	req.EmitEvent(ProgressEvent{Stage: "traceroute-hop", Hop: hop})
}
