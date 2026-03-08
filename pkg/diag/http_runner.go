package diag

import (
	"context"
	"fmt"
	"log/slog"
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
func (r *HTTPRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}
	url := req.Options.Web.URL
	if url == "" {
		host := req.Options.Net.Host
		if host == "" {
			host = "example.com"
		}
		url = "https://" + host
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

	summary := fmt.Sprintf("HTTP %d, RTT %s", res.StatusCode, res.RTT.Round(time.Millisecond))
	req.Report.AddProto(ProtoResult{
		Protocol: "http",
		Host:     req.Options.Net.Host,
		OK:       res.StatusCode >= 200 && res.StatusCode < 400,
		Summary:  summary,
	})
	return nil
}
