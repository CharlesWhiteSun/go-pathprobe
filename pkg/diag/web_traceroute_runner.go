package diag

import "context"

// WebTracerouteRunner wraps an inner Runner (typically TracerouteRunner) and
// makes it participate in the web-mode single-select model.
//
// It delegates to the inner Runner only when the request's WebMode is either
// WebModeAll (legacy, which runs every sub-diagnostic) or WebModeTraceroute.
// For any other explicit mode it is a no-op.
//
// It also bridges WebOptions.MaxHops into NetworkOptions.MaxHops so that
// callers who set the traceroute depth via the web-mode path get the expected
// behaviour without having to populate the Net sub-options separately.
type WebTracerouteRunner struct{ inner Runner }

// NewWebTracerouteRunner wraps inner so that it only fires for web
// traceroute-mode requests.  inner is typically a *TracerouteRunner.
func NewWebTracerouteRunner(inner Runner) *WebTracerouteRunner {
	return &WebTracerouteRunner{inner: inner}
}

// Run delegates to the inner runner only when the web mode is blank
// (legacy all-in-one) or explicitly "traceroute".
//
// When WebOptions.MaxHops is set and NetworkOptions.MaxHops has not been
// independently configured, the web-level value is forwarded so the inner
// TracerouteRunner does not need to read from two different options paths.
func (r *WebTracerouteRunner) Run(ctx context.Context, req Request) error {
	mode := req.Options.Web.Mode
	if mode != WebModeAll && mode != WebModeTraceroute {
		return nil
	}

	// Bridge web-level MaxHops into network-level when not already set.
	if req.Options.Web.MaxHops > 0 && req.Options.Net.MaxHops == 0 {
		req.Options.Net.MaxHops = req.Options.Web.MaxHops
	}

	return r.inner.Run(ctx, req)
}
