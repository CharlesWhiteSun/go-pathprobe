package diag

import (
	"context"
	"errors"
	"testing"
)

type dummyRunner struct {
	err   error
	calls int
}

func (d *dummyRunner) Run(ctx context.Context, req Request) error {
	d.calls++
	return d.err
}

// TestMultiRunnerStopsOnError ensures sequential execution halts on first error.
func TestMultiRunnerStopsOnError(t *testing.T) {
	r1 := &dummyRunner{}
	r2 := &dummyRunner{err: errors.New("boom")}
	r3 := &dummyRunner{}

	mr := NewMultiRunner(r1, r2, r3)
	err := mr.Run(context.Background(), Request{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if r1.calls != 1 || r2.calls != 1 || r3.calls != 0 {
		t.Fatalf("unexpected call counts %d %d %d", r1.calls, r2.calls, r3.calls)
	}
}

// TestMultiRunnerAllRun when no errors occur.
func TestMultiRunnerAllRun(t *testing.T) {
	r1 := &dummyRunner{}
	r2 := &dummyRunner{}
	mr := NewMultiRunner(r1, r2)
	if err := mr.Run(context.Background(), Request{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r1.calls != 1 || r2.calls != 1 {
		t.Fatalf("expected both runners to execute")
	}
}

// ctxAwareRunner returns ctx.Err() when the context is already cancelled.
type ctxAwareRunner struct{ calls int }

func (r *ctxAwareRunner) Run(ctx context.Context, _ Request) error {
	r.calls++
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

// TestMultiRunnerContextCancel verifies context cancellation stops further runner execution.
func TestMultiRunnerContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r1 := &ctxAwareRunner{}
	r2 := &ctxAwareRunner{}
	mr := NewMultiRunner(r1, r2)
	err := mr.Run(ctx, Request{})
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
	if r1.calls != 1 {
		t.Fatalf("expected r1 to run once, got %d", r1.calls)
	}
	if r2.calls != 0 {
		t.Fatalf("expected r2 not to run, got %d", r2.calls)
	}
}
