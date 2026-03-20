package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// ConnectivityRunner performs TCP port reachability probes with MTR-style repetition.
type ConnectivityRunner struct {
	Prober netprobe.PortProber
	Logger *slog.Logger
}

// NewConnectivityRunner builds a connectivity runner.
func NewConnectivityRunner(prober netprobe.PortProber, logger *slog.Logger) *ConnectivityRunner {
	return &ConnectivityRunner{Prober: prober, Logger: logger}
}

// Run executes port probes based on target and options.
func (r *ConnectivityRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}

	host := req.Options.Net.Host
	if host == "" {
		host = "example.com"
	}

	ports := req.Options.Net.Ports
	if len(ports) == 0 {
		ports = DefaultPorts(req.Target)
	}

	req.Emitf("network", "Probing %d port(s) on %s …", len(ports), host)

	results, err := netprobe.ProbePorts(ctx, host, ports, req.Options.Global.MTRCount, r.Prober)
	if err != nil {
		return err
	}

	for _, res := range results {
		req.Emitf("port_result", "Port %d: sent=%d recv=%d loss=%.1f%% avg=%s",
			res.Port, res.Stats.Sent, res.Stats.Received,
			res.Stats.LossPct, res.Stats.AvgRTT)
		r.Logger.Info("port probe", "target", req.Target, "host", host, "port", res.Port, "loss_pct", res.Stats.LossPct, "avg_rtt", res.Stats.AvgRTT, "min_rtt", res.Stats.MinRTT, "max_rtt", res.Stats.MaxRTT, "sent", res.Stats.Sent, "received", res.Stats.Received)
	}

	if req.Report != nil {
		req.Report.AddPorts(results)
	}
	return nil
}
