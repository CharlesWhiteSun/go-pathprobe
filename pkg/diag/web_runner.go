package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// WebMode controls which sub-diagnostic a web target run executes.
// An empty value retains the legacy "run everything" behaviour for backwards
// compatibility with direct API and CLI callers that do not set the field.
type WebMode string

const (
	WebModeAll        WebMode = ""           // legacy: public-IP + DNS + HTTP + port
	WebModePublicIP   WebMode = "public-ip"  // public IP detection only
	WebModeDNS        WebMode = "dns"        // DNS cross-resolver comparison only
	WebModeHTTP       WebMode = "http"       // HTTP/HTTPS protocol probe only
	WebModePort       WebMode = "port"       // port connectivity only
	WebModeTraceroute WebMode = "traceroute" // network path discovery (traceroute)
)

// IsValidWebMode reports whether m is a recognised web sub-mode value.
// The empty string ("") is valid and selects the legacy all-in-one behaviour.
func IsValidWebMode(m WebMode) bool {
	switch m {
	case WebModeAll, WebModePublicIP, WebModeDNS, WebModeHTTP, WebModePort, WebModeTraceroute:
		return true
	}
	return false
}

// WebRunner performs public IP detection and/or DNS comparison.
type WebRunner struct {
	fetcher      netprobe.PublicIPFetcher
	comparator   netprobe.DNSComparator
	defaultTypes []netprobe.RecordType
	logger       *slog.Logger
}

// WebOptions configures web diagnostics.
type WebOptions struct {
	Mode    WebMode
	Domains []string
	Types   []netprobe.RecordType
	URL     string
	// MaxHops sets the maximum TTL for traceroute sub-mode.
	// Zero means use diag.DefaultMaxHops.
	MaxHops int
}

// NewWebRunner wires public IP fetcher and DNS comparator.
func NewWebRunner(fetcher netprobe.PublicIPFetcher, comparator netprobe.DNSComparator, logger *slog.Logger) *WebRunner {
	return &WebRunner{
		fetcher:      fetcher,
		comparator:   comparator,
		defaultTypes: []netprobe.RecordType{netprobe.RecordTypeA, netprobe.RecordTypeAAAA, netprobe.RecordTypeMX},
		logger:       logger,
	}
}

// Run executes public IP retrieval and/or DNS comparison depending on Mode.
//
//   - WebModeAll (""):       both public IP and DNS run sequentially (legacy / backwards-compat).
//   - WebModePublicIP:       only the public IP fetch runs.
//   - WebModeDNS:            only the DNS cross-resolver comparison runs.
//   - WebModeHTTP, WebModePort, WebModeTraceroute: this runner is a no-op; other runners handle those modes.
func (r *WebRunner) Run(ctx context.Context, req Request) error {
	mode := req.Options.Web.Mode
	runPublicIP := mode == WebModeAll || mode == WebModePublicIP
	runDNS := mode == WebModeAll || mode == WebModeDNS

	if !runPublicIP && !runDNS {
		return nil // http, port, or traceroute mode — not this runner's responsibility
	}

	if runPublicIP {
		req.Emit("web", "Fetching public IP address …")
		ipRes, err := r.fetcher.Fetch(ctx)
		if err != nil {
			return err
		}
		req.Emitf("web", "Public IP: %s (source: %s)", ipRes.IP, ipRes.Source)
		r.logger.Info("public ip fetched", "ip", ipRes.IP, "source", ipRes.Source, "rtt", ipRes.RTT)
		if req.Report != nil {
			req.Report.SetPublicIP(ipRes.IP)
		}
	}

	if runDNS {
		webOpts := req.Options.Web
		domains := webOpts.Domains
		if len(domains) == 0 {
			domains = []string{"example.com"}
		}

		types := webOpts.Types
		if len(types) == 0 {
			types = r.defaultTypes
		}

		req.Emitf("dns", "Comparing DNS records for %d domain(s) …", len(domains))

		comparisons, err := r.comparator.Compare(ctx, domains, types)
		if err != nil {
			return err
		}
		for _, comp := range comparisons {
			req.Emitf("dns_result", "%-4s %s divergent=%v", comp.Type, comp.Name, comp.HasDivergence())
			r.logger.Info("dns comparison", "domain", comp.Name, "type", comp.Type, "divergent", comp.HasDivergence(), "results", comp.Results)
		}
	}

	return nil
}
