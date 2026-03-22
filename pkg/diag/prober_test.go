package diag

import (
	"testing"

	"go-pathprobe/pkg/netprobe"
)

// ── SelectTracerouteProber — prober selection policy ──────────────────────
//
// SelectTracerouteProber always returns an *OsTracerouteProber regardless of
// the icmpAvailable flag.  The OS-native traceroute command (tracert on
// Windows, traceroute on Unix/macOS) correctly captures intermediate router
// IP addresses without requiring elevated privileges, superseding the former
// ICMP / TCP prober implementations.

// TestSelectTracerouteProber_AlwaysReturnsOsProber verifies that both the
// "ICMP available" and "ICMP unavailable" cases now return *OsTracerouteProber.
func TestSelectTracerouteProber_AlwaysReturnsOsProber(t *testing.T) {
	for _, icmpAvail := range []bool{true, false} {
		prober := SelectTracerouteProber(icmpAvail)
		if _, ok := prober.(*netprobe.OsTracerouteProber); !ok {
			t.Errorf("icmpAvail=%v: expected *netprobe.OsTracerouteProber, got %T", icmpAvail, prober)
		}
	}
}

// TestSelectTracerouteProber_ImplementsInterface verifies that the returned
// prober satisfies the netprobe.TracerouteProber interface so it can be used
// interchangeably by TracerouteRunner.
func TestSelectTracerouteProber_ImplementsInterface(t *testing.T) {
	for _, icmpAvailable := range []bool{true, false} {
		p := SelectTracerouteProber(icmpAvailable)
		var _ netprobe.TracerouteProber = p // compile-time check
		if p == nil {
			t.Errorf("icmpAvailable=%v: SelectTracerouteProber returned nil", icmpAvailable)
		}
	}
}
