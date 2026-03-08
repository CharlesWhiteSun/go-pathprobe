package netprobe

import (
	"context"
	"errors"
	"net"
	"time"
)

// ProbeAttempt captures a single connectivity attempt outcome.
type ProbeAttempt struct {
	RTT     time.Duration
	Success bool
	Err     error
}

// PortProbeResult aggregates attempts for a port.
type PortProbeResult struct {
	Port     int
	Attempts []ProbeAttempt
	Stats    ProbeStats
}

// ProbeStats summarizes loss and latency.
type ProbeStats struct {
	Sent       int
	Received   int
	LossPct    float64
	MinRTT     time.Duration
	MaxRTT     time.Duration
	AvgRTT     time.Duration
	LastErrStr string
}

// PortProber performs one attempt for host:port.
type PortProber interface {
	ProbeOnce(ctx context.Context, host string, port int) ProbeAttempt
}

// TCPPortProber dials TCP to assess reachability.
type TCPPortProber struct {
	Dialer  *net.Dialer
	Timeout time.Duration
}

// ProbeOnce implements PortProber.
func (p *TCPPortProber) ProbeOnce(ctx context.Context, host string, port int) ProbeAttempt {
	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(host, intToString(port))
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	rtt := time.Since(start)
	if err != nil {
		return ProbeAttempt{RTT: rtt, Success: false, Err: err}
	}
	conn.Close()
	return ProbeAttempt{RTT: rtt, Success: true}
}

// ProbePorts runs multiple attempts per port and computes stats.
func ProbePorts(ctx context.Context, host string, ports []int, attempts int, prober PortProber) ([]PortProbeResult, error) {
	if attempts <= 0 {
		return nil, errors.New("attempts must be >0")
	}
	if prober == nil {
		return nil, errors.New("prober is required")
	}
	results := make([]PortProbeResult, 0, len(ports))
	for _, port := range ports {
		attemptsRes := make([]ProbeAttempt, 0, attempts)
		for i := 0; i < attempts; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			attempt := prober.ProbeOnce(ctx, host, port)
			attemptsRes = append(attemptsRes, attempt)
		}
		stats := computeStats(attemptsRes)
		stats.Sent = attempts
		stats.Received = len(filterSuccess(attemptsRes))
		results = append(results, PortProbeResult{Port: port, Attempts: attemptsRes, Stats: stats})
	}
	return results, nil
}

func computeStats(attempts []ProbeAttempt) ProbeStats {
	var stats ProbeStats
	if len(attempts) == 0 {
		return stats
	}
	var sum time.Duration
	stats.MinRTT = time.Duration(1<<63 - 1)
	for _, a := range attempts {
		if a.Success {
			if a.RTT < stats.MinRTT {
				stats.MinRTT = a.RTT
			}
			if a.RTT > stats.MaxRTT {
				stats.MaxRTT = a.RTT
			}
			sum += a.RTT
		}
		if a.Err != nil {
			stats.LastErrStr = a.Err.Error()
		}
	}
	successes := len(filterSuccess(attempts))
	stats.Received = successes
	stats.Sent = len(attempts)
	if stats.Sent > 0 {
		stats.LossPct = float64(stats.Sent-stats.Received) * 100 / float64(stats.Sent)
	}
	if successes > 0 {
		stats.AvgRTT = time.Duration(int64(sum) / int64(successes))
	}
	if stats.MinRTT == time.Duration(1<<63-1) {
		stats.MinRTT = 0
	}
	return stats
}

func filterSuccess(attempts []ProbeAttempt) []ProbeAttempt {
	var out []ProbeAttempt
	for _, a := range attempts {
		if a.Success {
			out = append(out, a)
		}
	}
	return out
}

func intToString(v int) string {
	// avoid strconv import to keep dependencies minimal
	return fmtInt(v)
}

// minimal int->string conversion
func fmtInt(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [32]byte
	i := len(buf)
	n := v
	if n < 0 {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if v < 0 {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
