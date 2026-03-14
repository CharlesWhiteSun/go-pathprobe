package diag

import (
	"context"
	"testing"
	"time"
)

// trackingRunner records whether Run was called.
type trackingRunner struct{ called bool }

func (r *trackingRunner) Run(_ context.Context, _ Request) error {
	r.called = true
	return nil
}

func webPortRequest(mode WebMode) Request {
	return Request{
		Target:  TargetWeb,
		Options: Options{Global: GlobalOptions{MTRCount: 1, Timeout: time.Second}, Web: WebOptions{Mode: mode}},
	}
}

// TestWebPortRunnerDelegatesOnPortMode verifies delegation when mode is "port".
func TestWebPortRunnerDelegatesOnPortMode(t *testing.T) {
	inner := &trackingRunner{}
	r := NewWebPortRunner(inner)
	if err := r.Run(context.Background(), webPortRequest(WebModePort)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.called {
		t.Fatal("inner runner must be called in port mode")
	}
}

// TestWebPortRunnerDelegatesOnLegacyMode verifies delegation when mode is "" (all).
func TestWebPortRunnerDelegatesOnLegacyMode(t *testing.T) {
	inner := &trackingRunner{}
	r := NewWebPortRunner(inner)
	if err := r.Run(context.Background(), webPortRequest(WebModeAll)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.called {
		t.Fatal("inner runner must be called in legacy (all) mode")
	}
}

// TestWebPortRunnerNoOpOnOtherModes verifies the runner is silent for public-ip/dns/http.
func TestWebPortRunnerNoOpOnOtherModes(t *testing.T) {
	for _, mode := range []WebMode{WebModePublicIP, WebModeDNS, WebModeHTTP} {
		inner := &trackingRunner{}
		r := NewWebPortRunner(inner)
		if err := r.Run(context.Background(), webPortRequest(mode)); err != nil {
			t.Fatalf("mode=%q: unexpected error %v", mode, err)
		}
		if inner.called {
			t.Fatalf("mode=%q: inner runner must NOT be called", mode)
		}
	}
}
