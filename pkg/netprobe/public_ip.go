package netprobe

import (
	"context"
	"errors"
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
