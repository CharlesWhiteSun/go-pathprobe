package diag

import "go-pathprobe/pkg/netprobe"

// SelectTracerouteProber returns the appropriate TracerouteProber based on
// whether the OS grants raw ICMP socket access.
//
// When icmpAvailable is true an ICMPTracerouteProber is returned, which sends
// real ICMP echo requests and yields the highest-fidelity hop results.
// When false a TCPTracerouteProber is returned as a privilege-free fallback
// that uses TCP SYN probes instead.
//
// This factory is the single authoritative place that maps capability detection
// to a concrete prober, making the policy easy to test without OS permissions.
func SelectTracerouteProber(icmpAvailable bool) netprobe.TracerouteProber {
	if icmpAvailable {
		return &netprobe.ICMPTracerouteProber{}
	}
	return &netprobe.TCPTracerouteProber{}
}
