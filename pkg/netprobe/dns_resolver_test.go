package netprobe

import (
	"context"
	"errors"
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

// TestHTTPDNSResolverNon2xx verifies a non-2xx HTTP response is treated as an error.
func TestHTTPDNSResolverNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	resolver := &HTTPDNSResolver{Client: srv.Client(), Endpoint: srv.URL, Name: "doh-test"}
	if _, err := resolver.Lookup(context.Background(), "example.com", RecordTypeA); err == nil {
		t.Fatal("expected error for non-2xx status")
	}
}

// TestHTTPDNSResolverDNSErrorStatus verifies a non-zero RCODE in the JSON body is treated as an error.
func TestHTTPDNSResolverDNSErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-json")
		w.Write([]byte(`{"Status":3,"Answer":[]}`)) // NXDOMAIN
	}))
	defer srv.Close()

	resolver := &HTTPDNSResolver{Client: srv.Client(), Endpoint: srv.URL, Name: "doh-test"}
	if _, err := resolver.Lookup(context.Background(), "nonexistent.example", RecordTypeA); err == nil {
		t.Fatal("expected error for non-zero DNS status")
	}
}

// TestHTTPDNSResolverEmptyAnswer verifies no error but empty values when Answer is absent.
func TestHTTPDNSResolverEmptyAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-json")
		w.Write([]byte(`{"Status":0}`))
	}))
	defer srv.Close()

	resolver := &HTTPDNSResolver{Client: srv.Client(), Endpoint: srv.URL, Name: "doh-test"}
	ans, err := resolver.Lookup(context.Background(), "example.com", RecordTypeMX)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ans.Values) != 0 {
		t.Fatalf("expected empty values, got %v", ans.Values)
	}
}

// TestSystemResolverUnsupportedType verifies an error is returned for unknown record types.
func TestSystemResolverUnsupportedType(t *testing.T) {
	resolver := &SystemResolver{Name: "system"}
	_, err := resolver.Lookup(context.Background(), "example.com", RecordType("TXT"))
	if err == nil {
		t.Fatal("expected error for unsupported record type TXT")
	}
}

// TestParseRecordTypesValid confirms that supported type strings are parsed correctly.
func TestParseRecordTypesValid(t *testing.T) {
	cases := []struct {
		inputs []string
		length int
	}{
		{[]string{"A"}, 1},
		{[]string{"aaaa"}, 1},
		{[]string{"MX"}, 1},
		{[]string{"A", "AAAA", "MX"}, 3},
		{[]string{"a", "Mx"}, 2}, // case-insensitive
	}
	for _, c := range cases {
		got, err := ParseRecordTypes(c.inputs)
		if err != nil {
			t.Fatalf("ParseRecordTypes(%v) unexpected error: %v", c.inputs, err)
		}
		if len(got) != c.length {
			t.Fatalf("ParseRecordTypes(%v) = %d items, want %d", c.inputs, len(got), c.length)
		}
	}
}

// TestParseRecordTypesInvalid verifies that unsupported type strings are rejected.
func TestParseRecordTypesInvalid(t *testing.T) {
	_, err := ParseRecordTypes([]string{"TXT"})
	if err == nil {
		t.Fatal("expected error for unsupported type TXT")
	}
}

