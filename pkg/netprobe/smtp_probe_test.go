package netprobe

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// generateTestCert builds an ephemeral self-signed TLS certificate for testing.
func generateTestCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("x509 key pair: %v", err)
	}
	return tlsCert
}

// TestDialSMTPProberStartTLS ensures STARTTLS flow is executed and auth/login path works.
func TestDialSMTPProberStartTLS(t *testing.T) {
	cert := generateTestCert(t)
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen err: %v", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleSMTP(conn, cfg)
		}
	}()
	defer listener.Close()

	prober := &DialSMTPProber{}
	req := SMTPProbeRequest{
		Host:      listener.Addr().(*net.TCPAddr).IP.String(),
		Port:      listener.Addr().(*net.TCPAddr).Port,
		StartTLS:  true,
		Username:  "user",
		Password:  "pass",
		From:      "from@test",
		To:        []string{"rcpt@test"},
		HelloName: "tester",
		Timeout:   3 * time.Second,
	}
	res, err := prober.Probe(context.Background(), req)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !res.UsedStartTLS {
		t.Fatalf("expected STARTTLS to be used")
	}
	if len(res.AuthTried) == 0 || res.AuthTried[0] != "LOGIN" {
		t.Fatalf("expected LOGIN auth attempted")
	}
	if !res.MailFromAccepted || len(res.RcptAccepted) != 1 {
		t.Fatalf("expected mail/rcpt accepted")
	}
}

// TestDialSMTPProberMissingHost validates guard for empty host.
func TestDialSMTPProberMissingHost(t *testing.T) {
	prober := &DialSMTPProber{}
	_, err := prober.Probe(context.Background(), SMTPProbeRequest{})
	if err == nil {
		t.Fatalf("expected error for missing host")
	}
}

// TestSupportsStartTLS helper ensures parsing logic works.
func TestSupportsStartTLS(t *testing.T) {
	caps := []string{"250-PIPELINING", "250-STARTTLS", "250 AUTH PLAIN"}
	if !supportsStartTLS(caps) {
		t.Fatalf("expected starttls supported")
	}
}

// TestAuthLoginEnc ensures base64 encoding helper works.
func TestAuthLoginEnc(t *testing.T) {
	got := encodeBase64("user")
	if got != "dXNlcg==" {
		t.Fatalf("unexpected base64: %s", got)
	}
}

func handleSMTP(conn net.Conn, cfg *tls.Config) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	rw.WriteString("220 smtp.test\r\n")
	rw.Flush()

	// EHLO
	rw.ReadString('\n')
	rw.WriteString("250-test\r\n")
	rw.WriteString("250-STARTTLS\r\n")
	rw.WriteString("250 AUTH LOGIN\r\n")
	rw.Flush()

	// STARTTLS
	line, _ := rw.ReadString('\n')
	if !strings.HasPrefix(line, "STARTTLS") {
		return
	}
	rw.WriteString("220 ready\r\n")
	rw.Flush()

	tlsConn := tls.Server(conn, cfg)
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	rw = bufio.NewReadWriter(bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn))

	// EHLO after TLS
	rw.ReadString('\n')
	rw.WriteString("250-test\r\n")
	rw.WriteString("250 AUTH LOGIN\r\n")
	rw.Flush()

	// AUTH LOGIN
	rw.ReadString('\n')
	rw.WriteString("334 VXNlcm5hbWU6\r\n")
	rw.Flush()
	rw.ReadString('\n')
	rw.WriteString("334 UGFzc3dvcmQ6\r\n")
	rw.Flush()
	rw.ReadString('\n')
	rw.WriteString("235 ok\r\n")
	rw.Flush()

	// MAIL FROM
	rw.ReadString('\n')
	rw.WriteString("250 ok\r\n")
	rw.Flush()
	// RCPT TO
	rw.ReadString('\n')
	rw.WriteString("250 ok\r\n")
	rw.Flush()
}

