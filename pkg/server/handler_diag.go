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

	timeout := ensureTracerouteTimeout(parseDiagTimeout(req.Options.Timeout), opts)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	dreq := diag.Request{Target: target, Options: opts, Report: diagReport}
	if err := h.dispatcher.Dispatch(ctx, dreq); err != nil {
		if errors.Is(err, diag.ErrRunnerNotFound) {
			writeError(w, http.StatusNotFound, "no runner registered for target: "+req.Target)
			return
		}
		h.logger.Warn("diagnostic failed",
			"target", target,
			"host", opts.Net.Host,
			"mode", string(opts.Web.Mode),
			"error", err)
		writeError(w, http.StatusInternalServerError, fmtDiagError(err, opts))
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

// tracerouteMinTimeout returns the minimum context timeout needed to complete a
// traceroute with the given options without a context deadline exceeded error.
//
// Formula: maxHops × mtrCount × 2 s (per-hop ceiling from netprobe.hopTimeout)
// plus a 15 s overhead buffer for DNS resolution and TCP handshake latency.
// Returns 0 when opts does not describe a traceroute request.
func tracerouteMinTimeout(opts diag.Options) time.Duration {
	if opts.Web.Mode != diag.WebModeTraceroute {
		return 0
	}
	maxHops := opts.Web.MaxHops
	if maxHops <= 0 {
		maxHops = diag.DefaultMaxHops
	}
	mtrCount := opts.Global.MTRCount
	if mtrCount <= 0 {
		mtrCount = diag.DefaultMTRCount
	}
	// 2 s matches netprobe.hopTimeout (package-level constant, not exported).
	const perHopSec = 2
	const bufferSec = 15
	return time.Duration(maxHops*mtrCount*perHopSec)*time.Second + bufferSec*time.Second
}

// ensureTracerouteTimeout returns t unchanged when t already exceeds the
// minimum timeout required for the traceroute configuration. If t is shorter
// than the computed minimum, the minimum is returned instead so the context
// does not expire before all hops are probed.
func ensureTracerouteTimeout(t time.Duration, opts diag.Options) time.Duration {
	if min := tracerouteMinTimeout(opts); min > 0 && t < min {
		return min
	}
	return t
}

// fmtDiagError converts a raw runner error into a human-readable description
// suitable for display in a UI error banner and structured log output.
// It recognises common failure modes (deadline, DNS, connection) and replaces
// opaque Go error strings with plain-language alternatives.
// Always log the original error separately for debugging.
func fmtDiagError(err error, opts diag.Options) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		if opts.Web.Mode == diag.WebModeTraceroute {
			maxHops := opts.Web.MaxHops
			if maxHops <= 0 {
				maxHops = diag.DefaultMaxHops
			}
			return fmt.Sprintf(
				"traceroute timed out (max %d hops) — try increasing Timeout in Advanced Options or reducing Max Hops",
				maxHops)
		}
		return "diagnostic timed out — try increasing Timeout in Advanced Options"
	}
	return "diagnostic error: " + err.Error()
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
