package netprobe

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"
)

// HTTPProbeRequest contains inputs for an HTTP/HTTPS probe.
type HTTPProbeRequest struct {
	URL      string
	Insecure bool
	Timeout  time.Duration
}

// CertSummary captures minimal certificate metadata.
type CertSummary struct {
	Subject    string
	Issuer     string
	NotBefore  time.Time
	NotAfter   time.Time
	DNSNames   []string
	OCSPServer []string
}

// TLSInfo summarizes TLS handshake results.
type TLSInfo struct {
	Version        string
	CipherSuite    string
	ServerName     string
	PeerCerts      []CertSummary
	NegotiatedALPN string
}

// HTTPProbeResult captures HTTP status and TLS details.
type HTTPProbeResult struct {
	StatusCode int
	RTT        time.Duration
	TLS        *TLSInfo
}

// HTTPProber performs HTTP requests and extracts handshake metadata.
type HTTPProber interface {
	Probe(ctx context.Context, req HTTPProbeRequest) (HTTPProbeResult, error)
}

// ClientHTTPProber implements HTTPProber using net/http Client.
type ClientHTTPProber struct {
	Client *http.Client
}

// Probe executes the request and captures status/TLS info.
func (p *ClientHTTPProber) Probe(ctx context.Context, req HTTPProbeRequest) (HTTPProbeResult, error) {
	if req.URL == "" {
		return HTTPProbeResult{}, errors.New("url is required")
	}
	client := p.Client
	if client == nil {
		client = &http.Client{}
	}
	timeout := req.Timeout
	if timeout > 0 {
		client = cloneClientWithTimeout(client, timeout, req.Insecure)
	} else {
		client = cloneClientWithTLS(client, req.Insecure)
	}

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return HTTPProbeResult{}, err
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return HTTPProbeResult{}, err
	}
	defer resp.Body.Close()

	var tlsInfo *TLSInfo
	if resp.TLS != nil {
		tlsInfo = convertTLSState(resp.TLS)
	}

	return HTTPProbeResult{StatusCode: resp.StatusCode, RTT: time.Since(start), TLS: tlsInfo}, nil
}

func cloneClientWithTimeout(src *http.Client, timeout time.Duration, insecure bool) *http.Client {
	c := *src
	c.Timeout = timeout
	c.Transport = cloneTransport(src.Transport, insecure)
	return &c
}

func cloneClientWithTLS(src *http.Client, insecure bool) *http.Client {
	c := *src
	c.Transport = cloneTransport(src.Transport, insecure)
	return &c
}

func cloneTransport(rt http.RoundTripper, insecure bool) http.RoundTripper {
	base, ok := rt.(*http.Transport)
	if !ok {
		base = http.DefaultTransport.(*http.Transport)
	}
	cp := base.Clone()
	if cp.TLSClientConfig == nil {
		cp.TLSClientConfig = &tls.Config{}
	}
	cp.TLSClientConfig.InsecureSkipVerify = insecure
	return cp
}

func convertTLSState(state *tls.ConnectionState) *TLSInfo {
	info := &TLSInfo{
		Version:        tlsVersionString(state.Version),
		CipherSuite:    tls.CipherSuiteName(state.CipherSuite),
		ServerName:     state.ServerName,
		NegotiatedALPN: state.NegotiatedProtocol,
	}
	for _, cert := range state.PeerCertificates {
		info.PeerCerts = append(info.PeerCerts, CertSummary{
			Subject:    cert.Subject.String(),
			Issuer:     cert.Issuer.String(),
			NotBefore:  cert.NotBefore,
			NotAfter:   cert.NotAfter,
			DNSNames:   cert.DNSNames,
			OCSPServer: cert.OCSPServer,
		})
	}
	return info
}

func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS1.3"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS10:
		return "TLS1.0"
	default:
		return "unknown"
	}
}
