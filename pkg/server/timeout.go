package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-pathprobe/pkg/diag"
)

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
