package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/report"
	"go-pathprobe/pkg/store"
)

// pipelineError is returned by runDiag for request-validation and pipeline
// failures.  The HTTP status code allows handlers to translate it directly
// into the correct response without re-inspecting the message text.
type pipelineError struct {
	code int
	msg  string
}

func (e *pipelineError) Error() string { return e.msg }

// diagPipeline encapsulates the shared dependencies for the diagnostic request
// pipeline.  Both DiagHandler and StreamDiagHandler embed this struct to avoid
// duplicating request validation, dispatch, and report-build logic.
type diagPipeline struct {
	dispatcher *diag.Dispatcher
	locator    geo.Locator
	store      store.Store
	logger     *slog.Logger
}

// runDiag validates the request, dispatches the diagnostic, builds and stores
// the AnnotatedReport, and returns it.  parentCtx is typically r.Context()
// (the incoming HTTP request context); runDiag applies the computed timeout on
// top of it and handles cancellation internally.
//
// hook is an optional ProgressHook injected by the streaming handler to relay
// real-time events as SSE messages.  Non-streaming callers should pass nil.
//
// All errors are returned as *pipelineError with an appropriate HTTP status
// code so that handlers can translate a single error type into the correct
// HTTP response.
func (p *diagPipeline) runDiag(parentCtx context.Context, req DiagRequest, hook diag.ProgressHook) (*report.AnnotatedReport, error) {
	// 1. Target validation.
	target := diag.Target(req.Target)
	if !isValidTarget(target) {
		return nil, &pipelineError{code: http.StatusBadRequest, msg: "unknown target: " + req.Target}
	}

	// 2. Options parsing and global validation.
	opts, err := buildOptions(req.Options)
	if err != nil {
		return nil, &pipelineError{code: http.StatusBadRequest, msg: "invalid options: " + err.Error()}
	}
	if err := opts.Global.Validate(); err != nil {
		return nil, &pipelineError{code: http.StatusBadRequest, msg: err.Error()}
	}

	// 3. Dispatch with timeout-adjusted context.
	diagReport := &diag.DiagReport{Target: target, Host: opts.Net.Host}
	timeout := ensureTracerouteTimeout(parseDiagTimeout(req.Options.Timeout), opts)
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	dreq := diag.Request{Target: target, Options: opts, Report: diagReport, Hook: hook}
	if err := p.dispatcher.Dispatch(ctx, dreq); err != nil {
		if errors.Is(err, diag.ErrRunnerNotFound) {
			return nil, &pipelineError{code: http.StatusNotFound, msg: "no runner registered for target: " + req.Target}
		}
		p.logger.Warn("diagnostic failed",
			"target", target,
			"host", opts.Net.Host,
			"mode", string(opts.Web.Mode),
			"error", err)
		return nil, &pipelineError{code: http.StatusInternalServerError, msg: fmtDiagError(err, opts)}
	}

	// 4. Geo annotation and report build.
	locator := p.resolveLocator(req.Options.DisableGeo)
	ar, err := report.Build(ctx, diagReport, locator)
	if err != nil {
		return nil, &pipelineError{code: http.StatusInternalServerError, msg: "report build failed: " + err.Error()}
	}

	// 5. Persist to history store.
	p.store.Save(store.HistoryEntry{Report: ar})
	return ar, nil
}

// resolveLocator returns the pipeline's configured locator, or a NoopLocator
// when the caller has opted out of geo annotation for this request.
func (p *diagPipeline) resolveLocator(disableGeo bool) geo.Locator {
	if disableGeo {
		return geo.NoopLocator{}
	}
	return p.locator
}
