package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// FTPRunner performs FTP/FTPS control-channel probe and optional directory listing.
type FTPRunner struct {
	Prober netprobe.FTPProber
	Logger *slog.Logger
}

// NewFTPRunner constructs an FTPRunner.
func NewFTPRunner(prober netprobe.FTPProber, logger *slog.Logger) *FTPRunner {
	return &FTPRunner{Prober: prober, Logger: logger}
}

// Run executes the FTP/FTPS probe based on target options.
func (r *FTPRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}

	host := req.Options.Net.Host
	if host == "" {
		host = "localhost"
	}
	port := choosePort(req.Options.Net.Ports, TargetFTP)
	opts := req.Options.FTP

	res, err := r.Prober.Probe(ctx, netprobe.FTPProbeRequest{
		Host:     host,
		Port:     port,
		Username: opts.Username,
		Password: opts.Password,
		UseTLS:   opts.UseTLS,
		AuthTLS:  opts.AuthTLS,
		Insecure: req.Options.Global.Insecure,
		Timeout:  req.Options.Global.Timeout,
		RunLIST:  opts.RunLIST,
	})
	if err != nil {
		return err
	}

	r.Logger.Info("ftp probe",
		"host", host,
		"port", port,
		"banner", res.Banner,
		"auth_tls", res.UsedAuthTLS,
		"implicit_tls", res.UsedImplicitTLS,
		"login_accepted", res.LoginAccepted,
		"list_entries", len(res.ListEntries),
		"rtt", res.RTT,
	)
	summary := res.Banner
	if res.UsedAuthTLS {
		summary += " [AUTH TLS]"
	} else if res.UsedImplicitTLS {
		summary += " [Implicit TLS]"
	}
	req.Report.AddProto(ProtoResult{
		Protocol: "ftp",
		Host:     host,
		Port:     port,
		OK:       res.LoginAccepted || res.Banner != "",
		Summary:  summary,
	})
	return nil
}
