// Package syscheck provides runtime detection of OS-level capabilities
// required for certain probe modes (e.g. raw ICMP sockets).
package syscheck

import "net"

// ICMPAvailability holds the result of a raw ICMP socket availability check.
type ICMPAvailability struct {
	// Available is true when a raw ICMP socket could be opened successfully.
	Available bool
	// Err carries the underlying OS error when Available is false.
	Err error
}

// Notice returns a human-readable status line suitable for logging or printing.
// When unavailable it explains the fallback mode so users understand the behaviour.
func (a ICMPAvailability) Notice() string {
	if a.Available {
		return "ICMP probe mode available"
	}
	return "ICMP unavailable (elevated privileges required); falling back to TCP probe mode"
}

// ICMPChecker tests whether raw ICMP probing is permitted by the OS.
// Implementations must be safe to call from main before any runner is started.
type ICMPChecker interface {
	Check() ICMPAvailability
}

// RawICMPChecker attempts to open a raw IPv4 ICMP socket to determine
// availability.  On Windows this requires Administrator rights; on Linux it
// requires root or the CAP_NET_RAW capability.
type RawICMPChecker struct{}

// Check tries to open a raw ICMP socket.  The socket is closed immediately;
// no packets are sent.  A failed open means ICMP is not available.
func (RawICMPChecker) Check() ICMPAvailability {
	conn, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return ICMPAvailability{Available: false, Err: err}
	}
	conn.Close()
	return ICMPAvailability{Available: true}
}
