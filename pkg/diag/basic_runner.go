package diag

import (
	"context"
	"log/slog"
	"time"
)

// BasicRunner logs a diagnostic start/end and can be extended with protocol-specific probes.
type BasicRunner struct {
	target Target
	logger *slog.Logger
}

// NewBasicRunner constructs a simple runner for a target.
func NewBasicRunner(target Target, logger *slog.Logger) Runner {
	return &BasicRunner{target: target, logger: logger}
}

// Run performs minimal bookkeeping; concrete probes can replace this in later milestones.
func (r *BasicRunner) Run(ctx context.Context, req Request) error {
	// Respect cancellation and timeout from context; in future, protocol probes will live here.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.logger.Info("diagnostic start", "target", r.target, "timeout", req.Options.Global.Timeout, "insecure", req.Options.Global.Insecure)

	// Simulate lightweight work respecting timeout for future extension.
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.DeadlineExceeded
		}
	}

	r.logger.Info("diagnostic end", "target", r.target)
	return nil
}
