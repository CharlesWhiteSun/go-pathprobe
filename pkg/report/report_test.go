package report_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/netprobe"
	"go-pathprobe/pkg/report"
)

// --- helpers -----------------------------------------------------------------

func buildSampleDiagReport() *diag.DiagReport {
	r := &diag.DiagReport{Target: diag.TargetWeb, Host: "example.com"}
	r.SetPublicIP("1.2.3.4")
	r.AddPorts([]netprobe.PortProbeResult{
		{Port: 80, Stats: netprobe.ProbeStats{Sent: 5, Received: 5, AvgRTT: 10 * time.Millisecond}},
		{Port: 443, Stats: netprobe.ProbeStats{Sent: 5, Received: 3, LossPct: 40, MinRTT: 8 * time.Millisecond, AvgRTT: 12 * time.Millisecond}},
	})
	r.AddProto(diag.ProtoResult{Protocol: "http", Host: "example.com", Port: 443, OK: true, Summary: "HTTP 200, RTT 25ms"})
	r.AddProto(diag.ProtoResult{Protocol: "smtp", Host: "mail.example.com", Port: 25, OK: false, Summary: "connection refused"})
	return r
}

func buildAnnotated(t *testing.T, dr *diag.DiagReport) *report.AnnotatedReport {
	t.Helper()
	ar, err := report.Build(context.Background(), dr, geo.NoopLocator{})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	return ar
}

// --- report.Build ------------------------------------------------------------

func TestBuildPopulatesFields(t *testing.T) {
	dr := buildSampleDiagReport()
	ar, err := report.Build(context.Background(), dr, geo.NoopLocator{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ar.Target != "web" {
		t.Fatalf("expected target 'web', got %q", ar.Target)
	}
	if ar.Host != "example.com" {
		t.Fatalf("expected host 'example.com', got %q", ar.Host)
	}
	if len(ar.Ports) != 2 {
		t.Fatalf("expected 2 port entries, got %d", len(ar.Ports))
	}
	if len(ar.Protos) != 2 {
		t.Fatalf("expected 2 proto entries, got %d", len(ar.Protos))
	}
	if ar.GeneratedAt == "" {
		t.Fatalf("expected non-empty GeneratedAt")
	}
}

func TestBuildPortEntryValues(t *testing.T) {
	dr := buildSampleDiagReport()
	ar := buildAnnotated(t, dr)

	p80 := ar.Ports[0]
	if p80.Port != 80 || p80.Sent != 5 || p80.Received != 5 || p80.LossPct != 0 {
		t.Fatalf("unexpected port 80 entry: %+v", p80)
	}
	p443 := ar.Ports[1]
	if p443.LossPct != 40 {
		t.Fatalf("expected loss 40%%, got %.1f%%", p443.LossPct)
	}
}

func TestBuildProtoEntryValues(t *testing.T) {
	dr := buildSampleDiagReport()
	ar := buildAnnotated(t, dr)

	if !ar.Protos[0].OK {
		t.Fatalf("expected first proto OK=true")
	}
	if ar.Protos[1].OK {
		t.Fatalf("expected second proto OK=false")
	}
	if ar.Protos[0].Protocol != "http" {
		t.Fatalf("expected protocol 'http', got %q", ar.Protos[0].Protocol)
	}
}

func TestBuildNoopGeoLeavesEmpty(t *testing.T) {
	dr := buildSampleDiagReport()
	ar := buildAnnotated(t, dr)
	// With NoopLocator, IP is still populated (it's known) but
	// location fields (lat/lon/city/country/ASN) must be empty.
	if ar.PublicGeo.HasLocation || ar.PublicGeo.City != "" || ar.PublicGeo.ASN != 0 {
		t.Fatalf("expected no geo location data with NoopLocator, got %+v", ar.PublicGeo)
	}
	if ar.TargetGeo.HasLocation || ar.TargetGeo.City != "" {
		t.Fatalf("expected no geo location data for target with NoopLocator, got %+v", ar.TargetGeo)
	}
}

func TestBuildNilLocator(t *testing.T) {
	dr := buildSampleDiagReport()
	ar, err := report.Build(context.Background(), dr, nil)
	if err != nil {
		t.Fatalf("unexpected error with nil locator: %v", err)
	}
	if ar.Host != "example.com" {
		t.Fatalf("expected host populated")
	}
}

// --- TableWriter -------------------------------------------------------------

func TestTableWriterContainsKeyFields(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	var buf bytes.Buffer
	tw := report.TableWriter{}
	if err := tw.Write(&buf, ar); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"web", "example.com", "PORT CONNECTIVITY", "80", "443", "PROTOCOL RESULTS", "http", "smtp", "OK", "FAIL"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n%s", want, out)
		}
	}
}

func TestTableWriterEmptyReport(t *testing.T) {
	dr := &diag.DiagReport{Target: diag.TargetFTP, Host: ""}
	ar := buildAnnotated(t, dr)
	var buf bytes.Buffer
	tw := report.TableWriter{}
	if err := tw.Write(&buf, ar); err != nil {
		t.Fatalf("unexpected error for empty report: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty output")
	}
}

// --- JSONWriter --------------------------------------------------------------

func TestJSONWriterProducesValidJSON(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	var buf bytes.Buffer
	jw := report.JSONWriter{}
	if err := jw.Write(&buf, ar); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if payload["target"] != "web" {
		t.Fatalf("expected target 'web' in JSON, got %v", payload["target"])
	}
	if _, ok := payload["ports"]; !ok {
		t.Fatal("expected 'ports' key in JSON")
	}
	if _, ok := payload["protos"]; !ok {
		t.Fatal("expected 'protos' key in JSON")
	}
}

