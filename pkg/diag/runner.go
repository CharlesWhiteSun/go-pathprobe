package diag

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrRunnerNotFound = errors.New("no runner registered for target")
)

// Runner performs a diagnostic for a given target.
type Runner interface {
	Run(ctx context.Context, req Request) error
}

// Request encapsulates the diagnostic target and options.
type Request struct {
	Target  Target
	Options Options
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
