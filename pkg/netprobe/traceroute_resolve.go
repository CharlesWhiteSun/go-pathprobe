package netprobe

import "net"

// pickIPv4Addr scans a resolved address list and returns the first address
// that parses as an IPv4 address.  Returns an empty string when no IPv4
// address is found, allowing callers to apply their own fallback policy:
//
//   - ICMPTracerouteProber requires IPv4 (raw ICMP over IPv6 is not supported);
//     an empty return must be treated as an unrecoverable error.
//   - TCPTracerouteProber can fall back to the first address in the list,
//     which may be an IPv6 address that the OS TCP stack handles natively.
//
// The function is a pure, side-effect-free helper; it performs no I/O and
// is safe to call from concurrent goroutines.
func pickIPv4Addr(addrs []string) string {
	for _, addr := range addrs {
		if net.ParseIP(addr).To4() != nil {
			return addr
		}
	}
	return ""
}
