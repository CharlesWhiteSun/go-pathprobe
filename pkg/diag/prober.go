package diag

import "go-pathprobe/pkg/netprobe"

// SelectTracerouteProber returns an OsTracerouteProber that delegates to the
// platform-native traceroute command (tracert on Windows, traceroute on
// Unix/macOS).
//
// The icmpAvailable parameter is accepted for API compatibility but no longer
// influences the returned prober.  OsTracerouteProber is preferred over the
// former ICMP / TCP probers because it correctly captures intermediate router
// IP addresses on all platforms and requires no elevated OS privileges.
//
// This factory is the single authoritative place that selects the prober
// implementation, making the policy easy to test without OS permissions.
func SelectTracerouteProber(_ bool) netprobe.TracerouteProber {
	return &netprobe.OsTracerouteProber{}
}
