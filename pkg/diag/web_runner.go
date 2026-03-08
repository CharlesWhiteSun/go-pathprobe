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
	ipRes, err := r.fetcher.Fetch(ctx)
	if err != nil {
		return err
	}
	r.logger.Info("public ip fetched", "ip", ipRes.IP, "source", ipRes.Source, "rtt", ipRes.RTT)

	webOpts := req.Options.Web
	domains := webOpts.Domains
	if len(domains) == 0 {
		domains = []string{"example.com"}
	}

	types := webOpts.Types
	if len(types) == 0 {
		types = r.defaultTypes
	}

	comparisons, err := r.comparator.Compare(ctx, domains, types)
	if err != nil {
		return err
	}
	for _, comp := range comparisons {
		r.logger.Info("dns comparison", "domain", comp.Name, "type", comp.Type, "results", comp.Results)
	}
	return nil
}
