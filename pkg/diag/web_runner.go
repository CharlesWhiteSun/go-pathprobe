package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// WebRunner performs public IP detection and DNS comparison.
type WebRunner struct {
	fetcher      netprobe.PublicIPFetcher
	comparator   netprobe.DNSComparator
	defaultTypes []netprobe.RecordType
	logger       *slog.Logger
}

// WebOptions configures web diagnostics.
type WebOptions struct {
	Domains []string
	Types   []netprobe.RecordType
	URL     string
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

// Run executes public IP retrieval and DNS comparison.
func (r *WebRunner) Run(ctx context.Context, req Request) error {
	// Public IP
	req.Emit("web", "Fetching public IP address …")
	ipRes, err := r.fetcher.Fetch(ctx)
	if err != nil {
		return err
	}
	req.Emitf("web", "Public IP: %s (source: %s)", ipRes.IP, ipRes.Source)
	r.logger.Info("public ip fetched", "ip", ipRes.IP, "source", ipRes.Source, "rtt", ipRes.RTT)
	req.Report.SetPublicIP(ipRes.IP)

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
	return nil
}
