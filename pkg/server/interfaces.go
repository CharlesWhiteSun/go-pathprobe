package server

import (
	"context"

	"go-pathprobe/pkg/diag"
)

// Dispatcher can dispatch a diagnostic request to the registered runner.
// *diag.Dispatcher satisfies this interface, enabling test doubles.
type Dispatcher interface {
	Dispatch(ctx context.Context, req diag.Request) error
}

// OptionsBuilder converts a server API request into fully-typed diag.Options
// for the given diagnostic target.  By default the server uses its built-in
// buildOptions function; protocol plugins may supply a custom implementation
// via WithOptionsBuilder so that adding a new protocol requires no changes to
// the server package (Open/Closed Principle).
type OptionsBuilder func(target diag.Target, req ReqOptions) (diag.Options, error)
