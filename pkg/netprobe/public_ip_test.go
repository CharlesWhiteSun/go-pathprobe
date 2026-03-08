package netprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
