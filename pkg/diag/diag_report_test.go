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
	r.SetRoute(nil)
	r.SetRoute(&netprobe.RouteResult{})
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

// TestSetRouteNilSafe verifies that SetRoute on a nil *DiagReport never panics.
func TestSetRouteNilSafe(t *testing.T) {
	var r *DiagReport
	// Must not panic regardless of argument.
	r.SetRoute(nil)
	r.SetRoute(&netprobe.RouteResult{})
}

// TestSetRouteStoresResult verifies that SetRoute correctly assigns the route.
func TestSetRouteStoresResult(t *testing.T) {
	r := &DiagReport{Target: TargetWeb, Host: "example.com"}
	if r.Route != nil {
		t.Fatal("expected nil Route before SetRoute")
	}

	route := &netprobe.RouteResult{
		Hops: []netprobe.HopResult{
			{TTL: 1, IP: "192.168.1.1", Stats: netprobe.ProbeStats{AvgRTT: 2 * time.Millisecond}},
			{TTL: 2, IP: "10.0.0.1", Stats: netprobe.ProbeStats{AvgRTT: 8 * time.Millisecond}},
			{TTL: 3, IP: "", Stats: netprobe.ProbeStats{LossPct: 100}}, // timed-out hop
		},
	}
	r.SetRoute(route)

	if r.Route == nil {
		t.Fatal("expected Route to be set after SetRoute")
	}
	if len(r.Route.Hops) != 3 {
		t.Fatalf("expected 3 hops, got %d", len(r.Route.Hops))
	}
	if r.Route.Hops[2].IP != "" {
		t.Fatalf("expected timed-out hop IP empty, got %q", r.Route.Hops[2].IP)
	}
}

// TestSetRouteOverwrite verifies that calling SetRoute twice replaces the value.
func TestSetRouteOverwrite(t *testing.T) {
	r := &DiagReport{Target: TargetWeb, Host: "example.com"}

	first := &netprobe.RouteResult{Hops: []netprobe.HopResult{{TTL: 1, IP: "1.1.1.1"}}}
	second := &netprobe.RouteResult{Hops: []netprobe.HopResult{{TTL: 1, IP: "2.2.2.2"}, {TTL: 2, IP: "3.3.3.3"}}}

	r.SetRoute(first)
	r.SetRoute(second)

	if r.Route != second {
		t.Fatal("expected Route to point to second result after overwrite")
	}
	if len(r.Route.Hops) != 2 {
		t.Fatalf("expected 2 hops after overwrite, got %d", len(r.Route.Hops))
	}
}
