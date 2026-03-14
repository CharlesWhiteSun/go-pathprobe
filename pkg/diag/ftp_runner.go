package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// FTPMode controls whether the FTP runner attempts a directory listing.
type FTPMode string

const (
	FTPModeAll   FTPMode = ""      // legacy: follow RunLIST flag in options
	FTPModeLogin FTPMode = "login" // connect + login, no listing
	FTPModeList  FTPMode = "list"  // connect + login + PASV/LIST
)

// IsValidFTPMode reports whether m is a recognised FTP sub-mode value.
func IsValidFTPMode(m FTPMode) bool {
	switch m {
	case FTPModeAll, FTPModeLogin, FTPModeList:
		return true
	}
	return false
}

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
// Mode controls the listing behaviour:
//   - FTPModeLogin: RunLIST is forced false regardless of options.
//   - FTPModeList: RunLIST is forced true regardless of options.
//   - FTPModeAll (legacy ""): RunLIST is taken from options as-provided.
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

	// Apply mode override for RunLIST.
	switch opts.Mode {
	case FTPModeLogin:
		opts.RunLIST = false
	case FTPModeList:
		opts.RunLIST = true
	}

	req.Emitf("ftp", "Connecting to FTP %s:%d …", host, port)

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

	req.Emitf("ftp_result", "%s:%d: login=%v banner=%q", host, port, res.LoginAccepted, res.Banner)
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
