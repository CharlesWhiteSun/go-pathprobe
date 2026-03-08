package diag

import (
	"context"
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
