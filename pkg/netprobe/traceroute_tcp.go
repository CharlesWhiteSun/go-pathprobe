package netprobe

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"
)

// TCPTracerouteProber discovers the network path to a host by opening TCP
// connections with incrementally increasing IP TTL values (1 … maxHops).
//
// Each router that decrements the TTL to zero closes the connection; Go exposes
// the peer address via the Dialer's Control hook, and the ICMP "Time Exceeded"
// response is implicit in the connection error.  Unlike ICMPTracerouteProber
// this implementation does not require raw socket privileges.
//
// The RemotePort field selects which destination TCP port each probe connects
// to.  Port 80 is used by default; use a port that is expected to be
// reachable on most hosts so that the final hop is detected when the
// connection succeeds (or receives a TCP RST).
//
// Note: because we rely on a full TCP connect, each hop takes up to HopTimeout
// per attempt even if the router responds instantly.  The implementation is
// therefore slower than ICMPTracerouteProber for the same configuration.
type TCPTracerouteProber struct {
	// RemotePort is the TCP port to dial on the destination.  Defaults to 80.
	RemotePort int

	// HopTimeout overrides hopTimeout for a single per-hop wait.
	// Zero uses the package-level default (2 s).
	HopTimeout time.Duration

	// ReverseLookup controls whether PTR records are resolved for each hop.
	// Defaults to true; set false to skip DNS lookups entirely.
	ReverseLookup *bool
}

// hopTimeoutFor returns the configured HopTimeout or the package default.
func (p *TCPTracerouteProber) hopTimeoutFor() time.Duration {
	if p.HopTimeout > 0 {
		return p.HopTimeout
	}
	return hopTimeout
}

// reverseLookupEnabled returns true unless the caller explicitly disabled it.
func (p *TCPTracerouteProber) reverseLookupEnabled() bool {
	return p.ReverseLookup == nil || *p.ReverseLookup
}

// remotePort returns the configured remote port or 80.
func (p *TCPTracerouteProber) remotePort() int {
	if p.RemotePort > 0 && p.RemotePort <= 65535 {
		return p.RemotePort
	}
	return 80
}

// Trace implements TracerouteProber using TTL-limited TCP connections.
func (p *TCPTracerouteProber) Trace(ctx context.Context, host string, maxHops, attemptsPerHop int) (RouteResult, error) {
	if maxHops <= 0 {
		return RouteResult{}, fmt.Errorf("maxHops must be > 0, got %d", maxHops)
	}
	if attemptsPerHop <= 0 {
		return RouteResult{}, fmt.Errorf("attemptsPerHop must be > 0, got %d", attemptsPerHop)
	}

	// Resolve destination once so we can compare against the peer address.
	dstAddrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return RouteResult{}, fmt.Errorf("traceroute tcp: resolve %q: %w", host, err)
	}
	dstAddr := dstAddrs[0]
	target := fmt.Sprintf("%s:%d", dstAddr, p.remotePort())

	hto := p.hopTimeoutFor()
	var result RouteResult

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		hop, reached := p.probeHop(ctx, target, dstAddr, ttl, attemptsPerHop, hto)
		result.Hops = append(result.Hops, hop)

		if hop.IP != "" && p.reverseLookupEnabled() {
			hop.Hostname = reverseLookup(hop.IP)
			result.Hops[len(result.Hops)-1].Hostname = hop.Hostname
		}

		if reached {
			break
		}
	}

	return result, nil
}

// probeHop sends attemptsPerHop TCP connect attempts with the specified TTL.
// It returns the aggregated HopResult and whether the destination was reached.
func (p *TCPTracerouteProber) probeHop(
	ctx context.Context,
	target, dstAddr string,
	ttl, attempts int,
	hto time.Duration,
) (HopResult, bool) {
	var (
		probeAttempts []ProbeAttempt
		hopIP         string
		reached       bool
	)

	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			probeAttempts = append(probeAttempts, ProbeAttempt{Success: false})
			continue
		default:
		}

		hopCtx, cancel := context.WithTimeout(ctx, hto)
		peerIP, rtt, dialErr := p.dialWithTTL(hopCtx, target, ttl)
		cancel()

		if peerIP != "" && hopIP == "" {
			hopIP = peerIP
		}

		if dialErr == nil {
			// TCP connection succeeded — we reached the destination.
			reached = true
			probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: true})
			continue
		}

		// A TCP RST from the destination also "reaches" the host.
		if isConnectionRefused(dialErr) {
			if hopIP == "" {
				hopIP = dstAddr
			}
			reached = true
			probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: true})
			continue
		}

		// Any other dialling error (including TTL expired ICMPs that abort the
		// connect) means we heard from intermediate router peerIP.
		probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: false, Err: dialErr})
	}

	stats := computeStats(probeAttempts)
	return HopResult{TTL: ttl, IP: hopIP, Attempts: probeAttempts, Stats: stats}, reached
}

// dialWithTTL opens a TCP connection to target with the specified IP TTL,
// using a raw socket Control function to apply the socket option before connect.
// Returns the first peer IP observed (even on error), the elapsed duration, and
// any dial error.
func (p *TCPTracerouteProber) dialWithTTL(ctx context.Context, target string, ttl int) (peerIP string, rtt time.Duration, err error) {
	var capturedPeer string
	dialer := &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			capturedPeer = hostFromAddr(address)
			return c.Control(func(fd uintptr) {
				setIPv4TTL(fd, ttl) //nolint:errcheck — best effort
			})
		},
	}

	start := time.Now()
	conn, dialErr := dialer.DialContext(ctx, "tcp4", target)
	rtt = time.Since(start)

	if dialErr == nil {
		// Record the actual remote peer (may differ from capturedPeer after NAT).
		if ra := conn.RemoteAddr(); ra != nil {
			capturedPeer = hostFromAddr(ra.String())
		}
		conn.Close()
	}
	return capturedPeer, rtt, dialErr
}

// isConnectionRefused reports whether err represents a TCP RST / ECONNREFUSED,
// which means the destination host was reached (port simply not open).
func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*net.OpError); ok {
			_ = sysErr
		}
		// Unwrap through *os.SyscallError layers.
		inner := opErr.Unwrap()
		if inner == nil {
			inner = opErr.Err
		}
		switch inner {
		case syscall.ECONNREFUSED:
			return true
		}
	}
	return false
}

// hostFromAddr extracts the host portion from a "host:port" or bare address
// string, returning the original string if no colon is present.
func hostFromAddr(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
