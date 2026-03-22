// Package server — tests for the Dispatcher interface.
package server

import (
	"context"
	"errors"
	"testing"

	"go-pathprobe/pkg/diag"
)

// Compile-time assertion: *diag.Dispatcher satisfies server.Dispatcher.
var _ Dispatcher = (*diag.Dispatcher)(nil)

// ---- mock Dispatcher test double -----------------------------------------

// mockDispatcher records the requests it receives and returns a preset error.
type mockDispatcher struct {
	calls []diag.Request
	err   error
}

func (m *mockDispatcher) Dispatch(_ context.Context, req diag.Request) error {
	m.calls = append(m.calls, req)
	return m.err
}

// ---- tests ----------------------------------------------------------------

func TestDispatcherInterface_MockSatisfiesInterface(t *testing.T) {
	// Verify *mockDispatcher satisfies the Dispatcher interface via type assertion.
	var d Dispatcher = &mockDispatcher{}
	if _, ok := d.(*mockDispatcher); !ok {
		t.Fatal("expected *mockDispatcher to satisfy Dispatcher interface")
	}
}

func TestDispatcherInterface_ForwardsRequest(t *testing.T) {
	mock := &mockDispatcher{}
	req := diag.Request{Target: diag.TargetWeb}

	if err := mock.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	if mock.calls[0].Target != diag.TargetWeb {
		t.Errorf("expected target %q, got %q", diag.TargetWeb, mock.calls[0].Target)
	}
}

func TestDispatcherInterface_PropagatesError(t *testing.T) {
	sentinel := errors.New("dispatch error")
	mock := &mockDispatcher{err: sentinel}

	err := mock.Dispatch(context.Background(), diag.Request{})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}
