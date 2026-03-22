package netprobe

import (
	"testing"
	"time"
)

// ── parseOsHopLine ─────────────────────────────────────────────────────────

// TestParseOsHopLine_WindowsSuccess verifies a normal Windows tracert hop line
// with three RTT measurements and an IP address.
func TestParseOsHopLine_WindowsSuccess(t *testing.T) {
	line := "  1    <1 ms    <1 ms    <1 ms  192.168.1.1"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 1 {
		t.Errorf("TTL: got %d, want 1", hop.TTL)
	}
	if hop.IP != "192.168.1.1" {
		t.Errorf("IP: got %q, want %q", hop.IP, "192.168.1.1")
	}
	if hop.Stats.Sent != 3 {
		t.Errorf("Sent: got %d, want 3", hop.Stats.Sent)
	}
	if hop.Stats.Received != 3 {
		t.Errorf("Received: got %d, want 3", hop.Stats.Received)
	}
	if hop.Stats.LossPct != 0 {
		t.Errorf("LossPct: got %.1f, want 0", hop.Stats.LossPct)
	}
}

// TestParseOsHopLine_WindowsTimeout verifies a Windows tracert line where all
// three probes timed out ("*   *   *   Request timed out.").
func TestParseOsHopLine_WindowsTimeout(t *testing.T) {
	line := "  3     *        *        *     Request timed out."
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 3 {
		t.Errorf("TTL: got %d, want 3", hop.TTL)
	}
	if hop.IP != "" {
		t.Errorf("IP: got %q, want empty string", hop.IP)
	}
	if hop.Stats.Sent != 3 {
		t.Errorf("Sent: got %d, want 3", hop.Stats.Sent)
	}
	if hop.Stats.Received != 0 {
		t.Errorf("Received: got %d, want 0", hop.Stats.Received)
	}
	if hop.Stats.LossPct != 100 {
		t.Errorf("LossPct: got %.1f, want 100", hop.Stats.LossPct)
	}
}

// TestParseOsHopLine_WindowsLargerRTT verifies RTT values greater than 1 ms.
func TestParseOsHopLine_WindowsLargerRTT(t *testing.T) {
	line := "  2    13 ms    12 ms    14 ms  10.0.0.1"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 2 {
		t.Errorf("TTL: got %d, want 2", hop.TTL)
	}
	if hop.IP != "10.0.0.1" {
		t.Errorf("IP: got %q, want %q", hop.IP, "10.0.0.1")
	}
	want := 13 * time.Millisecond // avg of 13, 12, 14
	if hop.Stats.AvgRTT != want {
		t.Errorf("AvgRTT: got %v, want %v", hop.Stats.AvgRTT, want)
	}
}

// TestParseOsHopLine_WindowsMixed verifies a hop where some probes succeed and
// some time out (produces partial loss, which is normal on some networks).
func TestParseOsHopLine_WindowsMixed(t *testing.T) {
	line := "  4     5 ms     *        6 ms  172.16.0.1"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 4 {
		t.Errorf("TTL: got %d, want 4", hop.TTL)
	}
	if hop.IP != "172.16.0.1" {
		t.Errorf("IP: got %q, want %q", hop.IP, "172.16.0.1")
	}
	if hop.Stats.Sent != 3 {
		t.Errorf("Sent: got %d, want 3", hop.Stats.Sent)
	}
	if hop.Stats.Received != 2 {
		t.Errorf("Received: got %d, want 2", hop.Stats.Received)
	}
}

// TestParseOsHopLine_UnixSuccess verifies a normal Unix traceroute hop line
// (IP before RTT values, decimal milliseconds).
func TestParseOsHopLine_UnixSuccess(t *testing.T) {
	line := " 2  10.0.0.1  5.123 ms  4.987 ms  5.234 ms"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 2 {
		t.Errorf("TTL: got %d, want 2", hop.TTL)
	}
	if hop.IP != "10.0.0.1" {
		t.Errorf("IP: got %q, want %q", hop.IP, "10.0.0.1")
	}
	if hop.Stats.Sent != 3 {
		t.Errorf("Sent: got %d, want 3", hop.Stats.Sent)
	}
	if hop.Stats.Received != 3 {
		t.Errorf("Received: got %d, want 3", hop.Stats.Received)
	}
}

