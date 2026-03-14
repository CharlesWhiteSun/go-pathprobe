package diag

import "context"

// WebPortRunner wraps an inner Runner (typically the connectivity/port runner)
// and makes it participate in the web-mode single-select model.
//
// It delegates to the inner Runner only when the request's WebMode is either
// WebModeAll (legacy, which runs every sub-diagnostic) or WebModePort.
// For any other explicit mode it is a no-op.
type WebPortRunner struct{ inner Runner }

// NewWebPortRunner wraps inner so that it only fires for web port-mode
// requests.  inner is typically the shared connect-runner.
func NewWebPortRunner(inner Runner) *WebPortRunner {
	return &WebPortRunner{inner: inner}
}

// Run delegates to the inner runner only when the web mode is blank
// (legacy all-in-one) or explicitly "port".
func (r *WebPortRunner) Run(ctx context.Context, req Request) error {
	mode := req.Options.Web.Mode
	if mode != WebModeAll && mode != WebModePort {
		return nil
	}
	return r.inner.Run(ctx, req)
}
