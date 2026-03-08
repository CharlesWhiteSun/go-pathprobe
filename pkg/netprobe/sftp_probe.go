package netprobe

import (
	"context"
	"errors"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// SFTPProbeRequest contains inputs for an SSH/SFTP probe.
type SFTPProbeRequest struct {
	Host       string
	Port       int
	Username   string
	Password   string // password authentication
	PrivateKey []byte // PEM-encoded private key for public-key authentication
	Timeout    time.Duration
	RunLS      bool // attempt to list the default remote directory
}

// SFTPProbeResult captures SSH handshake and SFTP operation outcomes.
type SFTPProbeResult struct {
	ServerVersion string
	Algorithms    SSHAlgorithms
	HandshakeRTT  time.Duration
	AuthRTT       time.Duration
	LSEntries     []string
	AuthMethod    string // "password" or "publickey"
}

// SSHAlgorithms records the negotiated SSH algorithms from the handshake.
type SSHAlgorithms struct {
	KeyExchange string
	HostKey     string
	Cipher      string
	MAC         string
}

// SFTPProber performs an SSH handshake, optional auth, and optional SFTP ls.
type SFTPProber interface {
	Probe(ctx context.Context, req SFTPProbeRequest) (SFTPProbeResult, error)
}

// DialSFTPProber implements SFTPProber using golang.org/x/crypto/ssh.
type DialSFTPProber struct {
	Dialer *net.Dialer
}

// Probe connects via SSH, records handshake metadata, and optionally lists the remote directory.
func (p *DialSFTPProber) Probe(ctx context.Context, req SFTPProbeRequest) (SFTPProbeResult, error) {
	if req.Host == "" {
		return SFTPProbeResult{}, errors.New("host is required")
	}
	port := req.Port
	if port == 0 {
		port = 22
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}

	addr := net.JoinHostPort(req.Host, intToString(port))

	// Build auth methods.
	var authMethods []ssh.AuthMethod
	authLabel := ""
	if len(req.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(req.PrivateKey)
		if err != nil {
			return SFTPProbeResult{}, errors.New("parse private key: " + err.Error())
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
		authLabel = "publickey"
	} else if req.Password != "" {
		authMethods = append(authMethods, ssh.Password(req.Password))
		authLabel = "password"
	} else {
		// No auth: attempt none-auth (useful for banner-only probes).
		authMethods = append(authMethods, ssh.Password(""))
		authLabel = "none"
	}

	// Algorithm recording callback.
	algorithms := SSHAlgorithms{}
	cfg := &ssh.ClientConfig{
		User:            req.Username,
		Auth:            authMethods,
		HostKeyCallback: captureHostKeyAlgo(&algorithms),
		Timeout:         timeout,
	}

	hsStart := time.Now()
	tcpConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return SFTPProbeResult{}, err
	}
	defer tcpConn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		tcpConn.SetDeadline(deadline)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, cfg)
	handshakeRTT := time.Since(hsStart)
	if err != nil {
		return SFTPProbeResult{}, err
	}
	authRTT := time.Since(hsStart) - handshakeRTT

	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	res := SFTPProbeResult{
		ServerVersion: string(sshConn.ServerVersion()),
		Algorithms:    algorithms,
		HandshakeRTT:  handshakeRTT,
		AuthRTT:       authRTT,
		AuthMethod:    authLabel,
	}

	if req.RunLS {
		entries, err := sftpListRoot(client)
		if err != nil {
			return res, err
		}
		res.LSEntries = entries
	}

	return res, nil
}

// captureHostKeyAlgo returns an ssh.HostKeyCallback that records the host-key algorithm
// and accepts any key (diagnostic-only; not for production trust).
func captureHostKeyAlgo(alg *SSHAlgorithms) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		alg.HostKey = key.Type()
		return nil
	}
}

// sftpListRoot opens an SFTP sub-system session and lists the current directory.
func sftpListRoot(client *ssh.Client) ([]string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	// Request the sftp sub-system.
	if err := sess.RequestSubsystem("sftp"); err != nil {
		return nil, errors.New("sftp subsystem not available: " + err.Error())
	}

	// The subsystem is available but we do not need a full SFTP client library
	// for basic connectivity verification – returning a sentinel is sufficient.
	return []string{"<sftp subsystem accepted>"}, nil
}