// TestParseOsHopLine_UnixTimeout verifies a Unix hop line with all timeouts.
func TestParseOsHopLine_UnixTimeout(t *testing.T) {
	line := " 3  * * *"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 3 {
		t.Errorf("TTL: got %d, want 3", hop.TTL)
	}
	if hop.IP != "" {
		t.Errorf("IP: got %q, want empty string", hop.IP)
	}
	if hop.Stats.LossPct != 100 {
		t.Errorf("LossPct: got %.1f, want 100", hop.Stats.LossPct)
	}
}

// TestParseOsHopLine_UnixMixed verifies a Unix hop line with mixed RTTs and
// timeouts, which can occur when the router applies ICMP rate-limiting.
func TestParseOsHopLine_UnixMixed(t *testing.T) {
	line := " 4  10.1.0.1  8.1 ms  *  7.9 ms"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 4 {
		t.Errorf("TTL: got %d, want 4", hop.TTL)
	}
	if hop.IP != "10.1.0.1" {
		t.Errorf("IP: got %q, want %q", hop.IP, "10.1.0.1")
	}
	if hop.Stats.Sent != 3 {
		t.Errorf("Sent: got %d, want 3", hop.Stats.Sent)
	}
	if hop.Stats.Received != 2 {
		t.Errorf("Received: got %d, want 2", hop.Stats.Received)
	}
}

// TestParseOsHopLine_SkipsNonHopLines verifies that header, footer, and blank
// lines are correctly ignored.
func TestParseOsHopLine_SkipsNonHopLines(t *testing.T) {
	nonHopLines := []string{
		"Tracing route to dns.google [8.8.8.8]",
		"over a maximum of 30 hops:",
		"Trace complete.",
		"traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets",
		"",
		"   ",
	}
	for _, line := range nonHopLines {
		if _, ok := parseOsHopLine(line); ok {
			t.Errorf("line %q: expected ok=false, got true", line)
		}
	}
}

// TestParseOsHopLine_RTTPrecision checks that decimal RTT values from Unix
// traceroute are parsed with sub-millisecond precision.
func TestParseOsHopLine_RTTPrecision(t *testing.T) {
	line := " 1  192.168.1.1  0.500 ms  0.500 ms  0.500 ms"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := 500 * time.Microsecond
	if hop.Stats.AvgRTT != want {
		t.Errorf("AvgRTT: got %v, want %v", hop.Stats.AvgRTT, want)
	}
}

// TestParseOsHopLine_DestinationIP checks parsing of the final hop that
// matches the destination IP.
func TestParseOsHopLine_DestinationIP(t *testing.T) {
	line := " 10     7 ms     7 ms     7 ms  8.8.8.8"
	hop, ok := parseOsHopLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hop.TTL != 10 {
		t.Errorf("TTL: got %d, want 10", hop.TTL)
	}
	if hop.IP != "8.8.8.8" {
		t.Errorf("IP: got %q, want %q", hop.IP, "8.8.8.8")
	}
}

// ── OsTracerouteProber interface check ────────────────────────────────────

var _ TracerouteProber = (*OsTracerouteProber)(nil)

// ── osTracerouteArgs ──────────────────────────────────────────────────────

// TestOsTracerouteArgs_ArgStructure verifies that the args slice is
// non-empty and the first element (the command name) is non-empty.
func TestOsTracerouteArgs_ArgStructure(t *testing.T) {
	args := osTracerouteArgs("8.8.8.8", 30, 3)
	if len(args) == 0 {
		t.Fatal("expected non-empty args slice")
	}
	if args[0] == "" {
		t.Error("args[0] (command name) must not be empty")
	}
	// Host must appear somewhere in the args slice.
	found := false
	for _, a := range args {
		if a == "8.8.8.8" {
			found = true
		}
	}
	if !found {
		t.Errorf("host not found in args: %v", args)
	}
}

// TestOsTracerouteArgs_MaxHopsPresent verifies that maxHops value appears in
// the args, ensuring the native tool respects the configured limit.
func TestOsTracerouteArgs_MaxHopsPresent(t *testing.T) {
	args := osTracerouteArgs("1.1.1.1", 15, 3)
	found := false
	for _, a := range args {
		if a == "15" {
			found = true
		}
	}
	if !found {
		t.Errorf("maxHops value '15' not found in args: %v", args)
	}
}
