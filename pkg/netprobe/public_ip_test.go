package netprobe

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPPublicIPFetcher validates we can parse a well-formed HTTPS echo response for public IP.
func TestHTTPPublicIPFetcher(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("198.51.100.1"))
	}))
	defer srv.Close()

	fetcher := &HTTPPublicIPFetcher{Client: srv.Client(), URL: srv.URL, Source: "test"}
	res, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.IP != "198.51.100.1" {
		t.Fatalf("unexpected IP: %s", res.IP)
	}
	if res.Source != "test" {
		t.Fatalf("unexpected source: %s", res.Source)
	}
}

// TestHTTPPublicIPFetcherInvalidResponse guards against accepting malformed IP payloads.
func TestHTTPPublicIPFetcherInvalidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not-an-ip"))
	}))
	defer srv.Close()

	fetcher := &HTTPPublicIPFetcher{Client: srv.Client(), URL: srv.URL, Source: "test"}
	if _, err := fetcher.Fetch(context.Background()); err == nil {
		t.Fatalf("expected error for invalid IP response")
	}
}

// TestHTTPPublicIPFetcherNon2xx verifies that a non-2xx status code is treated as an error.
func TestHTTPPublicIPFetcherNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	fetcher := &HTTPPublicIPFetcher{Client: srv.Client(), URL: srv.URL, Source: "test"}
	if _, err := fetcher.Fetch(context.Background()); err == nil {
		t.Fatal("expected error for 500 status")
	}
}

// TestHTTPPublicIPFetcherRTT verifies that a positive RTT is recorded for each fetch.
func TestHTTPPublicIPFetcherRTT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("10.0.0.1"))
	}))
	defer srv.Close()

	fetcher := &HTTPPublicIPFetcher{Client: srv.Client(), URL: srv.URL, Source: "test"}
	res, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RTT < 0 {
		t.Fatalf("expected non-negative RTT, got %v", res.RTT)
	}
}

// TestSTUNPublicIPFetcherEmptyServer verifies that an empty Server field returns an error.
func TestSTUNPublicIPFetcherEmptyServer(t *testing.T) {
	fetcher := &STUNPublicIPFetcher{}
	if _, err := fetcher.Fetch(context.Background()); err == nil {
		t.Fatal("expected error for empty server")
	}
}

// TestSTUNPublicIPFetcher uses a mock UDP STUN server to validate the full binding-request flow.
func TestSTUNPublicIPFetcher(t *testing.T) {
	wantIP := "198.51.100.42"

	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	go func() {
		buf := make([]byte, 512)
		n, remoteAddr, err := pc.ReadFrom(buf)
		if err != nil || n < 20 {
			return
		}
		txID := make([]byte, 12)
		copy(txID, buf[8:20])
		resp := buildTestSTUNResponse(txID, net.ParseIP(wantIP).To4())
		pc.WriteTo(resp, remoteAddr)
	}()

	fetcher := &STUNPublicIPFetcher{
		Server:  pc.LocalAddr().String(),
		Timeout: 3 * time.Second,
	}
	res, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.IP != wantIP {
		t.Fatalf("expected IP %s, got %s", wantIP, res.IP)
	}
	if res.RTT < 0 {
		t.Fatalf("expected non-negative RTT")
	}
	// Source should default to "stun:<server>" when not set.
	if res.Source == "" {
		t.Fatal("expected non-empty Source")
	}
}

// TestSTUNPublicIPFetcherCustomSource verifies the Source label is honoured.
func TestSTUNPublicIPFetcherCustomSource(t *testing.T) {
	wantIP := "192.0.2.99"

	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	go func() {
		buf := make([]byte, 512)
		n, remoteAddr, _ := pc.ReadFrom(buf)
		if n < 20 {
			return
		}
		txID := make([]byte, 12)
		copy(txID, buf[8:20])
		pc.WriteTo(buildTestSTUNResponse(txID, net.ParseIP(wantIP).To4()), remoteAddr)
	}()

	fetcher := &STUNPublicIPFetcher{
		Server:  pc.LocalAddr().String(),
		Source:  "my-stun",
		Timeout: 3 * time.Second,
	}
	res, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != "my-stun" {
		t.Fatalf("expected source 'my-stun', got %q", res.Source)
	}
}

