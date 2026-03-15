package diag

import (
	"testing"

	"go-pathprobe/pkg/netprobe"
)

// ── SelectTracerouteProber — prober selection policy ──────────────────────
//
// These tests serve as the programmatic equivalent of the manual 7-2 / 7-3
// verifications:
//   - 7-2 (non-admin / no CAP_NET_RAW): ICMP unavailable → TCP prober
//   - 7-3 (admin / CAP_NET_RAW):        ICMP available   → ICMP prober

// TestSelectTracerouteProber_ICMPAvailable corresponds to Phase 7 item 7-3:
// when the OS grants raw ICMP socket access, SelectTracerouteProber must
// return an *ICMPTracerouteProber so that high-fidelity hop detection is used.
func TestSelectTracerouteProber_ICMPAvailable(t *testing.T) {
	prober := SelectTracerouteProber(true)
	if _, ok := prober.(*netprobe.ICMPTracerouteProber); !ok {
		t.Errorf("ICMP available: expected *netprobe.ICMPTracerouteProber, got %T", prober)
	}
}

// TestSelectTracerouteProber_ICMPUnavailable corresponds to Phase 7 item 7-2:
// when the OS denies raw ICMP access (non-admin / insufficient privileges),
// SelectTracerouteProber must return a *TCPTracerouteProber as a
// privilege-free fallback so traceroute still works without elevation.
func TestSelectTracerouteProber_ICMPUnavailable(t *testing.T) {
	prober := SelectTracerouteProber(false)
	if _, ok := prober.(*netprobe.TCPTracerouteProber); !ok {
		t.Errorf("ICMP unavailable: expected *netprobe.TCPTracerouteProber, got %T", prober)
	}
}

// TestSelectTracerouteProber_ImplementsInterface verifies that both returned
// probers satisfy the netprobe.TracerouteProber interface, ensuring they can
// be used interchangeably by TracerouteRunner.
func TestSelectTracerouteProber_ImplementsInterface(t *testing.T) {
	for _, icmpAvailable := range []bool{true, false} {
		p := SelectTracerouteProber(icmpAvailable)
		var _ netprobe.TracerouteProber = p // compile-time check
		if p == nil {
			t.Errorf("icmpAvailable=%v: SelectTracerouteProber returned nil", icmpAvailable)
		}
	}
}
