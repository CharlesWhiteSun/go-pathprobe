package netprobe

import (
	"context"
	"time"
)

// HopResult captures the outcome of probing a single TTL hop on the route path.
// When no ICMP response is received within the per-hop timeout, IP and Hostname
// are empty strings (rendered as "???" by callers).
type HopResult struct {
	// TTL is the Time-To-Live value that triggered this hop's response.
	TTL int

	// IP is the responding router's IPv4 address, or "" when the hop timed out.
	IP string

	// Hostname is the reverse-DNS name for IP, resolved on a best-effort basis.
	// Empty when IP is empty or the PTR lookup fails/times out.
	Hostname string

	// Attempts holds the raw per-probe RTTs and success flags for this TTL.
	// Length equals the attemptsPerHop argument supplied to TracerouteProber.Trace.
	Attempts []ProbeAttempt

	// Stats aggregates the Attempts into loss% and min/avg/max RTT.
	Stats ProbeStats
}

// RouteResult holds the complete sequence of hops discovered during a traceroute.
// Hops is ordered by TTL (ascending).  The final hop is the destination host,
// or the last hop that responded before maxHops was reached.
type RouteResult struct {
	Hops []HopResult
}

// HopEmitter is an optional callback invoked by TracerouteProber.Trace after
// each TTL hop is fully probed (including any PTR lookup).  Implementations
// call it in-order, incrementally, so callers can stream per-hop results
// without waiting for the full traceroute to finish.
// Pass nil when incremental updates are not needed.
type HopEmitter func(hop HopResult)

// TracerouteProber executes a path-discovery traceroute to a host.
// Implementations must:
//   - increment TTL from 1 to maxHops (inclusive);
//   - send attemptsPerHop probes per TTL to compute per-hop statistics;
//   - stop early when the destination host responds;
//   - honour context cancellation / deadline;
//   - call onHop (when non-nil) once per completed hop, in TTL order.
//
// Two concrete implementations are provided:
//   - ICMPTracerouteProber: uses raw ICMP Echo; requires elevated privileges.
//   - TCPTracerouteProber:  uses TCP SYN + IP_TTL socket option; works without root.
type TracerouteProber interface {
	Trace(ctx context.Context, host string, maxHops, attemptsPerHop int, onHop HopEmitter) (RouteResult, error)
}

// hopTimeout is the per-hop deadline applied when waiting for an ICMP or RST response.
// It is intentionally shorter than the overall diagnostic timeout so that silent hops
// ("???") don't stall the entire traceroute.
const hopTimeout = 2 * time.Second

// reverseHostnameTimeout is the maximum time spent on a single PTR lookup.
const reverseHostnameTimeout = 500 * time.Millisecond