// TestMultiSourcePublicIPFetcherFirstSucceeds verifies the first source result is returned
// when it succeeds without attempting further sources.
func TestMultiSourcePublicIPFetcherFirstSucceeds(t *testing.T) {
	first := &stubIPFetcher{ip: "1.2.3.4"}
	second := &stubIPFetcher{ip: "5.6.7.8"}
	f := &MultiSourcePublicIPFetcher{Sources: []PublicIPFetcher{first, second}}
	res, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IP != "1.2.3.4" {
		t.Fatalf("expected first source IP, got %s", res.IP)
	}
	if second.fetched {
		t.Fatal("second source should not be called when first succeeds")
	}
}

// TestMultiSourcePublicIPFetcherFallback verifies fallback to the second source when the first fails.
func TestMultiSourcePublicIPFetcherFallback(t *testing.T) {
	first := &failIPFetcher{}
	second := &stubIPFetcher{ip: "9.9.9.9"}
	f := &MultiSourcePublicIPFetcher{Sources: []PublicIPFetcher{first, second}}
	res, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}
	if res.IP != "9.9.9.9" {
		t.Fatalf("expected fallback IP, got %s", res.IP)
	}
}

// TestMultiSourcePublicIPFetcherAllFail verifies an error is returned when all sources fail.
func TestMultiSourcePublicIPFetcherAllFail(t *testing.T) {
	f := &MultiSourcePublicIPFetcher{
		Sources: []PublicIPFetcher{&failIPFetcher{}, &failIPFetcher{}},
	}
	if _, err := f.Fetch(context.Background()); err == nil {
		t.Fatal("expected error when all sources fail")
	}
}

// TestMultiSourcePublicIPFetcherNoSources verifies an error for an empty sources list.
func TestMultiSourcePublicIPFetcherNoSources(t *testing.T) {
	f := &MultiSourcePublicIPFetcher{}
	if _, err := f.Fetch(context.Background()); err == nil {
		t.Fatal("expected error for no sources")
	}
}

// buildTestSTUNResponse constructs a minimal STUN Binding Success Response containing
// an XOR-MAPPED-ADDRESS attribute for the given IPv4 address.
func buildTestSTUNResponse(txID []byte, ip net.IP) []byte {
	const mc uint32 = 0x2112A442
	xorIP := binary.BigEndian.Uint32(ip.To4()) ^ mc

	attr := make([]byte, 12)
	binary.BigEndian.PutUint16(attr[0:2], 0x0020) // XOR-MAPPED-ADDRESS
	binary.BigEndian.PutUint16(attr[2:4], 8)      // Value length
	attr[4] = 0x00                                // Reserved
	attr[5] = 0x01                                // Family: IPv4
	binary.BigEndian.PutUint16(attr[6:8], 0)      // XOR port (unused in test)
	binary.BigEndian.PutUint32(attr[8:12], xorIP)

	msg := make([]byte, 20+len(attr))
	binary.BigEndian.PutUint16(msg[0:2], 0x0101) // Binding Success Response
	binary.BigEndian.PutUint16(msg[2:4], uint16(len(attr)))
	binary.BigEndian.PutUint32(msg[4:8], mc)
	copy(msg[8:20], txID)
	copy(msg[20:], attr)
	return msg
}

type stubIPFetcher struct {
	ip      string
	fetched bool
}

func (s *stubIPFetcher) Fetch(_ context.Context) (PublicIPResult, error) {
	s.fetched = true
	return PublicIPResult{IP: s.ip, Source: "stub"}, nil
}

type failIPFetcher struct{}

func (f *failIPFetcher) Fetch(_ context.Context) (PublicIPResult, error) {
	return PublicIPResult{}, errors.New("intentional failure")
}
