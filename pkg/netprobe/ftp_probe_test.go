package netprobe

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// startFTPServer starts a minimal fake FTP server for testing.
// handler gets each accepted connection; the listener is closed when the test ends.
func startFTPServer(t *testing.T, handler func(net.Conn)) (host string, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(conn)
		}
	}()
	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

// simpleFTPHandler simulates a minimal FTP server that accepts anonymous login and no-ops.
func simpleFTPHandler(conn net.Conn) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	rw.WriteString("220 go-pathprobe test FTP\r\n")
	rw.Flush()

	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "USER"):
			rw.WriteString("331 Password required\r\n")
		case strings.HasPrefix(cmd, "PASS"):
			rw.WriteString("230 Login successful\r\n")
		case cmd == "QUIT":
			rw.WriteString("221 Bye\r\n")
			rw.Flush()
			return
		default:
			rw.WriteString("502 Not implemented\r\n")
		}
		rw.Flush()
	}
}

// TestDialFTPProberBanner verifies the prober reads the server banner.
func TestDialFTPProberBanner(t *testing.T) {
	host, port := startFTPServer(t, simpleFTPHandler)

	prober := &DialFTPProber{}
	res, err := prober.Probe(context.Background(), FTPProbeRequest{
		Host:    host,
		Port:    port,
		Timeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !strings.Contains(res.Banner, "220") {
		t.Fatalf("expected 220 banner, got %q", res.Banner)
	}
}

// TestDialFTPProberLoginAccepted verifies USER/PASS flow sets LoginAccepted.
func TestDialFTPProberLoginAccepted(t *testing.T) {
	host, port := startFTPServer(t, simpleFTPHandler)

	prober := &DialFTPProber{}
	res, err := prober.Probe(context.Background(), FTPProbeRequest{
		Host:     host,
		Port:     port,
		Username: "user",
		Password: "pass",
		Timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !res.LoginAccepted {
		t.Fatalf("expected login accepted")
	}
}

// TestDialFTPProberMissingHost validates guard for empty host.
func TestDialFTPProberMissingHost(t *testing.T) {
	prober := &DialFTPProber{}
	_, err := prober.Probe(context.Background(), FTPProbeRequest{})
	if err == nil {
		t.Fatalf("expected error for missing host")
	}
}

// TestDialFTPProberAuthTLS verifies explicit FTPS (AUTH TLS) flow.
func TestDialFTPProberAuthTLS(t *testing.T) {
	cert := generateTestCert(t)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	host, port := startFTPServer(t, func(conn net.Conn) {
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		rw.WriteString("220 FTP ready\r\n")
		rw.Flush()

		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case cmd == "AUTH TLS":
				rw.WriteString("234 Begin TLS negotiation\r\n")
				rw.Flush()
				// Upgrade to TLS.
				tlsConn := tls.Server(conn, tlsCfg)
				if err := tlsConn.Handshake(); err != nil {
					return
				}
				conn = tlsConn
				rw = bufio.NewReadWriter(bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn))
			case strings.HasPrefix(cmd, "PBSZ"):
				rw.WriteString("200 PBSZ=0\r\n")
			case strings.HasPrefix(cmd, "PROT"):
				rw.WriteString("200 Protection level set\r\n")
			case strings.HasPrefix(cmd, "USER"):
				rw.WriteString("331 Password\r\n")
			case strings.HasPrefix(cmd, "PASS"):
				rw.WriteString("230 Logged in\r\n")
			default:
				rw.WriteString("502 Unknown\r\n")
			}
			rw.Flush()
		}
	})

	prober := &DialFTPProber{}
	res, err := prober.Probe(context.Background(), FTPProbeRequest{
		Host:     host,
		Port:     port,
		AuthTLS:  true,
		Insecure: true,
		Username: "user",
		Password: "pass",
		Timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !res.UsedAuthTLS {
		t.Fatalf("expected AUTH TLS used")
	}
	if !res.LoginAccepted {
		t.Fatalf("expected login accepted after AUTH TLS")
	}
}

// TestDialFTPProberImplicitTLS verifies implicit FTPS (immediate TLS handshake) flow.
func TestDialFTPProberImplicitTLS(t *testing.T) {
	cert := generateTestCert(t)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	host, port := startFTPServer(t, func(conn net.Conn) {
		// Immediately upgrade to TLS.
		tlsConn := tls.Server(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return
		}
		defer tlsConn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn))
		rw.WriteString("220 FTPS ready\r\n")
		rw.Flush()
		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "USER"):
				rw.WriteString("331 Password\r\n")
			case strings.HasPrefix(cmd, "PASS"):
				rw.WriteString("230 OK\r\n")
			default:
				rw.WriteString("502 Unknown\r\n")
			}
			rw.Flush()
		}
	})

	prober := &DialFTPProber{}
	res, err := prober.Probe(context.Background(), FTPProbeRequest{
		Host:     host,
		Port:     port,
		UseTLS:   true,
		Insecure: true,
		Username: "user",
		Password: "pass",
		Timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !res.UsedImplicitTLS {
		t.Fatalf("expected implicit TLS used")
	}
	if !res.LoginAccepted {
		t.Fatalf("expected login accepted over implicit TLS")
	}
}

// TestDialFTPProberPASVList verifies PASV + LIST directory listing flow.
func TestDialFTPProberPASVList(t *testing.T) {
	// Start a data listener first to get its port.
	dataLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("data listen: %v", err)
	}
	t.Cleanup(func() { dataLn.Close() })
	dataAddr := dataLn.Addr().(*net.TCPAddr)

	go func() {
		conn, err := dataLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("-rw-r--r-- 1 ftp ftp 0 Jan 1 00:00 readme.txt\r\n"))
	}()

	host, port := startFTPServer(t, func(conn net.Conn) {
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		rw.WriteString("220 FTP\r\n")
		rw.Flush()

		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "USER"):
				rw.WriteString("331 Password\r\n")
			case strings.HasPrefix(cmd, "PASS"):
				rw.WriteString("230 OK\r\n")
			case cmd == "PASV":
				ip := dataAddr.IP.To4()
				p1 := dataAddr.Port / 256
				p2 := dataAddr.Port % 256
				rw.WriteString(fmt.Sprintf("227 Entering Passive Mode (%d,%d,%d,%d,%d,%d)\r\n",
					ip[0], ip[1], ip[2], ip[3], p1, p2))
			case cmd == "LIST":
				rw.WriteString("150 Opening data connection\r\n")
				rw.Flush()
				rw.WriteString("226 Transfer complete\r\n")
			default:
				rw.WriteString("502 Unknown\r\n")
			}
			rw.Flush()
		}
	})

	prober := &DialFTPProber{}
	res, err := prober.Probe(context.Background(), FTPProbeRequest{
		Host:     host,
		Port:     port,
		Username: "user",
		Password: "pass",
		RunLIST:  true,
		Timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !res.LoginAccepted {
		t.Fatalf("expected login accepted")
	}
	if len(res.ListEntries) == 0 {
		t.Fatalf("expected at least one LIST entry")
	}
}

// TestParsePASV verifies the PASV tuple parser handles correct and malformed input.
func TestParsePASV(t *testing.T) {
	host, port, err := parsePASV("227 Entering Passive Mode (192,168,1,10,195,149)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "192.168.1.10" {
		t.Fatalf("unexpected host %q", host)
	}
	// 195*256 + 149 = 50069
	if port != 50069 {
		t.Fatalf("unexpected port %d", port)
	}

	if _, _, err := parsePASV("227 Bad response"); err == nil {
		t.Fatalf("expected error for malformed PASV")
	}
}
