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
