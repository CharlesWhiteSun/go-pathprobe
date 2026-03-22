package netprobe

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// OsTracerouteProber executes the platform-native traceroute command and
// parses its output to produce per-hop results.
//
//   - Windows: tracert -d -h <maxHops> <host>
//   - Linux/macOS: traceroute -n -m <maxHops> -q <attemptsPerHop> <host>
//
// Unlike ICMPTracerouteProber and TCPTracerouteProber, this implementation
// correctly reports intermediate router IP addresses on all platforms and
// requires no elevated OS privileges (tracert.exe is always available on
// Windows; traceroute is standard on most Unix systems).
type OsTracerouteProber struct {
	// ReverseLookup controls whether PTR records are resolved for each hop.
	// Defaults to true.  The native tool's own DNS lookup is suppressed
	// (-d / -n flags) so resolution is performed uniformly by reverseLookup.
	ReverseLookup *bool
}

func (p *OsTracerouteProber) reverseLookupEnabled() bool {
	return p.ReverseLookup == nil || *p.ReverseLookup
}

// Trace implements TracerouteProber using the platform-native traceroute command.
func (p *OsTracerouteProber) Trace(ctx context.Context, host string, maxHops, attemptsPerHop int, onHop HopEmitter) (RouteResult, error) {
	if maxHops <= 0 {
		return RouteResult{}, fmt.Errorf("maxHops must be > 0, got %d", maxHops)
	}
	if attemptsPerHop <= 0 {
		return RouteResult{}, fmt.Errorf("attemptsPerHop must be > 0, got %d", attemptsPerHop)
	}

	// Resolve the target to a concrete IPv4 address before passing it to the
	// OS tool.  This ensures we compare hop IPs against the right address
	// even if the tool resolves differently.
	dstAddrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return RouteResult{}, fmt.Errorf("traceroute os: resolve %q: %w", host, err)
	}
	dstIP := pickIPv4Addr(dstAddrs)
	if dstIP == "" {
		dstIP = dstAddrs[0]
	}

	args := osTracerouteArgs(dstIP, maxHops, attemptsPerHop)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RouteResult{}, fmt.Errorf("traceroute os: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return RouteResult{}, fmt.Errorf("traceroute os: start %q: %w", args[0], err)
	}

	var result RouteResult
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		hop, ok := parseOsHopLine(scanner.Text())
		if !ok {
			continue
		}
		if hop.IP != "" && p.reverseLookupEnabled() {
			hop.Hostname = reverseLookup(hop.IP)
		}
		result.Hops = append(result.Hops, hop)
		if onHop != nil {
			onHop(hop)
		}
		// Stop parsing once we see the destination — the native tool will
		// continue printing "Trace complete." etc. which the scanner can skip.
		if hop.IP == dstIP {
			break
		}
	}

	// Reap the subprocess; non-zero exit is normal when any hop times out.
	_ = cmd.Wait()

	return result, nil
}

// osTracerouteArgs returns the command + arguments for the platform-native
// traceroute tool.  DNS resolution is suppressed so the caller controls it.
func osTracerouteArgs(host string, maxHops, attemptsPerHop int) []string {
	if runtime.GOOS == "windows" {
		return []string{
			"tracert",
			"-d",                        // Suppress hostname resolution
			"-h", strconv.Itoa(maxHops), // Maximum hops
			host,
		}
	}
	// Linux / macOS
	return []string{
		"traceroute",
		"-n",                        // Suppress hostname resolution
		"-m", strconv.Itoa(maxHops), // Maximum hops
		"-q", strconv.Itoa(attemptsPerHop), // Probes per hop
		host,
	}
}

// Compiled patterns used by parseOsHopLine.
var (
	// osHopNumRe matches a line that opens with an optional hop number (1–999).
	osHopNumRe = regexp.MustCompile(`^\s*(\d{1,3})\b`)

	// osIPv4Re finds the first IPv4 address string inside a hop line.
	osIPv4Re = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)

	// osRTTRe matches RTT tokens emitted by both tracert ("<1 ms", "12 ms")
	// and traceroute ("0.456 ms", "12.5 ms").
	osRTTRe = regexp.MustCompile(`<?\d+(?:\.\d+)?\s+ms`)
)

// parseOsHopLine extracts a HopResult from one output line of tracert /
// traceroute.  Returns (HopResult, true) for hop lines; (HopResult{}, false)
// for header, footer, and blank lines.
//
// Accepted formats:
//
//	Windows tracert:
//	  "  1    <1 ms    <1 ms    <1 ms  192.168.1.1"
//	  "  3     *        *        *     Request timed out."
//	  "  4     5 ms     *        6 ms  172.16.0.1"
//
//	Unix traceroute (-n):
//	  " 1  192.168.1.1  0.456 ms  0.234 ms  0.123 ms"
//	  " 3  * * *"
//	  " 4  10.0.0.1  8.1 ms  *  7.9 ms"
func parseOsHopLine(line string) (HopResult, bool) {
	m := osHopNumRe.FindStringSubmatch(line)
	if m == nil {
		return HopResult{}, false
	}
	ttl, _ := strconv.Atoi(m[1])

	// First IPv4 address in the line is the responding router.
	var ip string
	if ipM := osIPv4Re.FindStringSubmatch(line); ipM != nil {
		ip = ipM[1]
	}

	var attempts []ProbeAttempt

	// Parse individual RTT measurements.
	for _, rttStr := range osRTTRe.FindAllString(line, -1) {
		// Strip leading '<' (tracert's "<1") then split into value + "ms".
		rttStr = strings.TrimPrefix(strings.TrimSpace(rttStr), "<")
		fields := strings.Fields(rttStr)
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			continue
		}
		rtt := time.Duration(v * float64(time.Millisecond))
		attempts = append(attempts, ProbeAttempt{RTT: rtt, Success: ip != ""})
	}

	// Each whitespace-separated token that is exactly "*" is a timed-out probe.
	for _, tok := range strings.Fields(line) {
		if tok == "*" {
			attempts = append(attempts, ProbeAttempt{Success: false})
		}
	}

	if len(attempts) == 0 {
		// No probe data found — header/footer line such as "Trace complete."
		return HopResult{}, false
	}

	stats := computeStats(attempts)
	return HopResult{TTL: ttl, IP: ip, Attempts: attempts, Stats: stats}, true
}
