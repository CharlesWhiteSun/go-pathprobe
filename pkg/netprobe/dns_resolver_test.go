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

// TestDNSComparatorResolverError verifies that a non-context resolver failure is recorded
// as a LookupError in the comparison result rather than being propagated as a fatal error.
// This allows the user to see results from the remaining resolvers.
func TestDNSComparatorResolverError(t *testing.T) {
	bad := &errorResolver{name: "bad", err: errors.New("resolution failed")}
	ok := &spyResolver{name: "ok", answers: map[string][]string{"example.com|A": {"1.2.3.4"}}}
	comparator := DNSComparator{Resolvers: []DNSResolver{bad, ok}}
	comps, err := comparator.Compare(context.Background(), []string{"example.com"}, []RecordType{RecordTypeA})
	if err != nil {
		t.Fatalf("expected no error from comparator: non-context resolver failures must not be fatal; got %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(comps))
	}
	comp := comps[0]
	if len(comp.Results) != 2 {
		t.Fatalf("expected 2 results (failing + ok), got %d", len(comp.Results))
	}
	// Failing resolver result must have LookupError set and no values.
	failed := comp.Results[0]
	if failed.LookupError == "" {
		t.Error("expected LookupError to be set for the failing resolver")
	}
	if len(failed.Values) != 0 {
		t.Errorf("unexpected Values in failed result: %v", failed.Values)
	}
	// Successful resolver result must be normal.
	good := comp.Results[1]
	if good.LookupError != "" {
		t.Errorf("unexpected LookupError on successful resolver: %q", good.LookupError)
	}
	if len(good.Values) != 1 || good.Values[0] != "1.2.3.4" {
		t.Errorf("unexpected Values on good resolver: %v", good.Values)
	}
}

// TestDNSComparatorNXDOMAINRecorded verifies that NXDOMAIN-style errors are recorded
// as LookupError entries and the comparison still succeeds, allowing the user to see
// that no resolver could resolve the domain.
func TestDNSComparatorNXDOMAINRecorded(t *testing.T) {
	nxdomain := &errorResolver{name: "sys", err: errors.New("lookup notexist.example: no such host")}
	comparator := DNSComparator{Resolvers: []DNSResolver{nxdomain}}
	comps, err := comparator.Compare(context.Background(), []string{"notexist.example"}, []RecordType{RecordTypeA})
	if err != nil {
		t.Fatalf("NXDOMAIN must not be a fatal error; got %v", err)
	}
	if len(comps[0].Results) != 1 {
		t.Fatalf("expected 1 result entry, got %d", len(comps[0].Results))
	}
	if comps[0].Results[0].LookupError == "" {
		t.Error("expected LookupError to contain the resolver error message")
	}
}

// TestDNSComparatorContextCancel verifies that context cancellation IS propagated
// as a fatal error, stopping the comparison immediately.
func TestDNSComparatorContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	// Use a resolver that wraps the context error so it reaches compareSingle.
	bad := &errorResolver{name: "ctx", err: context.Canceled}
	comparator := DNSComparator{Resolvers: []DNSResolver{bad}}
	if _, err := comparator.Compare(ctx, []string{"example.com"}, []RecordType{RecordTypeA}); err == nil {
		t.Fatal("expected error for context cancellation")
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

// TestDNSComparisonAllEmpty verifies that AllEmpty returns true when every
// resolver successfully responded with no records (e.g. a domain has no MX).
func TestDNSComparisonAllEmpty(t *testing.T) {
	comp := DNSComparison{
		Name: "example.com", Type: RecordTypeMX,
		Results: []DNSAnswer{
			{Source: "sys", Values: nil},
			{Source: "cf", Values: []string{}},
		},
	}
	if !comp.AllEmpty() {
		t.Fatal("expected AllEmpty=true when all resolvers return no records")
	}
	if comp.HasDivergence() {
		t.Fatal("all-empty results must not be flagged as divergent")
	}
}

// TestDNSComparisonAllEmptyFalseWhenHasValues verifies AllEmpty is false
// when at least one resolver returned records.
func TestDNSComparisonAllEmptyFalseWhenHasValues(t *testing.T) {
	comp := DNSComparison{
		Name: "example.com", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "sys", Values: []string{"1.2.3.4"}},
			{Source: "cf", Values: []string{}},
		},
	}
	if comp.AllEmpty() {
		t.Fatal("expected AllEmpty=false when at least one resolver has values")
	}
}

