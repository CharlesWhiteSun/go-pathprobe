package diag

import (
	"go-pathprobe/pkg/netprobe"
	"testing"
	"time"
)

// TestDiagReportNilSafe verifies that nil-pointer methods never panic.
func TestDiagReportNilSafe(t *testing.T) {
	var r *DiagReport
	// None of these should panic.
	r.SetPublicIP("1.2.3.4")
	r.AddProto(ProtoResult{Protocol: "http"})
	r.AddPorts([]netprobe.PortProbeResult{{Port: 80}})
}

// TestDiagReportAccumulates verifies that results are correctly accumulated.
func TestDiagReportAccumulates(t *testing.T) {
	r := &DiagReport{Target: TargetWeb, Host: "example.com"}

	r.SetPublicIP("9.9.9.9")
	if r.PublicIP != "9.9.9.9" {
		t.Fatalf("expected public IP '9.9.9.9', got %q", r.PublicIP)
	}

	r.AddProto(ProtoResult{Protocol: "http", OK: true, Summary: "HTTP 200"})
	r.AddProto(ProtoResult{Protocol: "smtp", OK: false, Summary: "timeout"})
	if len(r.Protos) != 2 {
		t.Fatalf("expected 2 proto results, got %d", len(r.Protos))
	}

	r.AddPorts([]netprobe.PortProbeResult{
		{Port: 80, Stats: netprobe.ProbeStats{Sent: 5, Received: 5, AvgRTT: 10 * time.Millisecond}},
		{Port: 443, Stats: netprobe.ProbeStats{Sent: 5, Received: 3, LossPct: 40}},
	})
	if len(r.Ports) != 2 {
		t.Fatalf("expected 2 port results, got %d", len(r.Ports))
	}
}
