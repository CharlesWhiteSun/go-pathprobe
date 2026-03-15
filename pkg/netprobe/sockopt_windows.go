//go:build windows

package netprobe

import "syscall"

// syscallFD converts the uintptr file descriptor to syscall.Handle on Windows.
func syscallFD(fd uintptr) syscall.Handle { return syscall.Handle(fd) }

// setIPv4TTL sets the IP_TTL socket option on a Windows TCP socket.
func setIPv4TTL(fd uintptr, ttl int) error {
	return syscall.SetsockoptInt(syscallFD(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}