// TestDNSComparisonAllEmptyFalseWhenHasLookupError verifies that AllEmpty
// returns false when any result carries a LookupError, distinguishing
// "resolver failure" from "no records found".
func TestDNSComparisonAllEmptyFalseWhenHasLookupError(t *testing.T) {
	comp := DNSComparison{
		Name: "notexist.example", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "sys", LookupError: "lookup notexist.example: no such host"},
			{Source: "cf", Values: []string{}},
		},
	}
	if comp.AllEmpty() {
		t.Fatal("expected AllEmpty=false when a resolver has a LookupError")
	}
}

// TestDNSComparisonAllEmptyFalseWhenNoResults verifies AllEmpty is false
// for an empty Results slice (no resolvers configured).
func TestDNSComparisonAllEmptyFalseWhenNoResults(t *testing.T) {
	comp := DNSComparison{Name: "example.com", Type: RecordTypeA}
	if comp.AllEmpty() {
		t.Fatal("expected AllEmpty=false when Results slice is empty")
	}
}

// ---------------------------------------------------------------------------
// AllFailed tests
// ---------------------------------------------------------------------------

// TestDNSComparisonAllFailed verifies AllFailed returns true when every
// resolver carries a LookupError.  This is the root cause of the bug where
// three failing resolvers all returned Values=[] and HasDivergence=false,
// misleadingly showing "Consistent".
func TestDNSComparisonAllFailed(t *testing.T) {
	comp := DNSComparison{
		Name: "https://www.example.com/", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "system", LookupError: "lookup https://www.example.com/: no such host"},
			{Source: "doh-1.1.1.1", LookupError: "resolver returned error status"},
			{Source: "doh-8.8.8.8", LookupError: "resolver returned error status"},
		},
	}
	if !comp.AllFailed() {
		t.Fatal("expected AllFailed=true when every resolver has a LookupError")
	}
}

// TestDNSComparisonAllFailedFalseWhenOneSucceeds verifies AllFailed is false
// when at least one resolver succeeded (even with empty records).
func TestDNSComparisonAllFailedFalseWhenOneSucceeds(t *testing.T) {
	comp := DNSComparison{
		Name: "example.com", Type: RecordTypeMX,
		Results: []DNSAnswer{
			{Source: "sys", LookupError: "no such host"},
			{Source: "cf", Values: []string{}}, // succeeded, just no records
		},
	}
	if comp.AllFailed() {
		t.Fatal("expected AllFailed=false when one resolver succeeded")
	}
}

// TestDNSComparisonAllFailedFalseWhenNoResults verifies AllFailed is false
// for an empty Results slice (no resolvers configured).
func TestDNSComparisonAllFailedFalseWhenNoResults(t *testing.T) {
	comp := DNSComparison{Name: "example.com", Type: RecordTypeA}
	if comp.AllFailed() {
		t.Fatal("expected AllFailed=false when Results slice is empty")
	}
}

// TestDNSComparisonAllFailedNotDivergent verifies that AllFailed results are
// not also marked as HasDivergence: all nil Values compare equal, so the
// fix must be at the badge-selection layer rather than in HasDivergence.
func TestDNSComparisonAllFailedNotDivergent(t *testing.T) {
	comp := DNSComparison{
		Name: "bad.example", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "sys", LookupError: "no such host"},
			{Source: "cf", LookupError: "resolver returned error status"},
		},
	}
	if comp.HasDivergence() {
		t.Fatal("all-failed results must not be flagged as divergent (Values are all nil/empty)")
	}
	if !comp.AllFailed() {
		t.Fatal("expected AllFailed=true when all resolvers errored")
	}
}

// TestDNSComparisonAllFailedWithValues verifies AllFailed is false when a
// resolver returned actual records even alongside errored resolvers.
func TestDNSComparisonAllFailedWithValues(t *testing.T) {
	comp := DNSComparison{
		Name: "example.com", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "sys", Values: []string{"93.184.216.34"}},
			{Source: "cf", LookupError: "resolver returned error status"},
		},
	}
	if comp.AllFailed() {
		t.Fatal("expected AllFailed=false when at least one resolver has Values")
	}
}

