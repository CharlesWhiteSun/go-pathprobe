package diag

import (
	"context"
	"fmt"
	"log/slog"
	neturl "net/url"
	"time"

	"go-pathprobe/pkg/netprobe"
)

// HTTPRunner performs HTTP/HTTPS probe to capture status and TLS info.
type HTTPRunner struct {
	Prober netprobe.HTTPProber
	Logger *slog.Logger
}

// NewHTTPRunner builds an HTTPRunner.
func NewHTTPRunner(prober netprobe.HTTPProber, logger *slog.Logger) *HTTPRunner {
	return &HTTPRunner{Prober: prober, Logger: logger}
}

// Run executes HTTP probe using Web options.
// It is a no-op for any explicit web mode other than WebModeHTTP so that
// single-select web diagnostics do not unexpectedly trigger an HTTP probe.
func (r *HTTPRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}
	// Skip when an explicit web mode is set that is not the HTTP probe mode.
	mode := req.Options.Web.Mode
	if mode != WebModeAll && mode != WebModeHTTP {
		return nil
	}
	url := req.Options.Web.URL
	if url == "" {
		req.Emitf("http", "HTTP probe skipped: no URL specified")
		return nil
	}
	// Normalise scheme: if the user omitted https:// or http://, prepend https://.
	// net/http rejects schemeless strings with "unsupported protocol scheme".
	if parsed, err := neturl.Parse(url); err == nil && parsed.Scheme == "" {
		url = "https://" + url
		req.Emitf("http", "No scheme detected — assuming HTTPS: %s", url)
	}
	req.Emitf("http", "Probing HTTP %s …", url)

	res, err := r.Prober.Probe(ctx, netprobe.HTTPProbeRequest{
		URL:      url,
		Insecure: req.Options.Global.Insecure,
		Timeout:  req.Options.Global.Timeout,
	})
	if err != nil {
		return err
	}
	req.Emitf("http_result", "HTTP %d, RTT %s", res.StatusCode, res.RTT.Round(time.Millisecond))
	r.Logger.Info("http probe", "url", url, "status", res.StatusCode, "rtt", res.RTT)
	if res.TLS != nil {
		r.Logger.Info("tls info", "version", res.TLS.Version, "cipher", res.TLS.CipherSuite, "alpn", res.TLS.NegotiatedALPN)
	}

	host := req.Options.Net.Host
	scheme := "http"
	if parsed, err := neturl.Parse(url); err == nil {
		if parsed.Hostname() != "" {
			host = parsed.Hostname()
		}
		if parsed.Scheme != "" {
			scheme = parsed.Scheme
		}
	}
	summary := fmt.Sprintf("HTTP %d, RTT %s", res.StatusCode, res.RTT.Round(time.Millisecond))
	if req.Report != nil {
		req.Report.AddProto(ProtoResult{
			Protocol: scheme,
			Host:     host,
			OK:       res.StatusCode >= 200 && res.StatusCode < 400,
			Summary:  summary,
		})
	}
	return nil
}
