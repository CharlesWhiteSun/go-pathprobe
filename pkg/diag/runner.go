package diag

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go-pathprobe/pkg/netprobe"
)

var (
	ErrRunnerNotFound = errors.New("no runner registered for target")
)

// Runner performs a diagnostic for a given target.
type Runner interface {
	Run(ctx context.Context, req Request) error
}

// ReportWriter is the minimal write-only interface for accumulating structured
// diagnostic results.  *DiagReport satisfies this interface.
// Runners accept a ReportWriter so they can be tested with any implementation.
type ReportWriter interface {
	AddProto(ProtoResult)
	AddPorts([]netprobe.PortProbeResult)
	SetRoute(*netprobe.RouteResult)
	SetPublicIP(string)
}

// Request encapsulates the diagnostic target and options.
type Request struct {
	Target  Target
	Options Options
	// Report is an optional accumulator for structured results.
	// Runners write to it when non-nil; a nil interface disables reporting.
	Report ReportWriter
	// Hook is an optional callback for real-time progress updates.
	// A nil Hook silently disables all progress emissions.
	Hook ProgressHook
}

// Emit calls the progress hook (if non-nil) with a plain-text message.
// It is a no-op when Hook is nil.
func (r Request) Emit(stage, message string) {
	if r.Hook != nil {
		r.Hook(ProgressEvent{Stage: stage, Message: message})
	}
}

// Emitf calls the progress hook (if non-nil) with a formatted message.
// It is a no-op when Hook is nil.
func (r Request) Emitf(stage, format string, args ...any) {
	if r.Hook != nil {
		r.Hook(ProgressEvent{Stage: stage, Message: fmt.Sprintf(format, args...)})
	}
}

// Dispatcher keeps target-to-runner mappings and delegates execution.
type Dispatcher struct {
	mu      sync.RWMutex
	runners map[Target]Runner
}

// NewDispatcher builds a dispatcher with an optional initial runner registry.
func NewDispatcher(initial map[Target]Runner) *Dispatcher {
	registry := make(map[Target]Runner)
	for k, v := range initial {
		registry[k] = v
	}
	return &Dispatcher{runners: registry}
}

// Register binds a runner to a target in a threadsafe manner.
func (d *Dispatcher) Register(target Target, runner Runner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.runners[target] = runner
}

// Dispatch locates the runner for the request target and executes it.
func (d *Dispatcher) Dispatch(ctx context.Context, req Request) error {
	d.mu.RLock()
	runner, ok := d.runners[req.Target]
	d.mu.RUnlock()
	if !ok {
		return ErrRunnerNotFound
	}
	return runner.Run(ctx, req)
}
