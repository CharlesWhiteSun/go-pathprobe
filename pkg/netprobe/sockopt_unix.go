//go:build !windows

package netprobe

import "syscall"

// syscallFD converts the uintptr file descriptor returned by net.RawConn.Control
// to the platform-native type expected by syscall.SetsockoptInt (int on Unix).
func syscallFD(fd uintptr) int { return int(fd) }

// setIPv4TTL sets the IP_TTL socket option on a Unix TCP socket.
func setIPv4TTL(fd uintptr, ttl int) error {
	return syscall.SetsockoptInt(syscallFD(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}
