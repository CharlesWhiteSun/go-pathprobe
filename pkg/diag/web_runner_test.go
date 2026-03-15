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

type stubFetcher struct {
	called bool
}

func (s *stubFetcher) Fetch(ctx context.Context) (netprobe.PublicIPResult, error) {
	s.called = true
	return netprobe.PublicIPResult{IP: "203.0.113.10", Source: "stub", RTT: time.Millisecond}, nil
}

type countingResolver struct {
	calls int
}

func (c *countingResolver) Lookup(ctx context.Context, name string, rtype netprobe.RecordType) (netprobe.DNSAnswer, error) {
	c.calls++
	return netprobe.DNSAnswer{Name: name, Type: rtype, Values: []string{"192.0.2.1"}, Source: "counting"}, nil
}

// TestWebRunnerUsesFetcherAndComparator verifies web runner orchestrates public IP fetch and DNS compare.
func TestWebRunnerUsesFetcherAndComparator(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Domains: []string{"example.net"}, Types: []netprobe.RecordType{netprobe.RecordTypeA}},
		},
	}

	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !fetcher.called {
		t.Fatalf("expected fetcher to be called")
	}
	if resolver.calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolver.calls)
	}
}

// TestWebRunnerDefaultsApplied ensures default domains/types kick in when not provided by user input.
func TestWebRunnerDefaultsApplied(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	req := Request{Target: TargetWeb, Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}}}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// defaultTypes has length 3; with single resolver expect 3 calls.
	if resolver.calls != 3 {
		t.Fatalf("expected resolver called 3 times, got %d", resolver.calls)
	}
}

// TestWebRunnerFetcherError verifies that a failing fetcher propagates its error.
func TestWebRunnerFetcherError(t *testing.T) {
	runner := NewWebRunner(&errorFetcher{}, netprobe.DNSComparator{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := Request{Target: TargetWeb, Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}}}
	if err := runner.Run(context.Background(), req); err == nil {
		t.Fatal("expected error from failing fetcher")
	}
}

// TestWebRunnerComparatorError verifies that a failing resolver propagates its error.
func TestWebRunnerComparatorError(t *testing.T) {
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{&errorWebResolver{}}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(&stubFetcher{}, comparator, logger)

	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Domains: []string{"example.com"}, Types: []netprobe.RecordType{netprobe.RecordTypeA}},
		},
	}
	if err := runner.Run(context.Background(), req); err == nil {
		t.Fatal("expected error from failing comparator resolver")
	}
}

// TestWebRunnerDivergentDNS verifies the runner completes without error even when resolvers diverge.
func TestWebRunnerDivergentDNS(t *testing.T) {
	r1 := &fixedResolver{name: "r1", values: []string{"1.1.1.1"}}
	r2 := &fixedResolver{name: "r2", values: []string{"2.2.2.2"}}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{r1, r2}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(&stubFetcher{}, comparator, logger)

	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web:    WebOptions{Domains: []string{"example.com"}, Types: []netprobe.RecordType{netprobe.RecordTypeA}},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("divergent DNS should not return an error, got: %v", err)
	}
}

type errorFetcher struct{}

func (e *errorFetcher) Fetch(_ context.Context) (netprobe.PublicIPResult, error) {
	return netprobe.PublicIPResult{}, errors.New("fetch failed")
}

type errorWebResolver struct{}

func (e *errorWebResolver) Lookup(_ context.Context, _ string, _ netprobe.RecordType) (netprobe.DNSAnswer, error) {
	return netprobe.DNSAnswer{}, errors.New("resolver failed")
}

type fixedResolver struct {
	name   string
	values []string
}

func (f *fixedResolver) Lookup(_ context.Context, name string, rtype netprobe.RecordType) (netprobe.DNSAnswer, error) {
	return netprobe.DNSAnswer{Name: name, Type: rtype, Values: f.values, Source: f.name}, nil
}

// ── WebMode tests ─────────────────────────────────────────────────────────

func makeWebRequest(mode WebMode) Request {
	return Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 1, Timeout: time.Second},
			Web: WebOptions{
				Mode:    mode,
				Domains: []string{"example.net"},
				Types:   []netprobe.RecordType{netprobe.RecordTypeA},
			},
		},
	}
}

// TestWebRunnerPublicIPMode: only the fetcher should fire; resolver must stay idle.
func TestWebRunnerPublicIPMode(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	if err := runner.Run(context.Background(), makeWebRequest(WebModePublicIP)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fetcher.called {
		t.Fatal("expected fetcher to be called in public-ip mode")
	}
	if resolver.calls != 0 {
		t.Fatalf("expected resolver silent in public-ip mode, got %d calls", resolver.calls)
	}
}

// TestWebRunnerDNSMode: only the resolver should fire; fetcher must stay idle.
func TestWebRunnerDNSMode(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	if err := runner.Run(context.Background(), makeWebRequest(WebModeDNS)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetcher.called {
		t.Fatal("fetcher must not be called in dns mode")
	}
	if resolver.calls == 0 {
		t.Fatal("expected resolver called in dns mode")
	}
}

// TestWebRunnerHTTPMode: this runner should be a no-op (HTTP and port handled elsewhere).
func TestWebRunnerHTTPMode(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	if err := runner.Run(context.Background(), makeWebRequest(WebModeHTTP)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetcher.called || resolver.calls != 0 {
		t.Fatal("WebRunner must be a no-op in http mode")
	}
}

// TestWebRunnerPortMode: this runner should be a no-op.
func TestWebRunnerPortMode(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	if err := runner.Run(context.Background(), makeWebRequest(WebModePort)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetcher.called || resolver.calls != 0 {
		t.Fatal("WebRunner must be a no-op in port mode")
	}
}

// TestIsValidWebMode verifies the validation helper.
func TestIsValidWebMode(t *testing.T) {
	valid := []WebMode{WebModeAll, WebModePublicIP, WebModeDNS, WebModeHTTP, WebModePort, WebModeTraceroute}
	for _, m := range valid {
		if !IsValidWebMode(m) {
			t.Errorf("expected %q to be valid", m)
		}
	}
	if IsValidWebMode("bogus") {
		t.Error("expected \"bogus\" to be invalid")
	}
}

// TestWebRunnerTracerouteMode: WebRunner must be a no-op in traceroute mode.
// The TracerouteRunner (a separate runner) is responsible for this mode.
func TestWebRunnerTracerouteMode(t *testing.T) {
	fetcher := &stubFetcher{}
	resolver := &countingResolver{}
	comparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{resolver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewWebRunner(fetcher, comparator, logger)

	if err := runner.Run(context.Background(), makeWebRequest(WebModeTraceroute)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetcher.called || resolver.calls != 0 {
		t.Fatal("WebRunner must be a no-op in traceroute mode")
	}
}

// TestWebModeTracerouteConstant verifies the constant value is stable.
func TestWebModeTracerouteConstant(t *testing.T) {
	if WebModeTraceroute != "traceroute" {
		t.Fatalf("expected WebModeTraceroute = \"traceroute\", got %q", WebModeTraceroute)
	}
}
