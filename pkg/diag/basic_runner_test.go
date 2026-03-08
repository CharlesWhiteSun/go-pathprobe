package diag

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func newTestBasicRunner(target Target) Runner {
	return NewBasicRunner(target, slog.Default())
}

// TestBasicRunnerRunSucceeds verifies nominal execution completes without error.
func TestBasicRunnerRunSucceeds(t *testing.T) {
	runner := newTestBasicRunner(TargetWeb)
	req := Request{
		Target: TargetWeb,
		Options: Options{
			Global: GlobalOptions{MTRCount: 5, Timeout: 5 * time.Second},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestBasicRunnerContextCancelled verifies the runner respects context cancellation.
func TestBasicRunnerContextCancelled(t *testing.T) {
	runner := newTestBasicRunner(TargetIMAP)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the channel is already closed

	err := runner.Run(ctx, Request{Target: TargetIMAP})
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

// TestBasicRunnerExpiredDeadline verifies the deadline guard detects a past deadline.
func TestBasicRunnerExpiredDeadline(t *testing.T) {
	runner := newTestBasicRunner(TargetFTP)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := runner.Run(ctx, Request{Target: TargetFTP})
	if err == nil {
		t.Fatal("expected deadline exceeded error, got nil")
	}
}

// TestBasicRunnerAllTargets ensures NewBasicRunner accepts every registered target without panicking.
func TestBasicRunnerAllTargets(t *testing.T) {
	for _, target := range AllTargets {
		runner := newTestBasicRunner(target)
		req := Request{
			Target:  target,
			Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}},
		}
		if err := runner.Run(context.Background(), req); err != nil {
			t.Fatalf("target %s: unexpected error: %v", target, err)
		}
	}
}
