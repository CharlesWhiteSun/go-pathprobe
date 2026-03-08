package netprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTPDNSResolver ensures DoH JSON responses are parsed into DNS answers correctly.
func TestHTTPDNSResolver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-json")
		w.Write([]byte(`{"Status":0,"Answer":[{"data":"93.184.216.34"}]}`))
	}))
	defer srv.Close()

	resolver := &HTTPDNSResolver{Client: srv.Client(), Endpoint: srv.URL, Name: "doh-test"}
	ans, err := resolver.Lookup(context.Background(), "example.com", RecordTypeA)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ans.Values) != 1 || ans.Values[0] != "93.184.216.34" {
		t.Fatalf("unexpected values: %#v", ans.Values)
	}
	if ans.Source != "doh-test" {
		t.Fatalf("unexpected source: %s", ans.Source)
	}
}

// TestDNSComparator confirms the comparator fans out to resolvers and aggregates results.
func TestDNSComparator(t *testing.T) {
	spy := &spyResolver{name: "r1", answers: map[string][]string{"example.com|A": {"1.1.1.1"}}}
	comparator := DNSComparator{Resolvers: []DNSResolver{spy}}
	comps, err := comparator.Compare(context.Background(), []string{"example.com"}, []RecordType{RecordTypeA})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(comps) != 1 || len(comps[0].Results) != 1 {
		t.Fatalf("unexpected comparison size")
	}
	if spy.calls != 1 {
		t.Fatalf("expected resolver called once, got %d", spy.calls)
	}
}

type spyResolver struct {
	name    string
	answers map[string][]string
	calls   int
}

func (s *spyResolver) Lookup(ctx context.Context, name string, rtype RecordType) (DNSAnswer, error) {
	s.calls++
	key := name + "|" + string(rtype)
	vals := s.answers[key]
	return DNSAnswer{Name: name, Type: rtype, Values: vals, Source: s.name}, nil
}