// startMockSMTPServer starts a TCP listener and invokes handler for every accepted connection.
// The listener is automatically closed when the test ends.
func startMockSMTPServer(t *testing.T, handler func(net.Conn)) (host string, port int) {
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

// TestDialSMTPProberAuthPlain verifies AUTH PLAIN is selected when the server advertises it.
func TestDialSMTPProberAuthPlain(t *testing.T) {
	host, port := startMockSMTPServer(t, func(conn net.Conn) {
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		rw.WriteString("220 smtp.test\r\n")
		rw.Flush()
		rw.ReadString('\n') // EHLO
		rw.WriteString("250-test\r\n")
		rw.WriteString("250 AUTH PLAIN LOGIN\r\n")
		rw.Flush()
		rw.ReadString('\n') // AUTH PLAIN <token>
		rw.WriteString("235 ok\r\n")
		rw.Flush()
	})

	prober := &DialSMTPProber{}
	req := SMTPProbeRequest{
		Host:        host,
		Port:        port,
		Username:    "user",
		Password:    "pass",
		AuthMethods: []string{"PLAIN", "LOGIN"},
		Timeout:     3 * time.Second,
	}
	res, err := prober.Probe(context.Background(), req)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if len(res.AuthTried) == 0 || res.AuthTried[0] != "PLAIN" {
		t.Fatalf("expected PLAIN auth used, got %v", res.AuthTried)
	}
}

// TestDialSMTPProberAuthFallback verifies the auth selector falls through to the next supported method.
// The request prefers XOAUTH2 first, but the server only advertises PLAIN; XOAUTH2 must be skipped.
func TestDialSMTPProberAuthFallback(t *testing.T) {
	host, port := startMockSMTPServer(t, func(conn net.Conn) {
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		rw.WriteString("220 smtp.test\r\n")
		rw.Flush()
		rw.ReadString('\n') // EHLO
		rw.WriteString("250-test\r\n")
		rw.WriteString("250 AUTH PLAIN\r\n")
		rw.Flush()
		rw.ReadString('\n') // AUTH PLAIN <token>
		rw.WriteString("235 ok\r\n")
		rw.Flush()
	})

	prober := &DialSMTPProber{}
	req := SMTPProbeRequest{
		Host:        host,
		Port:        port,
		Username:    "user",
		Password:    "pass",
		AuthMethods: []string{"XOAUTH2", "PLAIN"},
		Timeout:     3 * time.Second,
	}
	res, err := prober.Probe(context.Background(), req)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if len(res.AuthTried) != 1 || res.AuthTried[0] != "PLAIN" {
		t.Fatalf("expected PLAIN fallback, got %v", res.AuthTried)
	}
}

// TestDialSMTPProberAuthXOAUTH2 verifies XOAUTH2 mechanism is correctly used when advertised.
func TestDialSMTPProberAuthXOAUTH2(t *testing.T) {
	host, port := startMockSMTPServer(t, func(conn net.Conn) {
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		rw.WriteString("220 smtp.test\r\n")
		rw.Flush()
		rw.ReadString('\n') // EHLO
		rw.WriteString("250-test\r\n")
		rw.WriteString("250 AUTH XOAUTH2\r\n")
		rw.Flush()
		rw.ReadString('\n') // AUTH XOAUTH2 <token>
		rw.WriteString("235 ok\r\n")
		rw.Flush()
	})

	prober := &DialSMTPProber{}
	req := SMTPProbeRequest{
		Host:        host,
		Port:        port,
		Username:    "user@example.com",
		Password:    "ya29.token",
		AuthMethods: []string{"XOAUTH2"},
		Timeout:     3 * time.Second,
	}
	res, err := prober.Probe(context.Background(), req)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if len(res.AuthTried) == 0 || res.AuthTried[0] != "XOAUTH2" {
		t.Fatalf("expected XOAUTH2 auth used, got %v", res.AuthTried)
	}
}

// TestDialSMTPProberImplicitTLS verifies the UseTLS path connects through immediate TLS upgrade.
func TestDialSMTPProberImplicitTLS(t *testing.T) {
	cert := generateTestCert(t)
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	host, port := startMockSMTPServer(t, func(conn net.Conn) {
		tlsConn := tls.Server(conn, cfg)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return
		}
		defer tlsConn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn))
		rw.WriteString("220 smtp.test TLS\r\n")
		rw.Flush()
		rw.ReadString('\n') // EHLO
		rw.WriteString("250-test\r\n")
		rw.WriteString("250 AUTH PLAIN\r\n")
		rw.Flush()
		rw.ReadString('\n') // AUTH PLAIN <token>
		rw.WriteString("235 ok\r\n")
		rw.Flush()
	})

	prober := &DialSMTPProber{}
	req := SMTPProbeRequest{
		Host:        host,
		Port:        port,
		Username:    "user",
		Password:    "pass",
		UseTLS:      true,
		AuthMethods: []string{"PLAIN"},
		Timeout:     3 * time.Second,
	}
	res, err := prober.Probe(context.Background(), req)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !strings.Contains(res.Banner, "TLS") {
		t.Fatalf("expected TLS banner, got %q", res.Banner)
	}
	if len(res.AuthTried) == 0 || res.AuthTried[0] != "PLAIN" {
		t.Fatalf("expected PLAIN auth over TLS, got %v", res.AuthTried)
	}
}

// TestSupportsAuthPlain unit-tests the PLAIN capability detection helper.
func TestSupportsAuthPlain(t *testing.T) {
	caps := []string{"250-PIPELINING", "250 AUTH PLAIN LOGIN"}
	if !supportsAuthPlain(caps) {
		t.Fatalf("expected auth plain supported")
	}
	if supportsAuthPlain([]string{"250 AUTH LOGIN"}) {
		t.Fatalf("expected auth plain NOT supported when only LOGIN advertised")
	}
}

// TestSupportsAuthXOAUTH2 unit-tests the XOAUTH2 capability detection helper.
func TestSupportsAuthXOAUTH2(t *testing.T) {
	if !supportsAuthXOAUTH2([]string{"250 AUTH XOAUTH2 PLAIN"}) {
		t.Fatalf("expected xoauth2 supported")
	}
	if supportsAuthXOAUTH2([]string{"250 AUTH PLAIN LOGIN"}) {
		t.Fatalf("expected xoauth2 NOT supported when not advertised")
	}
}

// TestBuildXOAUTH2Token verifies the SASL XOAUTH2 token format per Google's specification.
func TestBuildXOAUTH2Token(t *testing.T) {
	got := buildXOAUTH2Token("user@example.com", "ya29.token")
	want := encodeBase64("user=user@example.com\x01auth=Bearer ya29.token\x01\x01")
	if got != want {
		t.Fatalf("unexpected xoauth2 token:\n got  %s\n want %s", got, want)
	}
}
