// Package server — white-box tests for diagPipeline.runDiag.
package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/store"
)

// ---- test doubles -------------------------------------------------------

// errRunner is a Runner that always returns a preset error.
type errRunner struct{ err error }

func (r errRunner) Run(_ context.Context, _ diag.Request) error { return r.err }

// okRunner is a Runner that completes successfully without performing any I/O.
type okRunner struct{}

func (okRunner) Run(_ context.Context, _ diag.Request) error { return nil }

// emitRunner emits a single progress event then returns successfully.
type emitRunner struct {
	stage   string
	message string
}

func (r emitRunner) Run(_ context.Context, req diag.Request) error {
	req.Emit(r.stage, r.message)
	return nil
}

// newPipelineWith returns a diagPipeline wired with the given runners map
// (may be nil to simulate "no runners registered").
func newPipelineWith(runners map[diag.Target]diag.Runner) diagPipeline {
	return diagPipeline{
		dispatcher: diag.NewDispatcher(runners),
		locator:    geo.NoopLocator{},
		store:      store.NewMemoryStore(10),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// ---- runDiag tests -------------------------------------------------------

// TestDiagPipeline_RunDiag_UnknownTarget_Returns400 verifies that an
// unrecognised target yields a pipelineError with HTTP 400.
func TestDiagPipeline_RunDiag_UnknownTarget_Returns400(t *testing.T) {
	p := newPipelineWith(nil)
	_, err := p.runDiag(context.Background(), DiagRequest{Target: "bogus"}, nil)
	var pe *pipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("want *pipelineError, got %T: %v", err, err)
	}
	if pe.code != http.StatusBadRequest {
		t.Errorf("code = %d, want %d", pe.code, http.StatusBadRequest)
	}
	if !strings.Contains(pe.msg, "bogus") {
		t.Errorf("msg = %q, should contain the unknown target name", pe.msg)
	}
}

// TestDiagPipeline_RunDiag_RunnerNotFound_Returns404 verifies that when no
// runner is registered for an otherwise valid target the result is a 404.
func TestDiagPipeline_RunDiag_RunnerNotFound_Returns404(t *testing.T) {
	p := newPipelineWith(nil) // empty dispatcher → ErrRunnerNotFound for any target
	req := DiagRequest{Target: string(diag.TargetSMTP)}
	_, err := p.runDiag(context.Background(), req, nil)
	var pe *pipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("want *pipelineError, got %T: %v", err, err)
	}
	if pe.code != http.StatusNotFound {
		t.Errorf("code = %d, want %d", pe.code, http.StatusNotFound)
	}
}

// TestDiagPipeline_RunDiag_DispatchError_Returns500 verifies that a runner
// returning a non-ErrRunnerNotFound error produces a 500 pipelineError whose
// message begins with the fmtDiagError prefix.
func TestDiagPipeline_RunDiag_DispatchError_Returns500(t *testing.T) {
	simErr := errors.New("simulated network failure")
	p := newPipelineWith(map[diag.Target]diag.Runner{
		diag.TargetSMTP: errRunner{err: simErr},
	})
	req := DiagRequest{Target: string(diag.TargetSMTP)}
	_, err := p.runDiag(context.Background(), req, nil)
	var pe *pipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("want *pipelineError, got %T: %v", err, err)
	}
	if pe.code != http.StatusInternalServerError {
		t.Errorf("code = %d, want %d", pe.code, http.StatusInternalServerError)
	}
	if !strings.HasPrefix(pe.msg, "diagnostic error: ") {
		t.Errorf("msg = %q, want prefix \"diagnostic error: \"", pe.msg)
	}
}

// TestDiagPipeline_RunDiag_Success_ReturnsReportAndPersists verifies that a
// successful run returns a non-nil AnnotatedReport and saves one entry to the
// history store.
func TestDiagPipeline_RunDiag_Success_ReturnsReportAndPersists(t *testing.T) {
	st := store.NewMemoryStore(10)
	p := diagPipeline{
		dispatcher: diag.NewDispatcher(map[diag.Target]diag.Runner{
			diag.TargetSMTP: okRunner{},
		}),
		locator: geo.NoopLocator{},
		store:   st,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := DiagRequest{Target: string(diag.TargetSMTP)}
	ar, err := p.runDiag(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ar == nil {
		t.Fatal("want non-nil *report.AnnotatedReport")
	}
	if entries := st.List(); len(entries) != 1 {
		t.Errorf("store has %d entries after successful run, want 1", len(entries))
	}
}

// TestDiagPipeline_RunDiag_HookCalledForProgress verifies that ProgressEvents
// emitted by the runner are forwarded to the hook parameter.
func TestDiagPipeline_RunDiag_HookCalledForProgress(t *testing.T) {
	p := newPipelineWith(map[diag.Target]diag.Runner{
		diag.TargetSMTP: emitRunner{stage: "test-stage", message: "hello"},
	})
	var events []diag.ProgressEvent
	hook := func(ev diag.ProgressEvent) { events = append(events, ev) }
	req := DiagRequest{Target: string(diag.TargetSMTP)}
	if _, err := p.runDiag(context.Background(), req, hook); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d progress events, want 1", len(events))
	}
	if events[0].Stage != "test-stage" {
		t.Errorf("stage = %q, want \"test-stage\"", events[0].Stage)
	}
}