func TestJSONWriterPortLossField(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	var buf bytes.Buffer
	jw := report.JSONWriter{}
	_ = jw.Write(&buf, ar)

	var payload struct {
		Ports []struct {
			Port    int     `json:"Port"`
			LossPct float64 `json:"LossPct"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(payload.Ports))
	}
	if payload.Ports[1].LossPct != 40 {
		t.Fatalf("expected loss 40, got %v", payload.Ports[1].LossPct)
	}
}

// --- HTMLWriter --------------------------------------------------------------

func TestHTMLWriterContainsLeafletRef(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	var buf bytes.Buffer
	hw := report.HTMLWriter{}
	if err := hw.Write(&buf, ar); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "leaflet") {
		t.Fatal("expected Leaflet reference in HTML output")
	}
	if !strings.Contains(html, "PathProbe") {
		t.Fatal("expected PathProbe branding in HTML output")
	}
}

func TestHTMLWriterContainsResultData(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	var buf bytes.Buffer
	hw := report.HTMLWriter{}
	_ = hw.Write(&buf, ar)
	html := buf.String()

	for _, want := range []string{"example.com", "80", "443", "http", "smtp"} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML missing %q", want)
		}
	}
}

func TestHTMLWriterNoMapWhenNoGeo(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	var buf bytes.Buffer
	hw := report.HTMLWriter{}
	_ = hw.Write(&buf, ar)
	html := buf.String()
	// Without geo data the map div should not be rendered.
	if strings.Contains(html, "L.map(") {
		t.Fatal("expected no Leaflet map initialisation without geo data")
	}
}

func TestHTMLWriterWriteFile(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	path := t.TempDir() + "/report.html"
	hw := report.HTMLWriter{}
	if err := hw.WriteFile(path, ar); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
}

// --- Route in Build ----------------------------------------------------------

// buildRouteResult constructs a simple 4-hop RouteResult for testing.
// Hop 3 intentionally has an empty IP to simulate a non-responding router.
func buildRouteResult() *netprobe.RouteResult {
	return &netprobe.RouteResult{
		Hops: []netprobe.HopResult{
			{TTL: 1, IP: "192.168.1.1", Hostname: "gateway.local",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 3, AvgRTT: 2 * time.Millisecond}},
			{TTL: 2, IP: "10.0.0.1", Hostname: "",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 3, AvgRTT: 8 * time.Millisecond}},
			{TTL: 3, IP: "", Hostname: "",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 0, LossPct: 100}},
			{TTL: 4, IP: "93.184.216.34", Hostname: "example.com",
				Stats: netprobe.ProbeStats{Sent: 3, Received: 2, LossPct: 33.3, AvgRTT: 50 * time.Millisecond}},
		},
	}
}

func TestBuildRouteHopsCount(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())

	ar, err := report.Build(context.Background(), dr, geo.NoopLocator{})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if len(ar.Route) != 4 {
		t.Fatalf("expected 4 route hops, got %d", len(ar.Route))
	}
}

func TestBuildRouteTTLOrdering(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())

	ar := buildAnnotated(t, dr)

	for i, hop := range ar.Route {
		if hop.TTL != i+1 {
			t.Errorf("hop %d: expected TTL=%d, got %d", i, i+1, hop.TTL)
		}
	}
}

func TestBuildRouteTimedOutHop(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())

	ar := buildAnnotated(t, dr)

	timedOut := ar.Route[2] // TTL=3, IP=""
	if timedOut.IP != "" {
		t.Fatalf("expected empty IP for timed-out hop, got %q", timedOut.IP)
	}
	if timedOut.HasGeo {
		t.Fatal("timed-out hop should have HasGeo=false")
	}
	if timedOut.ASN != 0 {
		t.Fatalf("timed-out hop should have ASN=0, got %d", timedOut.ASN)
	}
	if timedOut.LossPct != 100 {
		t.Fatalf("expected 100%% loss for timed-out hop, got %.1f%%", timedOut.LossPct)
	}
}

func TestBuildRouteHopFields(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())

	ar := buildAnnotated(t, dr)

	h1 := ar.Route[0] // TTL=1, gateway.local
	if h1.IP != "192.168.1.1" {
		t.Fatalf("expected IP '192.168.1.1', got %q", h1.IP)
	}
	if h1.Hostname != "gateway.local" {
		t.Fatalf("expected Hostname 'gateway.local', got %q", h1.Hostname)
	}
	if h1.AvgRTT == "" || h1.AvgRTT == "—" {
		t.Fatalf("expected non-empty AvgRTT for hop with RTT, got %q", h1.AvgRTT)
	}

	h4 := ar.Route[3] // TTL=4, 33.3% loss
	if h4.LossPct != 33.3 {
		t.Fatalf("expected LossPct=33.3, got %.1f", h4.LossPct)
	}
}

func TestBuildRouteNilRouteMeansEmptySlice(t *testing.T) {
	dr := buildSampleDiagReport()
	// dr.Route is nil by default — no SetRoute called.

	ar := buildAnnotated(t, dr)

	if ar.Route != nil {
		t.Fatalf("expected nil Route slice when DiagReport.Route is nil, got %v", ar.Route)
	}
}

func TestBuildRouteWithNilLocator(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())

	ar, err := report.Build(context.Background(), dr, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Hops must still be populated even without a locator.
	if len(ar.Route) != 4 {
		t.Fatalf("expected 4 hops with nil locator, got %d", len(ar.Route))
	}
	// Geo fields must be empty.
	for _, hop := range ar.Route {
		if hop.HasGeo {
			t.Fatalf("expected HasGeo=false with nil locator, hop TTL=%d", hop.TTL)
		}
	}
}
