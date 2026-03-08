package diag

import "context"

// NotImplementedRunner is a placeholder until a target-specific implementation is available.
type NotImplementedRunner struct {
	target Target
}

// NewNotImplementedRunner returns a runner that signals missing implementation.
func NewNotImplementedRunner(target Target) Runner {
	return &NotImplementedRunner{target: target}
}

// Run returns ErrNotImplemented to indicate the probe is a stub.
func (r *NotImplementedRunner) Run(_ context.Context, _ Request) error {
	return ErrNotImplemented
}
