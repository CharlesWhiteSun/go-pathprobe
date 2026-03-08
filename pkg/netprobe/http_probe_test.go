package netprobe

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestClientHTTPProberSuccess covers status extraction and TLS metadata.
func TestClientHTTPProberSuccess(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	prober := &ClientHTTPProber{Client: srv.Client()}
	res, err := prober.Probe(context.Background(), HTTPProbeRequest{URL: srv.URL, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if res.TLS == nil || res.TLS.Version == "" || len(res.TLS.PeerCerts) == 0 {
		t.Fatalf("expected TLS info populated")
	}
}

// TestClientHTTPProberBadURL validates input guard.
func TestClientHTTPProberBadURL(t *testing.T) {
	prober := &ClientHTTPProber{Client: &http.Client{}}
	if _, err := prober.Probe(context.Background(), HTTPProbeRequest{}); err == nil {
		t.Fatalf("expected error for empty url")
	}
}

// TestClientHTTPProberInsecure ensures insecure flag allows self-signed.
func TestClientHTTPProberInsecure(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// force a client without the server's CA; rely on InsecureSkipVerify
	custom := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	prober := &ClientHTTPProber{Client: custom}
	res, err := prober.Probe(context.Background(), HTTPProbeRequest{URL: srv.URL, Insecure: true})
	if err != nil {
		t.Fatalf("expected no error with insecure, got %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}
}
