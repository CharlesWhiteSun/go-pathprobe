package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/netprobe"
	"go-pathprobe/pkg/report"
	"go-pathprobe/pkg/store"
)

// DiagHandler serves POST /api/diag.
// It decodes the request, maps it to diag.Options, dispatches to the registered
// Runner, then builds and returns a JSON-encoded AnnotatedReport.
type DiagHandler struct {
	dispatcher *diag.Dispatcher
	locator    geo.Locator
	store      store.Store
	logger     *slog.Logger
}

// ServeHTTP handles POST /api/diag.
func (h *DiagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Guard against excessively large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req DiagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	target := diag.Target(req.Target)
	if !isValidTarget(target) {
		writeError(w, http.StatusBadRequest, "unknown target: "+req.Target)
		return
	}

	opts, err := buildOptions(req.Options)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid options: "+err.Error())
		return
	}

	if err := opts.Global.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	diagReport := &diag.DiagReport{Target: target, Host: opts.Net.Host}

	timeout := parseDiagTimeout(req.Options.Timeout)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	dreq := diag.Request{Target: target, Options: opts, Report: diagReport}
	if err := h.dispatcher.Dispatch(ctx, dreq); err != nil {
		if errors.Is(err, diag.ErrRunnerNotFound) {
			writeError(w, http.StatusNotFound, "no runner registered for target: "+req.Target)
			return
		}
		h.logger.Warn("diagnostic failed", "target", target, "error", err)
		writeError(w, http.StatusInternalServerError, "diagnostic error: "+err.Error())
		return
	}

	locator := h.resolveLocator(req.Options.DisableGeo)
	ar, err := report.Build(ctx, diagReport, locator)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "report build failed: "+err.Error())
		return
	}

	h.store.Save(store.HistoryEntry{Report: ar})

	writeJSON(w, http.StatusOK, ar)
}

// resolveLocator returns the handler's locator normally, or a NoopLocator when
// the caller has opted out of geo annotation for this request.
func (h *DiagHandler) resolveLocator(disableGeo bool) geo.Locator {
	if disableGeo {
		return geo.NoopLocator{}
	}
	return h.locator
}

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
