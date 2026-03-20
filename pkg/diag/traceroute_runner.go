package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// TracerouteRunner discovers the network path to a host by sending TTL-limited
// probes and recording each responding hop.
//
// It satisfies the Runner interface and is designed to be composed inside a
// MultiRunner alongside ConnectivityRunner and protocol runners.
//
// Prober selects the probe strategy:
//   - netprobe.ICMPTracerouteProber — high-fidelity ICMP mode (requires elevated OS privileges)
//   - netprobe.TCPTracerouteProber  — privilege-free TCP fallback
//
// The choice of prober is the caller's responsibility; use syscheck.RawICMPChecker
// to decide at startup which implementation is appropriate.
type TracerouteRunner struct {
	Prober netprobe.TracerouteProber
	Logger *slog.Logger
}

// NewTracerouteRunner is a convenience constructor that validates its arguments.
func NewTracerouteRunner(prober netprobe.TracerouteProber, logger *slog.Logger) *TracerouteRunner {
	return &TracerouteRunner{Prober: prober, Logger: logger}
}

// Run executes the traceroute probe and emits per-hop progress events.
//
// Configuration is read from req.Options:
//   - Net.Host        — target hostname (defaults to "example.com")
//   - Net.MaxHops     — maximum TTL to probe (defaults to DefaultMaxHops)
//   - Global.MTRCount — probes per hop (defaults to DefaultMTRCount)
//
// Each discovered hop is emitted via req.Hook under the "traceroute" stage.
// On completion the full RouteResult is stored in req.Report via SetRoute.
func (r *TracerouteRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}

	host := req.Options.Net.Host
	if host == "" {
		host = "example.com"
	}

	maxHops := req.Options.Net.MaxHops
	if maxHops <= 0 {
		maxHops = DefaultMaxHops
	}

	attemptsPerHop := req.Options.Global.MTRCount
	if attemptsPerHop <= 0 {
		attemptsPerHop = DefaultMTRCount
	}

	req.Emitf("traceroute", "Tracing route to %s (max %d hops, %d probe(s)/hop) …", host, maxHops, attemptsPerHop)

	result, err := r.Prober.Trace(ctx, host, maxHops, attemptsPerHop)
	if err != nil {
		return err
	}

	for _, hop := range result.Hops {
		ip := hop.IP
		if ip == "" {
			ip = "???"
		}

		name := hop.Hostname
		if name == "" {
			name = ip
		}

		req.Emitf("traceroute", "  %2d  %-15s  %-40s  avg=%-10s  loss=%.1f%%",
			hop.TTL, ip, name, hop.Stats.AvgRTT, hop.Stats.LossPct)

		r.Logger.Info("traceroute hop",
			"ttl", hop.TTL,
			"ip", ip,
			"hostname", hop.Hostname,
			"avg_rtt", hop.Stats.AvgRTT,
			"loss_pct", hop.Stats.LossPct,
			"sent", hop.Stats.Sent,
			"received", hop.Stats.Received,
		)
	}

	if req.Report != nil {
		req.Report.SetRoute(&result)
	}
	return nil
}
