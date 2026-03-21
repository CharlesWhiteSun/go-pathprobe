package netprobe

import "testing"

// TestPickIPv4Addr_Empty verifies that an empty address list returns an empty
// string — the caller must handle this as a "no address found" signal.
func TestPickIPv4Addr_Empty(t *testing.T) {
	if got := pickIPv4Addr(nil); got != "" {
		t.Errorf("nil addrs: expected empty, got %q", got)
	}
	if got := pickIPv4Addr([]string{}); got != "" {
		t.Errorf("empty addrs: expected empty, got %q", got)
	}
}

// TestPickIPv4Addr_IPv4Only verifies that a list containing only IPv4 addresses
// returns the first one unchanged.
func TestPickIPv4Addr_IPv4Only(t *testing.T) {
	addrs := []string{"1.2.3.4", "5.6.7.8"}
	if got := pickIPv4Addr(addrs); got != "1.2.3.4" {
		t.Errorf("IPv4-only: expected %q, got %q", "1.2.3.4", got)
	}
}

// TestPickIPv4Addr_IPv6Only verifies that a list containing only IPv6 addresses
// returns an empty string, signalling that no IPv4 address is available.
func TestPickIPv4Addr_IPv6Only(t *testing.T) {
	addrs := []string{"2404:6800:4012:7::2004", "::1", "fe80::1"}
	if got := pickIPv4Addr(addrs); got != "" {
		t.Errorf("IPv6-only: expected empty, got %q", got)
	}
}

// TestPickIPv4Addr_IPv6First verifies that when the resolver returns IPv6 before
// IPv4 (the real-world scenario that caused the traceroute failure for
// www.google.com), the function skips the leading IPv6 addresses and returns
// the first IPv4 address found.
func TestPickIPv4Addr_IPv6First(t *testing.T) {
	addrs := []string{
		"2404:6800:4012:7::2004", // IPv6 — should be skipped
		"142.250.196.100",        // IPv4 — should be returned
		"142.250.196.101",        // IPv4 — second, not returned
	}
	want := "142.250.196.100"
	if got := pickIPv4Addr(addrs); got != want {
		t.Errorf("IPv6-first list: expected %q, got %q", want, got)
	}
}

// TestPickIPv4Addr_Mixed_PreservesOrder verifies that when multiple IPv4
// addresses appear in the list the first one (by list position) is returned.
func TestPickIPv4Addr_Mixed_PreservesOrder(t *testing.T) {
	addrs := []string{
		"::1",      // IPv6, skip
		"10.0.0.1", // first IPv4
		"10.0.0.2", // second IPv4, not chosen
	}
	want := "10.0.0.1"
	if got := pickIPv4Addr(addrs); got != want {
		t.Errorf("expected first IPv4 %q, got %q", want, got)
	}
}

// TestPickIPv4Addr_SingleIPv4 verifies that a single-element IPv4 list is
// handled correctly (regression guard for the trivial case).
func TestPickIPv4Addr_SingleIPv4(t *testing.T) {
	addrs := []string{"93.184.216.34"}
	if got := pickIPv4Addr(addrs); got != "93.184.216.34" {
		t.Errorf("single IPv4: expected %q, got %q", "93.184.216.34", got)
	}
}

// TestPickIPv4Addr_SingleIPv6 verifies that a single-element IPv6 list returns
// an empty string rather than the IPv6 address, so ICMP callers get a clear
// "no IPv4 available" signal instead of a nil-pointer panic.
func TestPickIPv4Addr_SingleIPv6(t *testing.T) {
	addrs := []string{"2001:db8::1"}
	if got := pickIPv4Addr(addrs); got != "" {
		t.Errorf("single IPv6: expected empty, got %q", got)
	}
}

// TestPickIPv4Addr_MappedIPv4 verifies that IPv4-mapped IPv6 addresses
// (::ffff:1.2.3.4) are correctly identified as IPv4 by net.ParseIP.To4() and
// returned by pickIPv4Addr, consistent with Go's standard library behaviour.
func TestPickIPv4Addr_MappedIPv4(t *testing.T) {
	// net.ParseIP("::ffff:1.2.3.4").To4() returns the IPv4 part "1.2.3.4".
	addrs := []string{"::ffff:1.2.3.4"}
	got := pickIPv4Addr(addrs)
	if got == "" {
		t.Error("IPv4-mapped IPv6 address should be treated as IPv4 by To4()")
	}
}