// TestParseRecordTypesEmpty ensures an empty (or all-blank) input returns an error.
func TestParseRecordTypesEmpty(t *testing.T) {
	_, err := ParseRecordTypes([]string{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

// TestTTLFromSecondsValid verifies valid numeric TTL strings parse to durations.
func TestTTLFromSecondsValid(t *testing.T) {
	cases := []struct {
		input string
		want  int64
	}{
		{"0", 0},
		{"300", 300},
		{"3600", 3600},
		{"  86400  ", 86400}, // trimmed whitespace
	}
	for _, c := range cases {
		got, err := TTLFromSeconds(c.input)
		if err != nil {
			t.Fatalf("TTLFromSeconds(%q) unexpected error: %v", c.input, err)
		}
		if int64(got.Seconds()) != c.want {
			t.Fatalf("TTLFromSeconds(%q) = %v, want %ds", c.input, got, c.want)
		}
	}
}

// TestTTLFromSecondsInvalid verifies non-numeric and negative values are rejected.
func TestTTLFromSecondsInvalid(t *testing.T) {
	cases := []string{"abc", "-1", "", "1.5"}
	for _, input := range cases {
		if _, err := TTLFromSeconds(input); err == nil {
			t.Fatalf("TTLFromSeconds(%q) expected error, got nil", input)
		}
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

// TestDNSComparatorMultipleDomains ensures all domain/type combinations are queried.
func TestDNSComparatorMultipleDomains(t *testing.T) {
	spy := &spyResolver{name: "r1", answers: map[string][]string{
		"a.test|A":  {"1.1.1.1"},
		"b.test|A":  {"2.2.2.2"},
		"a.test|MX": {"mail.a.test:10"},
	}}
	comparator := DNSComparator{Resolvers: []DNSResolver{spy}}
	comps, err := comparator.Compare(context.Background(), []string{"a.test", "b.test"}, []RecordType{RecordTypeA, RecordTypeMX})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2 domains × 2 types = 4 comparisons
	if len(comps) != 4 {
		t.Fatalf("expected 4 comparisons, got %d", len(comps))
	}
	if spy.calls != 4 {
		t.Fatalf("expected 4 resolver calls, got %d", spy.calls)
	}
}

// TestDNSComparatorResolverError verifies that a resolver failure propagates as an error.
func TestDNSComparatorResolverError(t *testing.T) {
	bad := &errorResolver{err: errors.New("resolution failed")}
	comparator := DNSComparator{Resolvers: []DNSResolver{bad}}
	if _, err := comparator.Compare(context.Background(), []string{"example.com"}, []RecordType{RecordTypeA}); err == nil {
		t.Fatal("expected error from failing resolver")
	}
}

// TestDNSComparisonHasDivergence verifies divergence is detected when resolvers disagree.
func TestDNSComparisonHasDivergence(t *testing.T) {
	r1 := &spyResolver{name: "r1", answers: map[string][]string{"example.com|A": {"1.1.1.1"}}}
	r2 := &spyResolver{name: "r2", answers: map[string][]string{"example.com|A": {"2.2.2.2"}}}
	comparator := DNSComparator{Resolvers: []DNSResolver{r1, r2}}
	comps, err := comparator.Compare(context.Background(), []string{"example.com"}, []RecordType{RecordTypeA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !comps[0].HasDivergence() {
		t.Fatal("expected divergence to be detected")
	}
}

// TestDNSComparisonNoDivergence verifies no divergence is reported when resolvers agree.
func TestDNSComparisonNoDivergence(t *testing.T) {
	r1 := &spyResolver{name: "r1", answers: map[string][]string{"example.com|A": {"1.1.1.1"}}}
	r2 := &spyResolver{name: "r2", answers: map[string][]string{"example.com|A": {"1.1.1.1"}}}
	comparator := DNSComparator{Resolvers: []DNSResolver{r1, r2}}
	comps, err := comparator.Compare(context.Background(), []string{"example.com"}, []RecordType{RecordTypeA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comps[0].HasDivergence() {
		t.Fatal("expected no divergence")
	}
}

// TestDNSComparisonHasDivergenceSortOrder verifies that answer-order differences alone
// do not count as divergence.
func TestDNSComparisonHasDivergenceSortOrder(t *testing.T) {
	r1 := &spyResolver{name: "r1", answers: map[string][]string{"example.com|A": {"1.1.1.1", "2.2.2.2"}}}
	r2 := &spyResolver{name: "r2", answers: map[string][]string{"example.com|A": {"2.2.2.2", "1.1.1.1"}}}
	comparator := DNSComparator{Resolvers: []DNSResolver{r1, r2}}
	comps, err := comparator.Compare(context.Background(), []string{"example.com"}, []RecordType{RecordTypeA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comps[0].HasDivergence() {
		t.Fatal("same values in different sort order should not be flagged as divergent")
	}
}

// TestDNSComparisonSingleResolver verifies HasDivergence is always false with one resolver.
func TestDNSComparisonSingleResolver(t *testing.T) {
	comp := DNSComparison{
		Results: []DNSAnswer{{Values: []string{"1.1.1.1"}}},
	}
	if comp.HasDivergence() {
		t.Fatal("single resolver cannot produce divergence")
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

type errorResolver struct {
	err error
}

func (e *errorResolver) Lookup(_ context.Context, _ string, _ RecordType) (DNSAnswer, error) {
	return DNSAnswer{}, e.err
}