// ---------------------------------------------------------------------------
// NoneFound tests
// ---------------------------------------------------------------------------

// TestDNSComparisonNoneFound verifies the primary trigger for NoneFound: the
// system resolver returns NXDOMAIN while DoH resolvers return empty (no error).
// Previously this fell through to "Consistent" because HasDivergence=false
// (all Values are empty) and AllEmpty=false (system has LookupError).
func TestDNSComparisonNoneFound(t *testing.T) {
	comp := DNSComparison{
		Name: "24h.pchome.com.tw", Type: RecordTypeAAAA,
		Results: []DNSAnswer{
			{Source: "system", LookupError: "lookup 24h.pchome.com.tw: no such host"},
			{Source: "doh-1.1.1.1", Values: []string{}},
			{Source: "doh-8.8.8.8", Values: []string{}},
		},
	}
	if !comp.NoneFound() {
		t.Fatal("expected NoneFound=true: system=NXDOMAIN, DoH=empty — no resolver returned records")
	}
	// Must not also claim HasDivergence (Values are all empty).
	if comp.HasDivergence() {
		t.Fatal("NoneFound case must not be flagged as HasDivergence")
	}
	// Must not claim AllFailed (DoH resolvers succeeded with empty results).
	if comp.AllFailed() {
		t.Fatal("NoneFound case must not claim AllFailed when DoH resolvers responded without error")
	}
}

// TestDNSComparisonNoneFoundFalseWhenHasValues verifies NoneFound is false
// when at least one resolver returned actual records.
func TestDNSComparisonNoneFoundFalseWhenHasValues(t *testing.T) {
	comp := DNSComparison{
		Name: "example.com", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "sys", Values: []string{"1.2.3.4"}},
			{Source: "cf", Values: []string{"1.2.3.4"}},
		},
	}
	if comp.NoneFound() {
		t.Fatal("expected NoneFound=false when a resolver returned actual records")
	}
}

// TestDNSComparisonNoneFoundFalseWhenAllFailed verifies NoneFound is false
// when all resolvers failed — AllFailed has higher badge priority.
func TestDNSComparisonNoneFoundFalseWhenAllFailed(t *testing.T) {
	comp := DNSComparison{
		Name: "bad.example", Type: RecordTypeA,
		Results: []DNSAnswer{
			{Source: "sys", LookupError: "no such host"},
			{Source: "cf", LookupError: "resolver returned error status"},
		},
	}
	if comp.NoneFound() {
		t.Fatal("expected NoneFound=false when AllFailed — AllFailed has higher badge priority")
	}
	if !comp.AllFailed() {
		t.Fatal("fixture setup: expected AllFailed=true")
	}
}

// TestDNSComparisonNoneFoundSubsumesAllEmpty verifies that when AllEmpty is
// true (all resolvers returned empty results with no errors), NoneFound is also
// true.  AllEmpty is a strict subset of NoneFound.
func TestDNSComparisonNoneFoundSubsumesAllEmpty(t *testing.T) {
	comp := DNSComparison{
		Name: "example.com", Type: RecordTypeMX,
		Results: []DNSAnswer{
			{Source: "sys", Values: nil},
			{Source: "cf", Values: []string{}},
		},
	}
	if !comp.AllEmpty() {
		t.Fatal("fixture setup: expected AllEmpty=true")
	}
	if !comp.NoneFound() {
		t.Fatal("expected NoneFound=true when AllEmpty=true (AllEmpty ⊆ NoneFound)")
	}
}

// TestDNSComparisonNoneFoundFalseWhenNoResults verifies NoneFound is false
// for an empty Results slice (no resolvers configured).
func TestDNSComparisonNoneFoundFalseWhenNoResults(t *testing.T) {
	comp := DNSComparison{Name: "example.com", Type: RecordTypeA}
	if comp.NoneFound() {
		t.Fatal("expected NoneFound=false when Results slice is empty")
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
	name string
	err  error
}

func (e *errorResolver) Lookup(_ context.Context, name string, rtype RecordType) (DNSAnswer, error) {
	return DNSAnswer{Name: name, Type: rtype, Source: e.name}, e.err
}
