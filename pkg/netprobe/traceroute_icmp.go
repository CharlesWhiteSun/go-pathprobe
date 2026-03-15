package netprobe

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ICMPTracerouteProber discovers the network path to a host by sending ICMP
// Echo Request packets with incrementally increasing TTL values (1 … maxHops).
//
// Each TTL value receives attemptsPerHop probes.  When an intermediate router
// decrements the TTL to zero it sends back an ICMP "Time Exceeded" message,
// revealing its address.  The sequence terminates early when an ICMP "Echo
// Reply" is received from the destination host itself.
//
// Requires a raw ICMP socket, which needs elevated OS privileges:
//   - Windows: Administrator rights
//   - Linux:   root or CAP_NET_RAW capability
//
// Use TCPTracerouteProber as a privilege-free fallback.
type ICMPTracerouteProber struct {
	// HopTimeout overrides hopTimeout for a single per-hop wait.
	// Zero uses the package-level default (2 s).
	HopTimeout time.Duration

	// ReverseLookup controls whether PTR records are resolved for each hop.
	// Defaults to true; set false to skip DNS lookups entirely.
	ReverseLookup *bool
}

// hopTimeoutFor returns the configured HopTimeout or the package default.
func (p *ICMPTracerouteProber) hopTimeoutFor() time.Duration {
	if p.HopTimeout > 0 {
		return p.HopTimeout
	}
	return hopTimeout
}

// reverseLookupEnabled returns true unless the caller explicitly disabled it.
func (p *ICMPTracerouteProber) reverseLookupEnabled() bool {
	return p.ReverseLookup == nil || *p.ReverseLookup
}

// Trace implements TracerouteProber using raw ICMP Echo packets.
func (p *ICMPTracerouteProber) Trace(ctx context.Context, host string, maxHops, attemptsPerHop int) (RouteResult, error) {
	if maxHops <= 0 {
		return RouteResult{}, fmt.Errorf("maxHops must be > 0, got %d", maxHops)
	}
	if attemptsPerHop <= 0 {
		return RouteResult{}, fmt.Errorf("attemptsPerHop must be > 0, got %d", attemptsPerHop)
	}

	// Resolve destination once.
	dstAddrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return RouteResult{}, fmt.Errorf("traceroute: resolve %q: %w", host, err)
	}
	dstIP := net.ParseIP(dstAddrs[0]).To4()
	if dstIP == nil {
		return RouteResult{}, fmt.Errorf("traceroute: IPv6 not supported for ICMP mode, resolved %q", dstAddrs[0])
	}

	// Open a raw IPv4 ICMP socket for sending.
	// "ip4:icmp" requires elevated privileges; the caller is responsible for
	// checking availability via syscheck.RawICMPChecker before calling Trace.
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return RouteResult{}, fmt.Errorf("traceroute: open raw ICMP socket: %w", err)
	}
	defer conn.Close()

	hto := p.hopTimeoutFor()
	var result RouteResult
	id := icmpID() // unique identifier for this traceroute session

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		hop, reached := p.probeHop(ctx, conn, dstIP, ttl, id, attemptsPerHop, hto)
		result.Hops = append(result.Hops, hop)

		// Perform reverse DNS for responsive hops.
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

// probeHop sends attemptsPerHop ICMP Echo Request packets with the given TTL
// and collects the responses.  It returns the aggregated HopResult and a
// boolean indicating whether the destination was reached.
func (p *ICMPTracerouteProber) probeHop(
	ctx context.Context,
	conn *icmp.PacketConn,
	dstIP net.IP,
	ttl, id, attempts int,
	hto time.Duration,
) (HopResult, bool) {
	_ = conn.IPv4PacketConn().SetTTL(ttl) //nolint:errcheck — best effort

	var (
		probeAttempts []ProbeAttempt
		hopIP         string
		reached       bool
	)

	dst := &net.IPAddr{IP: dstIP}

	for seq := 0; seq < attempts; seq++ {
		select {
		case <-ctx.Done():
			probeAttempts = append(probeAttempts, ProbeAttempt{Success: false})
			continue
		default:
		}

		msg := buildICMPEcho(id, seq)
		start := time.Now()

		if _, err := conn.WriteTo(msg, dst); err != nil {
			probeAttempts = append(probeAttempts, ProbeAttempt{Success: false, Err: err})
			continue
		}

		deadline := time.Now().Add(hto)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		conn.SetReadDeadline(deadline) //nolint:errcheck

		buf := make([]byte, 512)
		n, peer, err := conn.ReadFrom(buf)
		rtt := time.Since(start)

		if err != nil {
			probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: false, Err: err})
			continue
		}

		respMsg, parseErr := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if parseErr != nil {
			probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: false})
			continue
		}

		peerStr := peer.String()
		switch respMsg.Type {
		case ipv4.ICMPTypeTimeExceeded:
			// A router decremented TTL to zero — this is the expected hop response.
			if hopIP == "" {
				hopIP = peerStr
			}
			probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: true})

		case ipv4.ICMPTypeEchoReply:
			// We reached the destination; validate the reply belongs to our session.
			if echo, ok := respMsg.Body.(*icmp.Echo); ok && echo.ID == id {
				hopIP = peerStr
				probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: true})
				reached = true
			}

		default:
			probeAttempts = append(probeAttempts, ProbeAttempt{RTT: rtt, Success: false})
		}
	}

	stats := computeStats(probeAttempts)

	return HopResult{TTL: ttl, IP: hopIP, Attempts: probeAttempts, Stats: stats}, reached
}

// buildICMPEcho constructs an ICMP Echo Request message body.
func buildICMPEcho(id, seq int) []byte {
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   id,
			Seq:  seq,
			Data: []byte("pathprobe"),
		},
	}
	b, _ := msg.Marshal(nil)
	return b
}

// icmpID returns a 16-bit identifier based on nanosecond timestamp entropy.
// It is not cryptographically random but is sufficient for correlating
// a single traceroute session's Echo / TimeExceeded pairs.
func icmpID() int {
	b := make([]byte, 8)
	t := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(b, uint64(t))
	return int(binary.BigEndian.Uint16(b[:2]))
}

// reverseLookup performs a PTR query for ip with a short timeout.
// Returns the first hostname on success, or "" on failure/timeout.
func reverseLookup(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), reverseHostnameTimeout)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	// Strip trailing dot that DNS conventionally appends.
	name := names[0]
	if len(name) > 0 && name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}
	return name
}
