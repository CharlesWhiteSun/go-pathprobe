package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// SFTPRunner performs SSH handshake and optional SFTP directory listing probe.
type SFTPRunner struct {
	Prober netprobe.SFTPProber
	Logger *slog.Logger
}

// NewSFTPRunner constructs an SFTPRunner.
func NewSFTPRunner(prober netprobe.SFTPProber, logger *slog.Logger) *SFTPRunner {
	return &SFTPRunner{Prober: prober, Logger: logger}
}

// Run executes the SFTP/SSH probe based on target options.
func (r *SFTPRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}

	host := req.Options.Net.Host
	if host == "" {
		host = "localhost"
	}
	port := choosePort(req.Options.Net.Ports, TargetSFTP)
	opts := req.Options.SFTP

	res, err := r.Prober.Probe(ctx, netprobe.SFTPProbeRequest{
		Host:       host,
		Port:       port,
		Username:   opts.Username,
		Password:   opts.Password,
		PrivateKey: opts.PrivateKey,
		Timeout:    req.Options.Global.Timeout,
		RunLS:      opts.RunLS,
	})
	if err != nil {
		return err
	}

	r.Logger.Info("sftp probe",
		"host", host,
		"port", port,
		"server_version", res.ServerVersion,
		"host_key_type", res.Algorithms.HostKey,
		"auth_method", res.AuthMethod,
		"handshake_rtt", res.HandshakeRTT,
		"auth_rtt", res.AuthRTT,
		"ls_entries", len(res.LSEntries),
	)
	summary := res.ServerVersion
	if res.Algorithms.HostKey != "" {
		summary += " (" + res.Algorithms.HostKey + ")"
	}
	req.Report.AddProto(ProtoResult{
		Protocol: "sftp",
		Host:     host,
		Port:     port,
		OK:       true,
		Summary:  summary,
	})
	return nil
}
