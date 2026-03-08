package diag

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-pathprobe/pkg/netprobe"
)

// stubSFTPProber is a minimal stub that records the last probe request.
type stubSFTPProber struct {
	called      bool
	lastRequest netprobe.SFTPProbeRequest
	result      netprobe.SFTPProbeResult
	err         error
}

func (s *stubSFTPProber) Probe(_ context.Context, req netprobe.SFTPProbeRequest) (netprobe.SFTPProbeResult, error) {
	s.called = true
	s.lastRequest = req
	return s.result, s.err
}

// TestSFTPRunnerNilProber ensures ErrRunnerNotFound is returned when the prober is nil.
func TestSFTPRunnerNilProber(t *testing.T) {
	runner := &SFTPRunner{Logger: newDiscardLogger()}
	err := runner.Run(context.Background(), Request{Target: TargetSFTP})
	if !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("expected ErrRunnerNotFound, got %v", err)
	}
}

// TestSFTPRunnerDefaultHost verifies that an empty host defaults to "localhost".
func TestSFTPRunnerDefaultHost(t *testing.T) {
	stub := &stubSFTPProber{}
	runner := NewSFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetSFTP,
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

// TestSFTPRunnerDefaultPort verifies the SFTP default port (22) is used when none is specified.
func TestSFTPRunnerDefaultPort(t *testing.T) {
	stub := &stubSFTPProber{}
	runner := NewSFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetSFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "sftp.test"},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastRequest.Port != 22 {
		t.Fatalf("expected default port 22, got %d", stub.lastRequest.Port)
	}
}

// TestSFTPRunnerExplicitPort verifies a custom port is forwarded.
func TestSFTPRunnerExplicitPort(t *testing.T) {
	stub := &stubSFTPProber{}
	runner := NewSFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetSFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "sftp.test", Ports: []int{2222}},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastRequest.Port != 2222 {
		t.Fatalf("expected port 2222, got %d", stub.lastRequest.Port)
	}
}

// TestSFTPRunnerOptionsPropagated verifies all SFTPOptions fields reach the prober.
func TestSFTPRunnerOptionsPropagated(t *testing.T) {
	stub := &stubSFTPProber{}
	runner := NewSFTPRunner(stub, newDiscardLogger())

	key := []byte("PEM data placeholder")
	req := Request{
		Target: TargetSFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: 5 * time.Second},
			Net:    NetworkOptions{Host: "sftp.test"},
			SFTP: SFTPOptions{
				Username:   "sftpuser",
				Password:   "sftppass",
				PrivateKey: key,
				RunLS:      true,
			},
		},
	}
	if err := runner.Run(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := stub.lastRequest
	if r.Username != "sftpuser" || r.Password != "sftppass" {
		t.Fatalf("credentials not propagated: user=%q pass=%q", r.Username, r.Password)
	}
	if string(r.PrivateKey) != string(key) {
		t.Fatalf("private key not propagated")
	}
	if !r.RunLS {
		t.Fatalf("expected RunLS propagated")
	}
	if r.Timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", r.Timeout)
	}
}

// TestSFTPRunnerProberError verifies a prober error is returned to the caller.
func TestSFTPRunnerProberError(t *testing.T) {
	expectedErr := errors.New("ssh: handshake failed")
	stub := &stubSFTPProber{err: expectedErr}
	runner := NewSFTPRunner(stub, newDiscardLogger())

	req := Request{
		Target: TargetSFTP,
		Options: Options{
			Global: GlobalOptions{Timeout: time.Second},
			Net:    NetworkOptions{Host: "sftp.test"},
		},
	}
	err := runner.Run(context.Background(), req)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected prober error, got %v", err)
	}
}
