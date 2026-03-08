package diag

import (
	"errors"
	"log/slog"
)

const (
	// DefaultMTRCount specifies default probe count per hop for traceroute-style diagnostics.
	DefaultMTRCount = 5
)

// GlobalOptions captures flags shared across diagnostic targets.
type GlobalOptions struct {
	JSON     bool
	Report   string
	MTRCount int
	LogLevel slog.Level
}

// Validate ensures the option values are within acceptable ranges.
func (o GlobalOptions) Validate() error {
	if o.MTRCount <= 0 {
		return errors.New("mtr-count must be greater than zero")
	}
	return nil
}

// Options bundles global options with target-specific placeholders for future extension.
type Options struct {
	Global GlobalOptions
}
