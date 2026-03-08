package netprobe

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// PublicIPFetcher fetches public IP information.
type PublicIPFetcher interface {
	Fetch(ctx context.Context) (PublicIPResult, error)
}

// HTTPPublicIPFetcher calls an HTTPS echo endpoint that returns plain IP text.
type HTTPPublicIPFetcher struct {
	Client *http.Client
	URL    string
	Source string
}

// Fetch implements PublicIPFetcher.
func (f *HTTPPublicIPFetcher) Fetch(ctx context.Context) (PublicIPResult, error) {
	if f.Client == nil {
		f.Client = http.DefaultClient
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return PublicIPResult{}, err
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		return PublicIPResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PublicIPResult{}, errors.New("public IP endpoint returned non-2xx")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PublicIPResult{}, err
	}

	ipStr := strings.TrimSpace(string(body))
	if net.ParseIP(ipStr) == nil {
		return PublicIPResult{}, errors.New("invalid IP response")
	}

	return PublicIPResult{IP: ipStr, Source: f.Source, RTT: time.Since(start)}, nil
}

// stunMagicCookie is the fixed value defined by RFC 5389.
const stunMagicCookie uint32 = 0x2112A442

// STUNPublicIPFetcher discovers the public/NAT IP by sending a STUN Binding Request
// over UDP (RFC 5389) and parsing the XOR-MAPPED-ADDRESS from the response.
type STUNPublicIPFetcher struct {
	Server  string        // host:port, e.g. "stun.l.google.com:19302"
	Source  string        // label for PublicIPResult.Source; defaults to "stun:<Server>"
	Timeout time.Duration // defaults to 3s when zero
}

// Fetch implements PublicIPFetcher via STUN.
func (f *STUNPublicIPFetcher) Fetch(ctx context.Context) (PublicIPResult, error) {
	if f.Server == "" {
		return PublicIPResult{}, errors.New("stun server address is required")
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	addr, err := net.ResolveUDPAddr("udp", f.Server)
	if err != nil {
		return PublicIPResult{}, fmt.Errorf("stun resolve: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return PublicIPResult{}, fmt.Errorf("stun dial: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	conn.SetDeadline(deadline) //nolint:errcheck

	// Build STUN Binding Request: 20-byte fixed header, no attributes.
	txID := make([]byte, 12)
	if _, err := rand.Read(txID); err != nil {
		return PublicIPResult{}, err
	}
	msg := make([]byte, 20)
	binary.BigEndian.PutUint16(msg[0:2], 0x0001) // Binding Request
	binary.BigEndian.PutUint16(msg[2:4], 0)      // Attributes length
	binary.BigEndian.PutUint32(msg[4:8], stunMagicCookie)
	copy(msg[8:20], txID)

	start := time.Now()
	if _, err := conn.Write(msg); err != nil {
		return PublicIPResult{}, fmt.Errorf("stun write: %w", err)
	}
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return PublicIPResult{}, fmt.Errorf("stun read: %w", err)
	}
	rtt := time.Since(start)

	resp := buf[:n]
	if len(resp) < 20 {
		return PublicIPResult{}, errors.New("stun: response too short")
	}
	if binary.BigEndian.Uint16(resp[0:2]) != 0x0101 {
		return PublicIPResult{}, fmt.Errorf("stun: unexpected message type 0x%04x", binary.BigEndian.Uint16(resp[0:2]))
	}

	ip, err := parseSTUNMappedAddress(resp[20:])
	if err != nil {
		return PublicIPResult{}, err
	}

	source := f.Source
	if source == "" {
		source = "stun:" + f.Server
	}
	return PublicIPResult{IP: ip, Source: source, RTT: rtt}, nil
}

// parseSTUNMappedAddress walks the TLV attribute list and returns the first
// IPv4 address from XOR-MAPPED-ADDRESS (0x0020) or MAPPED-ADDRESS (0x0001).
func parseSTUNMappedAddress(attrs []byte) (string, error) {
	for len(attrs) >= 4 {
		aType := binary.BigEndian.Uint16(attrs[0:2])
		aLen := int(binary.BigEndian.Uint16(attrs[2:4]))
		padded := (aLen + 3) &^ 3
		if len(attrs) < 4+padded {
			break
		}
		val := attrs[4 : 4+aLen]
		switch aType {
		case 0x0020: // XOR-MAPPED-ADDRESS
			if len(val) >= 8 && val[1] == 0x01 { // IPv4
				xorAddr := binary.BigEndian.Uint32(val[4:8]) ^ stunMagicCookie
				ip := net.IP{byte(xorAddr >> 24), byte(xorAddr >> 16), byte(xorAddr >> 8), byte(xorAddr)}
				return ip.String(), nil
			}
		case 0x0001: // MAPPED-ADDRESS (legacy fallback)
			if len(val) >= 8 && val[1] == 0x01 { // IPv4
				return net.IP(val[4:8]).String(), nil
			}
		}
		attrs = attrs[4+padded:]
	}
	return "", errors.New("stun: no mapped address attribute found")
}

// MultiSourcePublicIPFetcher tries each source in order and returns the first
// successful result, providing transparent fallback between STUN and HTTPS echo.
type MultiSourcePublicIPFetcher struct {
	Sources []PublicIPFetcher
}

// Fetch implements PublicIPFetcher using the first source that succeeds.
func (f *MultiSourcePublicIPFetcher) Fetch(ctx context.Context) (PublicIPResult, error) {
	if len(f.Sources) == 0 {
		return PublicIPResult{}, errors.New("no public IP sources configured")
	}
	var lastErr error
	for _, src := range f.Sources {
		res, err := src.Fetch(ctx)
		if err == nil {
			return res, nil
		}
		lastErr = err
	}
	return PublicIPResult{}, lastErr
}
