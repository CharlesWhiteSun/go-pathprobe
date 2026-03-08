package netprobe

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// FTPProbeRequest contains inputs for an FTP/FTPS probe.
type FTPProbeRequest struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool // implicit FTPS (port 990 typically)
	AuthTLS  bool // explicit FTPS via AUTH TLS after control connection
	Insecure bool // skip TLS certificate verification
	Timeout  time.Duration
	RunLIST  bool // attempt PASV + LIST after authentication
}

// FTPProbeResult captures FTP handshake and command outcomes.
type FTPProbeResult struct {
	Banner          string
	UsedAuthTLS     bool
	UsedImplicitTLS bool
	LoginAccepted   bool
	ListEntries     []string
	RTT             time.Duration
}

// FTPProber performs an FTP/FTPS handshake and optional directory listing.
type FTPProber interface {
	Probe(ctx context.Context, req FTPProbeRequest) (FTPProbeResult, error)
}

// DialFTPProber implements FTPProber using raw FTP dialogue.
type DialFTPProber struct {
	Dialer *net.Dialer
}

// Probe executes the FTP/FTPS control-channel dialogue.
func (p *DialFTPProber) Probe(ctx context.Context, req FTPProbeRequest) (FTPProbeResult, error) {
	if req.Host == "" {
		return FTPProbeResult{}, errors.New("host is required")
	}
	if req.Port == 0 {
		if req.UseTLS {
			req.Port = 990
		} else {
			req.Port = 21
		}
	}
	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tlsCfg := &tls.Config{ServerName: req.Host, InsecureSkipVerify: req.Insecure}
	addr := net.JoinHostPort(req.Host, strconv.Itoa(req.Port))
	start := time.Now()

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return FTPProbeResult{}, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	res := FTPProbeResult{}

	// Implicit FTPS: wrap in TLS before any dialogue.
	if req.UseTLS {
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			return FTPProbeResult{}, err
		}
		conn = tlsConn
		res.UsedImplicitTLS = true
	}

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	// Read server banner (may be multi-line 220-).
	banner, err := ftpReadResponse(rw)
	if err != nil {
		return FTPProbeResult{}, err
	}
	res.Banner = banner
	res.RTT = time.Since(start)

	// Explicit FTPS: AUTH TLS upgrade.
	if req.AuthTLS && !req.UseTLS {
		if err := ftpSend(rw, "AUTH TLS"); err != nil {
			return res, err
		}
		if _, err := ftpReadResponse(rw); err != nil {
			return res, err
		}
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			return res, err
		}
		conn = tlsConn
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetDeadline(deadline)
		}
		rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		res.UsedAuthTLS = true
		// PBSZ 0 + PROT P required after AUTH TLS per RFC 4217.
		if err := ftpSend(rw, "PBSZ 0"); err != nil {
			return res, err
		}
		if _, err := ftpReadResponse(rw); err != nil {
			return res, err
		}
		if err := ftpSend(rw, "PROT P"); err != nil {
			return res, err
		}
		if _, err := ftpReadResponse(rw); err != nil {
			return res, err
		}
	}

	// USER / PASS login.
	if req.Username != "" {
		if err := ftpSend(rw, "USER "+req.Username); err != nil {
			return res, err
		}
		if _, err := ftpReadResponse(rw); err != nil {
			return res, err
		}
		if req.Password != "" {
			if err := ftpSend(rw, "PASS "+req.Password); err != nil {
				return res, err
			}
			resp, err := ftpReadResponse(rw)
			if err != nil {
				return res, err
			}
			res.LoginAccepted = strings.HasPrefix(resp, "230")
		}
	}

	// PASV + LIST directory listing.
	if req.RunLIST && res.LoginAccepted {
		entries, err := ftpPassiveList(ctx, rw, conn, tlsCfg, req.AuthTLS || req.UseTLS)
		if err != nil {
			return res, err
		}
		res.ListEntries = entries
	}

	return res, nil
}

// ftpPassiveList issues PASV, opens a data connection, sends LIST, and reads directory entries.
func ftpPassiveList(ctx context.Context, ctrl *bufio.ReadWriter, ctrlConn net.Conn, tlsCfg *tls.Config, protectedData bool) ([]string, error) {
	if err := ftpSend(ctrl, "PASV"); err != nil {
		return nil, err
	}
	resp, err := ftpReadResponse(ctrl)
	if err != nil {
		return nil, err
	}
	dataHost, dataPort, err := parsePASV(resp)
	if err != nil {
		return nil, err
	}

	dataAddr := net.JoinHostPort(dataHost, strconv.Itoa(dataPort))
	dialer := &net.Dialer{}
	dataConn, err := dialer.DialContext(ctx, "tcp", dataAddr)
	if err != nil {
		return nil, err
	}
	defer dataConn.Close()

	// Wrap data connection in TLS when PROT P is in effect.
	var dataReader *bufio.Reader
	if protectedData && tlsCfg != nil {
		dataTLS := tls.Client(dataConn, tlsCfg)
		if err := dataTLS.Handshake(); err != nil {
			return nil, err
		}
		dataReader = bufio.NewReader(dataTLS)
	} else {
		dataReader = bufio.NewReader(dataConn)
	}

	if err := ftpSend(ctrl, "LIST"); err != nil {
		return nil, err
	}
	// Consume "150 Opening data connection" from control channel.
	if _, err := ftpReadResponse(ctrl); err != nil {
		return nil, err
	}

	var entries []string
	scanner := bufio.NewScanner(dataReader)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line != "" {
			entries = append(entries, line)
		}
	}

	// Consume "226 Transfer complete" from control channel.
	if _, err := ftpReadResponse(ctrl); err != nil {
		return nil, err
	}
	return entries, nil
}

// parsePASV parses the host and port from an FTP 227 PASV response.
// Format: 227 Entering Passive Mode (h1,h2,h3,h4,p1,p2)
func parsePASV(resp string) (string, int, error) {
	start := strings.LastIndex(resp, "(")
	end := strings.LastIndex(resp, ")")
	if start < 0 || end < 0 || end <= start {
		return "", 0, fmt.Errorf("cannot parse PASV response: %q", resp)
	}
	parts := strings.Split(resp[start+1:end], ",")
	if len(parts) != 6 {
		return "", 0, fmt.Errorf("unexpected PASV tuple length: %q", resp)
	}
	host := strings.Join(parts[:4], ".")
	p1, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return "", 0, fmt.Errorf("PASV port parse: %w", err)
	}
	p2, err := strconv.Atoi(strings.TrimSpace(parts[5]))
	if err != nil {
		return "", 0, fmt.Errorf("PASV port parse: %w", err)
	}
	return host, p1*256 + p2, nil
}

// ftpSend writes an FTP command terminated by CRLF.
func ftpSend(rw *bufio.ReadWriter, cmd string) error {
	if _, err := rw.WriteString(cmd + "\r\n"); err != nil {
		return err
	}
	return rw.Flush()
}

// ftpReadResponse reads a potentially multi-line FTP response and returns
// the last (or only) line which contains the three-digit code.
func ftpReadResponse(rw *bufio.ReadWriter) (string, error) {
	var last string
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return last, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		last = trimmed
		// Continuation lines: "XXX-" where XXX matches the code.
		if len(trimmed) < 4 || trimmed[3] != '-' {
			break
		}
	}
	if last == "" {
		return "", errors.New("empty FTP response")
	}
	return last, nil
}
