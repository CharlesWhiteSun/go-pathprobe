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

// TestHTTPRunnerSkipsWhenURLEmpty verifies that the runner emits a skip message
// and does not call the prober when no URL is provided.  This reflects the new
// behaviour after removing the net.Host fallback — the HTTP mode now owns its
// own URL input and does not inherit the target-host field.
func TestHTTPRunnerSkipsWhenURLEmpty(t *testing.T) {
	prober := &stubHTTPProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewHTTPRunner(prober, logger)

	var events []string
	req := Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{Timeout: time.Second, MTRCount: 1}},
		Hook:    func(e ProgressEvent) { events = append(events, e.Stage) },
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if prober.called {
		t.Fatal("prober must not be called when URL is empty")
	}
	found := false
	for _, e := range events {
		if e == "http" {
			found = true
		}
	}
	if !found {
		t.Error("expected an 'http' skip event to be emitted when URL is empty")
	}
}

// TestHTTPRunnerProtoResultHostFromURL verifies that ProtoResult.Host is parsed
// from the probed URL hostname rather than net.Host (which is now empty when
// the host input is hidden in http mode).
func TestHTTPRunnerProtoResultHostFromURL(t *testing.T) {
	prober := &stubHTTPProber{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewHTTPRunner(prober, logger)

	report := &DiagReport{}
	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Web:    WebOptions{Mode: WebModeHTTP, URL: "https://check.example.com/path"},
			Net:    NetworkOptions{Host: ""},
		},
		Report: report,
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Protos) == 0 {
		t.Fatal("expected a ProtoResult to be recorded")
	}
	if got := report.Protos[0].Host; got != "check.example.com" {
		t.Errorf("ProtoResult.Host = %q; want %q", got, "check.example.com")
	}
	if got := report.Protos[0].Protocol; got != "https" {
		t.Errorf("ProtoResult.Protocol = %q; want \"https\" (scheme must reflect URL, not be hardcoded 'http')", got)
	}
}

// TestHTTPRunnerSchemeNormalisation verifies that bare hostnames (no https:// or
// http://) are automatically promoted to https:// before probing, so the user
// can type "www.google.com" without getting "unsupported protocol scheme" errors.
// A progress event must also be emitted to inform the user of the auto-correction.
func TestHTTPRunnerSchemeNormalisation(t *testing.T) {
	cases := []struct {
		input   string
		wantURL string
	}{
		{"google.com", "https://google.com"},
		{"www.google.com", "https://www.google.com"},
		{"google.com/path?q=1", "https://google.com/path?q=1"},
		// Already correct — must pass through unchanged.
		{"https://google.com", "https://google.com"},
		{"http://google.com", "http://google.com"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			prober := &stubHTTPProber{}
			runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))

			var events []string
			req := Request{
				Target:  TargetWeb,
				Options: Options{Global: GlobalOptions{Timeout: time.Second}, Web: WebOptions{URL: tc.input}},
				Hook:    func(e ProgressEvent) { events = append(events, e.Message) },
			}
			if err := runner.Run(context.Background(), req); err != nil {
				t.Fatalf("input=%q: unexpected error: %v", tc.input, err)
			}
			if prober.url != tc.wantURL {
				t.Errorf("input=%q: probed URL = %q; want %q", tc.input, prober.url, tc.wantURL)
			}
			// When scheme was auto-added, a "assuming HTTPS" event must have been emitted.
			if tc.wantURL != tc.input {
				found := false
				for _, msg := range events {
					if len(msg) > 0 && contains(msg, "assuming HTTPS") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("input=%q: expected 'assuming HTTPS' emit event, got %v", tc.input, events)
				}
			}
		})
	}
}

// contains is a trivial helper to avoid importing strings in test file.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
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

// TestHTTPRunnerSkippedOnNonHTTPModes verifies the runner is a no-op when the
// explicit web mode is not "http".
func TestHTTPRunnerSkippedOnNonHTTPModes(t *testing.T) {
	for _, mode := range []WebMode{WebModePublicIP, WebModeDNS, WebModePort} {
		prober := &stubHTTPProber{}
		runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
		req := Request{Target: TargetWeb, Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Web:    WebOptions{Mode: mode, URL: "https://example.com"},
		}}
		if err := runner.Run(context.Background(), req); err != nil {
			t.Fatalf("mode=%q: unexpected error %v", mode, err)
		}
		if prober.called {
			t.Fatalf("mode=%q: prober must not be called when mode is not http", mode)
		}
	}
}

// TestHTTPRunnerCalledOnHTTPMode verifies the runner fires for WebModeHTTP.
func TestHTTPRunnerCalledOnHTTPMode(t *testing.T) {
	prober := &stubHTTPProber{}
	runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := Request{Target: TargetWeb, Options: Options{
		Global: GlobalOptions{Timeout: time.Second},
		Web:    WebOptions{Mode: WebModeHTTP, URL: "https://example.com"},
		Net:    NetworkOptions{Host: "example.com"},
	}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !prober.called {
		t.Fatal("prober must fire in http mode")
	}
}
