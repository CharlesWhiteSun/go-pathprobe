package diag

import "go-pathprobe/pkg/netprobe"

// DiagReport accumulates structured results produced by runners during a
// single diagnostic run.  It is injected into Request.Report (nil pointer =
// reporting disabled) so callers can opt-in without modifying runner logic.
type DiagReport struct {
	Target   Target
	Host     string
	PublicIP string

	// Ports holds per-port connectivity statistics from ConnectivityRunner.
	Ports []netprobe.PortProbeResult

	// Protos holds protocol-level probe results from each protocol runner.
	Protos []ProtoResult
}

// ProtoResult captures the outcome of a single protocol probe attempt.
type ProtoResult struct {
	Protocol string // e.g. "http", "smtp", "ftp", "sftp"
	Host     string
	Port     int
	OK       bool
	Summary  string
	Details  map[string]any // optional extra key-value pairs
}

// AddProto appends a protocol result.  It is nil-safe.
func (r *DiagReport) AddProto(pr ProtoResult) {
	if r != nil {
		r.Protos = append(r.Protos, pr)
	}
}

// SetPublicIP stores the detected public IP.  It is nil-safe.
func (r *DiagReport) SetPublicIP(ip string) {
	if r != nil {
		r.PublicIP = ip
	}
}

// AddPorts appends port probe results.  It is nil-safe.
func (r *DiagReport) AddPorts(results []netprobe.PortProbeResult) {
	if r != nil {
		r.Ports = append(r.Ports, results...)
	}
}
