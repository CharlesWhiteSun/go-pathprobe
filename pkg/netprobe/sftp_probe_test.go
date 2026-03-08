package netprobe

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startMockSSHServer starts a minimal in-process SSH server for testing.
// It accepts password authentication with the provided credentials.
// Returns the listener address so the test can connect to it.
func startMockSSHServer(t *testing.T, user, pass string) (host string, port int) {
	t.Helper()

	// Generate an RSA host key for the server.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	serverCfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, inPass []byte) (*ssh.Permissions, error) {
			if c.User() == user && string(inPass) == pass {
				return &ssh.Permissions{}, nil
			}
			return nil, ssh.ErrNoAuth
		},
	}
	serverCfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleMockSSHConn(conn, serverCfg)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

// handleMockSSHConn performs SSH handshake and accepts channel requests.
func handleMockSSHConn(conn net.Conn, cfg *ssh.ServerConfig) {
	defer conn.Close()
	serverConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer serverConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		ch, requests, err := newChan.Accept()
		if err != nil {
			return
		}
		go func(ch ssh.Channel, requests <-chan *ssh.Request) {
			defer ch.Close()
			for req := range requests {
				if req.Type == "subsystem" && len(req.Payload) >= 4 {
					nameLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 |
						int(req.Payload[2])<<8 | int(req.Payload[3])
					if nameLen <= len(req.Payload)-4 {
						_ = string(req.Payload[4 : 4+nameLen])
					}
					// Accept any subsystem (sftp or otherwise).
					if req.WantReply {
						req.Reply(true, nil)
					}
					return
				}
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}(ch, requests)
	}
}

// TestDialSFTPProberMissingHost verifies that an empty host returns an error.
func TestDialSFTPProberMissingHost(t *testing.T) {
	prober := &DialSFTPProber{}
	_, err := prober.Probe(context.Background(), SFTPProbeRequest{})
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

// TestDialSFTPProberPasswordAuth verifies a successful handshake with password auth.
func TestDialSFTPProberPasswordAuth(t *testing.T) {
	host, port := startMockSSHServer(t, "testuser", "testpass")

	prober := &DialSFTPProber{}
	res, err := prober.Probe(context.Background(), SFTPProbeRequest{
		Host:     host,
		Port:     port,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if res.AuthMethod != "password" {
		t.Fatalf("expected auth method 'password', got %q", res.AuthMethod)
	}
	if res.ServerVersion == "" {
		t.Fatalf("expected non-empty server version")
	}
	if res.Algorithms.HostKey == "" {
		t.Fatalf("expected host key algorithm to be recorded")
	}
}

// TestDialSFTPProberWrongPassword verifies a failed authentication returns an error.
func TestDialSFTPProberWrongPassword(t *testing.T) {
	host, port := startMockSSHServer(t, "testuser", "correctpass")

	prober := &DialSFTPProber{}
	_, err := prober.Probe(context.Background(), SFTPProbeRequest{
		Host:     host,
		Port:     port,
		Username: "testuser",
		Password: "wrongpass",
		Timeout:  5 * time.Second,
	})
	if err == nil {
		t.Fatal("expected authentication error")
	}
}

// TestDialSFTPProberRunLS verifies the SFTP subsystem probe path.
func TestDialSFTPProberRunLS(t *testing.T) {
	host, port := startMockSSHServer(t, "sftpuser", "sftppass")

	prober := &DialSFTPProber{}
	res, err := prober.Probe(context.Background(), SFTPProbeRequest{
		Host:     host,
		Port:     port,
		Username: "sftpuser",
		Password: "sftppass",
		RunLS:    true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if len(res.LSEntries) == 0 {
		t.Fatalf("expected at least one LS entry")
	}
}

// TestCaptureHostKeyAlgo verifies the HostKeyCallback records the key type.
func TestCaptureHostKeyAlgo(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubKey, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("new public key: %v", err)
	}

	alg := &SSHAlgorithms{}
	cb := captureHostKeyAlgo(alg)
	if err := cb("addr", nil, pubKey); err != nil {
		t.Fatalf("callback returned error: %v", err)
	}
	if alg.HostKey != "ssh-rsa" {
		t.Fatalf("expected 'ssh-rsa', got %q", alg.HostKey)
	}
}
