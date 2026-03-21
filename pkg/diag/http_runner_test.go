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

// tlsStubHTTPProber is a stub that returns a configurable TLS state alongside
// a fixed HTTP 200 response, allowing summary and TLS rendering to be tested
// without a real network connection.
type tlsStubHTTPProber struct {
	tls *netprobe.TLSInfo
}

func (s *tlsStubHTTPProber) Probe(_ context.Context, _ netprobe.HTTPProbeRequest) (netprobe.HTTPProbeResult, error) {
	return netprobe.HTTPProbeResult{StatusCode: 200, RTT: 50 * time.Millisecond, TLS: s.tls}, nil
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
	if got := report.Protos[0].Port; got != 443 {
		t.Errorf("ProtoResult.Port = %d; want 443 (https default port)", got)
	}
}

// TestHTTPRunnerProtoResultPort validates that ProtoResult.Port is populated
// correctly for every combination of scheme and explicit port specification:
//   - https with no port → 443 (well-known default)
//   - http  with no port → 80  (well-known default)
//   - https with explicit :8443 → 8443 (explicit wins over default)
//   - http  with explicit :8080 → 8080 (explicit wins over default)
func TestHTTPRunnerProtoResultPort(t *testing.T) {
	cases := []struct {
		url      string
		wantPort int
	}{
		{"https://example.com/path", 443},
		{"http://example.com/path", 80},
		{"https://example.com:8443/path", 8443},
		{"http://example.com:8080/path", 8080},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.url, func(t *testing.T) {
			prober := &stubHTTPProber{}
			runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
			report := &DiagReport{}
			req := Request{
				Target: TargetWeb,
				Options: Options{
					Global: GlobalOptions{Timeout: time.Second},
					Web:    WebOptions{Mode: WebModeHTTP, URL: tc.url},
				},
				Report: report,
			}
			if err := runner.Run(context.Background(), req); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(report.Protos) == 0 {
				t.Fatal("expected a ProtoResult to be recorded")
			}
			if got := report.Protos[0].Port; got != tc.wantPort {
				t.Errorf("url=%q: ProtoResult.Port = %d; want %d", tc.url, got, tc.wantPort)
			}
		})
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

// TestHTTPRunnerSchemeCaseNormalisation verifies that an uppercase scheme prefix
// (e.g. "HTTPS://") is lowercased before the request is sent to the prober,
// so log output, ProtoResult.Protocol, and the probed URL are all consistent
// regardless of how the user typed the scheme.
func TestHTTPRunnerSchemeCaseNormalisation(t *testing.T) {
	cases := []struct {
		input   string
		wantURL string
	}{
		{"HTTPS://google.com", "https://google.com"},
		{"HTTP://google.com", "http://google.com"},
		{"HTTPS://google.com/Path?q=Search", "https://google.com/Path?q=Search"},
		// Mixed case — only the scheme prefix is lowercased, path is preserved.
		{"Https://google.com", "https://google.com"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			prober := &stubHTTPProber{}
			runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
			req := Request{
				Target:  TargetWeb,
				Options: Options{Global: GlobalOptions{Timeout: time.Second}, Web: WebOptions{URL: tc.input}},
			}
			if err := runner.Run(context.Background(), req); err != nil {
				t.Fatalf("input=%q: unexpected error: %v", tc.input, err)
			}
			if prober.url != tc.wantURL {
				t.Errorf("input=%q: probed URL = %q; want %q", tc.input, prober.url, tc.wantURL)
			}
		})
	}
}

// TestHTTPRunnerSummaryFormat verifies that ProtoResult.Summary reflects the
// actual scheme (HTTPS / HTTP), includes TLS version and ALPN when available,
// and always ends with the RTT.  This prevents the regression where the summary
// was always "HTTP 200, RTT ..." regardless of the scheme or TLS details.
func TestHTTPRunnerSummaryFormat(t *testing.T) {
	cases := []struct {
		name        string
		url         string
		tls         *netprobe.TLSInfo
		wantParts   []string
		absentParts []string
	}{
		{
			name:      "https with TLS1.3 and h2",
			url:       "https://example.com",
			tls:       &netprobe.TLSInfo{Version: "TLS1.3", NegotiatedALPN: "h2"},
			wantParts: []string{"HTTPS 200", "TLS1.3", "h2", "RTT"},
		},
		{
			name:      "https with TLS1.2 no ALPN",
			url:       "https://example.com",
			tls:       &netprobe.TLSInfo{Version: "TLS1.2"},
			wantParts: []string{"HTTPS 200", "TLS1.2", "RTT"},
		},
		{
			name:        "http plain no TLS",
			url:         "http://example.com",
			tls:         nil,
			wantParts:   []string{"HTTP 200", "RTT"},
			absentParts: []string{"TLS"},
		},
		{
			name:      "uppercase scheme normalised to https in summary",
			url:       "HTTPS://example.com",
			tls:       &netprobe.TLSInfo{Version: "TLS1.3", NegotiatedALPN: "h2"},
			wantParts: []string{"HTTPS 200", "TLS1.3", "h2", "RTT"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			prober := &tlsStubHTTPProber{tls: tc.tls}
			runner := NewHTTPRunner(prober, slog.New(slog.NewTextHandler(io.Discard, nil)))
			report := &DiagReport{}
			req := Request{
				Target: TargetWeb,
				Options: Options{
					Global: GlobalOptions{Timeout: time.Second},
					Web:    WebOptions{Mode: WebModeHTTP, URL: tc.url},
				},
				Report: report,
			}
			if err := runner.Run(context.Background(), req); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(report.Protos) == 0 {
				t.Fatal("expected a ProtoResult to be recorded")
			}
			summary := report.Protos[0].Summary
			for _, part := range tc.wantParts {
				if !contains(summary, part) {
					t.Errorf("summary = %q; expected to contain %q", summary, part)
				}
			}
			for _, part := range tc.absentParts {
				if contains(summary, part) {
					t.Errorf("summary = %q; must NOT contain %q", summary, part)
				}
			}
		})
	}
}
