package diag

import "context"

// MultiRunner runs a sequence of runners and stops on first error.
type MultiRunner struct {
	runners []Runner
}

// NewMultiRunner constructs a MultiRunner.
func NewMultiRunner(runners ...Runner) Runner {
	return &MultiRunner{runners: runners}
}

// Run executes each runner in order.
func (m *MultiRunner) Run(ctx context.Context, req Request) error {
	for _, r := range m.runners {
		if err := r.Run(ctx, req); err != nil {
			return err
		}
	}
	return nil
}
