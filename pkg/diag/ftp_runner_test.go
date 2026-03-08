package diag

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"go-pathprobe/pkg/netprobe"
)

// stubFTPProber is a minimal stub that records the last probe request.
type stubFTPProber struct {
	called      bool
	lastRequest netprobe.FTPProbeRequest
	result      netprobe.FTPProbeResult
	err         error
}

func (s *stubFTPProber) Probe(_ context.Context, req netprobe.FTPProbeRequest) (netprobe.FTPProbeResult, error) {
	s.called = true
	s.lastRequest = req
	return s.result, s.err
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestFTPRunnerNilProber ensures ErrRunnerNotFound is returned when the prober is nil.
func TestFTPRunnerNilProber(t *testing.T) {
	runner := &FTPRunner{Logger: newDiscardLogger()}
	err := runner.Run(context.Background(), Request{Target: TargetFTP})
	if !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("expected ErrRunnerNotFound, got %v", err)
	}
}

// TestFTPRunnerDefaultHost verifies that an empty host defaults to "localhost".
func TestFTPRunnerDefaultHost(t *testing.T) {
	stub := &stubFTPProber{result: netprobe.FTPProbeResult{}}
	runner := NewFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastRequest.Host != "localhost" {
		t.Fatalf("expected host 'localhost', got %q", stub.lastRequest.Host)
	}
}

// TestFTPRunnerExplicitHost verifies the host from NetworkOptions is forwarded.
func TestFTPRunnerExplicitHost(t *testing.T) {
	stub := &stubFTPProber{}
	runner := NewFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "ftp.example.com", Ports: []int{2121}},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastRequest.Host != "ftp.example.com" {
		t.Fatalf("expected explicit host, got %q", stub.lastRequest.Host)
	}
	if stub.lastRequest.Port != 2121 {
		t.Fatalf("expected port 2121, got %d", stub.lastRequest.Port)
	}
}

// TestFTPRunnerDefaultPort verifies the FTP default port (21) is used when none is specified.
func TestFTPRunnerDefaultPort(t *testing.T) {
	stub := &stubFTPProber{}
	runner := NewFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "ftp.test"},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastRequest.Port != 21 {
		t.Fatalf("expected default port 21, got %d", stub.lastRequest.Port)
	}
}

// TestFTPRunnerOptionsPropagated verifies all FTPOptions fields reach the prober.
func TestFTPRunnerOptionsPropagated(t *testing.T) {
	stub := &stubFTPProber{}
	runner := NewFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: 5 * time.Second, Insecure: true},
			Net:    NetworkOptions{Host: "ftp.test"},
			FTP: FTPOptions{
				Username: "ftpuser",
				Password: "ftppass",
				UseTLS:   true,
				AuthTLS:  true,
				RunLIST:  true,
			},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := stub.lastRequest
	if r.Username != "ftpuser" || r.Password != "ftppass" {
		t.Fatalf("credentials not propagated: user=%q pass=%q", r.Username, r.Password)
	}
	if !r.UseTLS || !r.AuthTLS || !r.RunLIST {
		t.Fatalf("boolean options not propagated: UseTLS=%v AuthTLS=%v RunLIST=%v", r.UseTLS, r.AuthTLS, r.RunLIST)
	}
	if !r.Insecure {
		t.Fatalf("expected Insecure flag propagated")
	}
	if r.Timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", r.Timeout)
	}
}

// TestFTPRunnerProberError verifies that a prober error is returned to the caller.
func TestFTPRunnerProberError(t *testing.T) {
	expectedErr := errors.New("connection refused")
	stub := &stubFTPProber{err: expectedErr}
	runner := NewFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "ftp.test"},
		},
	}
	err := runner.Run(context.Background(), req)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected prober error, got %v", err)
	}
}
