package server

import (
	"fmt"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/netprobe"
)

// isValidTarget returns true when t is one of the registered diagnostic targets.
func isValidTarget(t diag.Target) bool {
	for _, known := range diag.AllTargets {
		if t == known {
			return true
		}
	}
	return false
}

// parseDiagTimeout parses a Go duration string, falling back to
// defaultDiagTimeout on empty input or parse error.
func parseDiagTimeout(s string) time.Duration {
	if s == "" {
		return defaultDiagTimeout
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return defaultDiagTimeout
	}
	return d
}

// buildOptions converts the API request model to diag.Options.
// Unset fields receive sensible defaults (e.g. MTRCount → DefaultMTRCount).
func buildOptions(r ReqOptions) (diag.Options, error) {
	mtrCount := r.MTRCount
	if mtrCount <= 0 {
		mtrCount = diag.DefaultMTRCount
	}

	opts := diag.Options{
		Global: diag.GlobalOptions{
			MTRCount: mtrCount,
			Timeout:  parseDiagTimeout(r.Timeout),
			Insecure: r.Insecure,
		},
	}

	if w := r.Web; w != nil {
		webMode := diag.WebMode(w.Mode)
		if !diag.IsValidWebMode(webMode) {
			return diag.Options{}, fmt.Errorf("web.mode: unknown mode %q", w.Mode)
		}
		opts.Web.Mode = webMode
		opts.Web.Domains = w.Domains
		opts.Web.URL = w.URL
		opts.Web.MaxHops = w.MaxHops
		if len(w.Types) > 0 {
			types, err := netprobe.ParseRecordTypes(w.Types)
			if err != nil {
				return diag.Options{}, fmt.Errorf("web.types: %w", err)
			}
			opts.Web.Types = types
		}
	}

	if n := r.Net; n != nil {
		opts.Net.Host = n.Host
		opts.Net.Ports = n.Ports
	}

	if s := r.SMTP; s != nil {
		smtpMode := diag.SMTPMode(s.Mode)
		if !diag.IsValidSMTPMode(smtpMode) {
			return diag.Options{}, fmt.Errorf("smtp.mode: unknown mode %q", s.Mode)
		}
		opts.SMTP = diag.SMTPOptions{
			Mode:        smtpMode,
			Domain:      s.Domain,
			Username:    s.Username,
			Password:    s.Password,
			From:        s.From,
			To:          s.To,
			UseTLS:      s.UseTLS,
			StartTLS:    s.StartTLS,
			AuthMethods: s.AuthMethods,
			MXProbeAll:  s.MXProbeAll,
		}
	}

	if f := r.FTP; f != nil {
		ftpMode := diag.FTPMode(f.Mode)
		if !diag.IsValidFTPMode(ftpMode) {
			return diag.Options{}, fmt.Errorf("ftp.mode: unknown mode %q", f.Mode)
		}
		opts.FTP = diag.FTPOptions{
			Mode:     ftpMode,
			Username: f.Username,
			Password: f.Password,
			UseTLS:   f.UseTLS,
			AuthTLS:  f.AuthTLS,
			RunLIST:  f.RunLIST,
		}
	}

	if sf := r.SFTP; sf != nil {
		sftpMode := diag.SFTPMode(sf.Mode)
		if !diag.IsValidSFTPMode(sftpMode) {
			return diag.Options{}, fmt.Errorf("sftp.mode: unknown mode %q", sf.Mode)
		}
		opts.SFTP = diag.SFTPOptions{
			Mode:     sftpMode,
			Username: sf.Username,
			Password: sf.Password,
			RunLS:    sf.RunLS,
			// PrivateKey is intentionally not exposed via the HTTP API.
		}
	}

	return opts, nil
}
