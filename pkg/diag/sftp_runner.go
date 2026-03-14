package diag

import (
	"context"
	"log/slog"

	"go-pathprobe/pkg/netprobe"
)

// SFTPMode controls whether the SFTP runner attempts a directory listing.
type SFTPMode string

const (
	SFTPModeAll  SFTPMode = ""     // legacy: follow RunLS flag in options
	SFTPModeAuth SFTPMode = "auth" // SSH handshake + authentication only
	SFTPModeLS   SFTPMode = "ls"   // SSH + auth + list remote default directory
)

// IsValidSFTPMode reports whether m is a recognised SFTP sub-mode value.
func IsValidSFTPMode(m SFTPMode) bool {
	switch m {
	case SFTPModeAll, SFTPModeAuth, SFTPModeLS:
		return true
	}
	return false
}

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
// Mode controls the listing behaviour:
//   - SFTPModeAuth: RunLS is forced false.
//   - SFTPModeLS: RunLS is forced true.
//   - SFTPModeAll (legacy ""): RunLS is taken from options as-provided.
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

	// Apply mode override for RunLS.
	switch opts.Mode {
	case SFTPModeAuth:
		opts.RunLS = false
	case SFTPModeLS:
		opts.RunLS = true
	}

	req.Emitf("sftp", "Connecting to SFTP %s:%d …", host, port)

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

	req.Emitf("sftp_result", "%s:%d: %s (%s) auth=%s",
		host, port, res.ServerVersion, res.Algorithms.HostKey, res.AuthMethod)
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
