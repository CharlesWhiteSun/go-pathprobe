package netprobe

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// SMTPProbeRequest describes the SMTP flow to execute.
type SMTPProbeRequest struct {
	Host      string
	Port      int
	Domain    string
	Username  string
	Password  string
	From      string
	To        []string
	UseTLS    bool // implicit TLS
	StartTLS  bool
	Timeout   time.Duration
	HelloName string
}

// SMTPProbeResult captures handshake and command outcomes.
type SMTPProbeResult struct {
	Banner           string
	Capabilities     []string
	UsedStartTLS     bool
	AuthTried        []string
	MailFromAccepted bool
	RcptAccepted     []string
	RTT              time.Duration
}

// SMTPProber performs an SMTP handshake and optional auth/mail/rcpt.
type SMTPProber interface {
	Probe(ctx context.Context, req SMTPProbeRequest) (SMTPProbeResult, error)
}

// DialSMTPProber implements SMTPProber using raw SMTP dialogue.
type DialSMTPProber struct {
	Dialer *net.Dialer
}

func (p *DialSMTPProber) Probe(ctx context.Context, req SMTPProbeRequest) (SMTPProbeResult, error) {
	if req.Host == "" {
		return SMTPProbeResult{}, errors.New("host is required")
	}
	if req.Port == 0 {
		req.Port = 25
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

	address := net.JoinHostPort(req.Host, intToString(req.Port))
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return SMTPProbeResult{}, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	if req.UseTLS {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: req.Host, InsecureSkipVerify: true})
		if err := tlsConn.Handshake(); err != nil {
			return SMTPProbeResult{}, err
		}
		conn = tlsConn
	}

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	banner, err := readLine(rw)
	if err != nil {
		return SMTPProbeResult{}, err
	}
	res := SMTPProbeResult{Banner: banner, RTT: time.Since(start)}

	hello := req.HelloName
	if hello == "" {
		hello = req.Domain
	}
	if hello == "" {
		hello = "localhost"
	}

	if err := writeLine(rw, fmt.Sprintf("EHLO %s", hello)); err != nil {
		return res, err
	}
	caps, err := readMultiLines(rw)
	if err != nil {
		return res, err
	}
	res.Capabilities = caps

	if req.StartTLS && supportsStartTLS(caps) && !req.UseTLS {
		if err := writeLine(rw, "STARTTLS"); err != nil {
			return res, err
		}
		if _, err := readLine(rw); err != nil {
			return res, err
		}
		tlsConn := tls.Client(conn, &tls.Config{ServerName: req.Host, InsecureSkipVerify: true})
		if err := tlsConn.Handshake(); err != nil {
			return res, err
		}
		conn = tlsConn
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetDeadline(deadline)
		}
		rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		res.UsedStartTLS = true
		// Re-issue EHLO after STARTTLS
		if err := writeLine(rw, fmt.Sprintf("EHLO %s", hello)); err != nil {
			return res, err
		}
		caps, err = readMultiLines(rw)
		if err != nil {
			return res, err
		}
		res.Capabilities = caps
	}

	if req.Username != "" && req.Password != "" && supportsAuthLogin(caps) {
		res.AuthTried = append(res.AuthTried, "LOGIN")
		if err := authLogin(rw, req.Username, req.Password); err != nil {
			return res, err
		}
	}

	// MAIL FROM / RCPT TO if provided
	if req.From != "" {
		if err := writeLine(rw, fmt.Sprintf("MAIL FROM:<%s>", req.From)); err != nil {
			return res, err
		}
		if _, err := readLine(rw); err != nil {
			return res, err
		}
		res.MailFromAccepted = true
	}
	for _, to := range req.To {
		if err := writeLine(rw, fmt.Sprintf("RCPT TO:<%s>", strings.TrimSpace(to))); err != nil {
			return res, err
		}
		if _, err := readLine(rw); err != nil {
			return res, err
		}
		res.RcptAccepted = append(res.RcptAccepted, strings.TrimSpace(to))
	}

	return res, nil
}

func readLine(rw *bufio.ReadWriter) (string, error) {
	line, err := rw.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func readMultiLines(rw *bufio.ReadWriter) ([]string, error) {
	var lines []string
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		lines = append(lines, trimmed)
		if len(trimmed) < 4 || trimmed[3] != '-' {
			break
		}
	}
	return lines, nil
}

func writeLine(rw *bufio.ReadWriter, line string) error {
	if _, err := rw.WriteString(line + "\r\n"); err != nil {
		return err
	}
	return rw.Flush()
}

func supportsStartTLS(caps []string) bool {
	for _, c := range caps {
		if strings.Contains(strings.ToUpper(c), "STARTTLS") {
			return true
		}
	}
	return false
}

func supportsAuthLogin(caps []string) bool {
	for _, c := range caps {
		upper := strings.ToUpper(c)
		if strings.Contains(upper, "AUTH") && strings.Contains(upper, "LOGIN") {
			return true
		}
	}
	return false
}

func authLogin(rw *bufio.ReadWriter, username, password string) error {
	if err := writeLine(rw, "AUTH LOGIN"); err != nil {
		return err
	}
	if _, err := readLine(rw); err != nil {
		return err
	}
	if err := writeLine(rw, encodeBase64(username)); err != nil {
		return err
	}
	if _, err := readLine(rw); err != nil {
		return err
	}
	if err := writeLine(rw, encodeBase64(password)); err != nil {
		return err
	}
	if _, err := readLine(rw); err != nil {
		return err
	}
	return nil
}

func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
