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

// ── Phase 6: HTML report route rendering ─────────────────────────────────

// TestHTMLWriterRouteTablePresent verifies that when an AnnotatedReport has
// Route entries, the HTML output includes a "Route Path" section with a table
// containing hop data (TTL numbers, IPs, and the timed-out hop placeholder).
func TestHTMLWriterRouteTablePresent(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())
	ar := buildAnnotated(t, dr)

	var buf bytes.Buffer
	hw := report.HTMLWriter{}
	if err := hw.Write(&buf, ar); err != nil {
		t.Fatalf("HTMLWriter.Write error: %v", err)
	}
	html := buf.String()

	// Section heading must appear.
	if !strings.Contains(html, "Route Path") {
		t.Error("HTML: missing 'Route Path' section heading")
	}
	// Known IPs must appear in the table.
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "93.184.216.34"} {
		if !strings.Contains(html, ip) {
			t.Errorf("HTML: missing expected IP %q in route table", ip)
		}
	}
	// Timed-out hop must display the ??? placeholder.
	if !strings.Contains(html, "???") {
		t.Error("HTML: timed-out hop must render '???' placeholder")
	}
	// Hostname from hop 1 and 4 must appear.
	if !strings.Contains(html, "gateway.local") {
		t.Error("HTML: missing hostname 'gateway.local' from hop 1")
	}
	if !strings.Contains(html, "example.com") {
		t.Error("HTML: missing hostname 'example.com' from hop 4")
	}
}

// TestHTMLWriterRouteTableAbsentWhenNoRoute confirms that the Route Path
// section is omitted entirely when the report has no traceroute data, to keep
// the HTML output clean for non-traceroute diagnostics.
func TestHTMLWriterRouteTableAbsentWhenNoRoute(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())

	var buf bytes.Buffer
	hw := report.HTMLWriter{}
	_ = hw.Write(&buf, ar)
	html := buf.String()

	if strings.Contains(html, "Route Path") {
		t.Error("HTML: 'Route Path' section must not appear when Route is nil")
	}
}

// TestHTMLWriterRouteLeafletPolyline checks that when all hops have geo
// coordinates (simulated via a geolocator stub), the HTML script block
// contains L.circleMarker and L.polyline calls for the route overlay.
// Uses a stub locator that returns a fixed coordinate for any IP.
func TestHTMLWriterRouteLeafletPolyline(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())

	ar, err := report.Build(context.Background(), dr, geoStubLocator{})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	// Mark hops as having geo data (stub locator sets HasLocation=true).
	// Verify the template emits the route overlay markers and polyline.
	var buf bytes.Buffer
	hw := report.HTMLWriter{}
	if err := hw.Write(&buf, ar); err != nil {
		t.Fatalf("HTMLWriter.Write error: %v", err)
	}
	html := buf.String()

	// circleMarker is emitted for each hop that HasGeo=true.
	if !strings.Contains(html, "circleMarker") {
		t.Error("HTML: expected L.circleMarker calls for geo-annotated route hops")
	}
	// The route polyline must be drawn when ≥2 hops have coordinates.
	if !strings.Contains(html, "routePts") {
		t.Error("HTML: expected routePts array for the route-path polyline")
	}
}

// ── Phase 6: TableWriter route rendering ─────────────────────────────────

// TestTableWriterRouteSection verifies that WriteRoute (called from Write)
// appends a ROUTE PATH section when Route entries are present, including the
// section header, column headers, all TTL values, known IPs, and the timed-out
// "???" placeholder for hops with an empty IP.
func TestTableWriterRouteSection(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.SetRoute(buildRouteResult())
	ar := buildAnnotated(t, dr)

	var buf bytes.Buffer
	tw := report.TableWriter{}
	if err := tw.Write(&buf, ar); err != nil {
		t.Fatalf("TableWriter.Write error: %v", err)
	}
	out := buf.String()

	// Section header.
	if !strings.Contains(out, "ROUTE PATH") {
		t.Error("table: missing 'ROUTE PATH' section header")
	}
	// Column headers.
	for _, col := range []string{"HOP", "IP", "HOSTNAME", "LOSS%", "AVG RTT"} {
		if !strings.Contains(out, col) {
			t.Errorf("table: missing column header %q", col)
		}
	}
	// Known hop IPs.
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "93.184.216.34"} {
		if !strings.Contains(out, ip) {
			t.Errorf("table: missing IP %q in route output", ip)
		}
	}
	// Timed-out hop placeholder.
	if !strings.Contains(out, "???") {
		t.Error("table: timed-out hop must render '???'")
	}
	// Timed-out hop loss must show "—", not a percentage.
	if !strings.Contains(out, "—") {
		t.Error("table: timed-out hop loss must show '—'")
	}
}

// TestTableWriterNoRouteSection confirms that the ROUTE PATH section is
// absent when the report contains no Route data, keeping the text output
// consistent for non-traceroute targets.
func TestTableWriterNoRouteSection(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())

	var buf bytes.Buffer
	tw := report.TableWriter{}
	_ = tw.Write(&buf, ar)
	out := buf.String()

	if strings.Contains(out, "ROUTE PATH") {
		t.Error("table: 'ROUTE PATH' section must not appear when Route is nil")
	}
}

// ── DNS section in Build ─────────────────────────────────────────────────

// TestBuildDNSPopulatesEntries verifies that DiagReport.DNSComparisons are
// flattened into AnnotatedReport.DNS with correct Domain, Type, HasDivergence,
// and per-resolver Answer entries.
func TestBuildDNSPopulatesEntries(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.AddDNSComparisons([]netprobe.DNSComparison{
		{
			Name: "example.com",
			Type: netprobe.RecordTypeA,
			Results: []netprobe.DNSAnswer{
				{Source: "sys", Values: []string{"93.184.216.34"}, RTT: 5 * time.Millisecond},
				{Source: "cf", Values: []string{"93.184.216.34"}, RTT: 3 * time.Millisecond},
			},
		},
		{
			Name: "example.com",
			Type: netprobe.RecordTypeAAAA,
			Results: []netprobe.DNSAnswer{
				{Source: "sys", Values: []string{"2606:2800::1"}, RTT: 6 * time.Millisecond},
				{Source: "cf", Values: []string{"2606:2800::2"}, RTT: 4 * time.Millisecond},
			},
		},
	})

	ar := buildAnnotated(t, dr)

	if len(ar.DNS) != 2 {
		t.Fatalf("expected 2 DNS entries, got %d", len(ar.DNS))
	}

	e0 := ar.DNS[0]
	if e0.Domain != "example.com" {
		t.Errorf("DNS[0].Domain: got %q, want %q", e0.Domain, "example.com")
	}
	// RecordTypeA must be shown as "IPv4" for user-facing clarity.
	if e0.Type != "IPv4" {
		t.Errorf("DNS[0].Type: got %q, want %q", e0.Type, "IPv4")
	}
	if e0.HasDivergence {
		t.Error("DNS[0].HasDivergence should be false when answers agree")
	}
	if len(e0.Answers) != 2 {
		t.Fatalf("DNS[0].Answers: expected 2, got %d", len(e0.Answers))
	}
	if e0.Answers[0].Source != "sys" {
		t.Errorf("DNS[0].Answers[0].Source: got %q, want %q", e0.Answers[0].Source, "sys")
	}
	if e0.Answers[0].RTT == "" {
		t.Error("DNS[0].Answers[0].RTT must not be empty")
	}

	e1 := ar.DNS[1]
	// RecordTypeAAAA must be shown as "IPv6" for user-facing clarity.
	if e1.Type != "IPv6" {
		t.Errorf("DNS[1].Type: got %q, want %q", e1.Type, "IPv6")
	}
	if !e1.HasDivergence {
		t.Error("DNS[1].HasDivergence should be true when AAAA answers differ")
	}
}

// TestBuildDNSNilWhenNone confirms AnnotatedReport.DNS is nil when no
// DNS comparisons were recorded (keeps JSON output clean).
func TestBuildDNSNilWhenNone(t *testing.T) {
	ar := buildAnnotated(t, buildSampleDiagReport())
	if ar.DNS != nil {
		t.Fatalf("expected DNS to be nil when no comparisons recorded, got %v", ar.DNS)
	}
}

// TestBuildDNSTypeDisplayNames verifies that the internal RecordType values
// are mapped to user-friendly display labels: A→IPv4, AAAA→IPv6, MX unchanged.
func TestBuildDNSTypeDisplayNames(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.AddDNSComparisons([]netprobe.DNSComparison{
		{Name: "example.com", Type: netprobe.RecordTypeA,
			Results: []netprobe.DNSAnswer{{Source: "sys", Values: []string{"1.2.3.4"}}}},
		{Name: "example.com", Type: netprobe.RecordTypeAAAA,
			Results: []netprobe.DNSAnswer{{Source: "sys", Values: []string{"::1"}}}},
		{Name: "example.com", Type: netprobe.RecordTypeMX,
			Results: []netprobe.DNSAnswer{{Source: "sys", Values: []string{"mail.example.com"}}}},
	})
	ar := buildAnnotated(t, dr)
	tests := []struct {
		idx  int
		want string
	}{
		{0, "IPv4"},
		{1, "IPv6"},
		{2, "MX"},
	}
	for _, tc := range tests {
		if ar.DNS[tc.idx].Type != tc.want {
			t.Errorf("DNS[%d].Type = %q, want %q", tc.idx, ar.DNS[tc.idx].Type, tc.want)
		}
	}
}

// ── test helpers ──────────────────────────────────────────────────────────

// TestBuildDNSLookupErrorPassThrough verifies that a DNSAnswer with LookupError set
// is faithfully propagated to DNSAnswerEntry.LookupError in the AnnotatedReport,
// allowing the UI to display the resolver failure reason to the user.
func TestBuildDNSLookupErrorPassThrough(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.AddDNSComparisons([]netprobe.DNSComparison{
		{
			Name: "notexist.example",
			Type: netprobe.RecordTypeA,
			Results: []netprobe.DNSAnswer{
				{Source: "sys", LookupError: "lookup notexist.example: no such host"},
				{Source: "cf", Values: []string{"1.1.1.1"}, RTT: 5 * time.Millisecond},
			},
		},
	})
	ar := buildAnnotated(t, dr)
	if len(ar.DNS) != 1 {
		t.Fatalf("expected 1 DNS entry, got %d", len(ar.DNS))
	}
	if len(ar.DNS[0].Answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(ar.DNS[0].Answers))
	}
	failing := ar.DNS[0].Answers[0]
	if failing.LookupError == "" {
		t.Error("expected LookupError to be propagated to DNSAnswerEntry")
	}
	if failing.Source != "sys" {
		t.Errorf("expected Source=%q, got %q", "sys", failing.Source)
	}
	ok := ar.DNS[0].Answers[1]
	if ok.LookupError != "" {
		t.Errorf("successful answer must have empty LookupError, got %q", ok.LookupError)
	}
}

// TestBuildDNSAllEmptyPassThrough verifies that when all resolvers return empty records
// (no values, no errors), DNSEntry.AllEmpty is set to true so the UI can distinguish
// "no records found" from "resolvers agree on non-empty results".
func TestBuildDNSAllEmptyPassThrough(t *testing.T) {
	dr := buildSampleDiagReport()
	dr.AddDNSComparisons([]netprobe.DNSComparison{
		// All resolvers return empty — AllEmpty should be true.
		{
			Name: "mx-less.example",
			Type: netprobe.RecordTypeMX,
			Results: []netprobe.DNSAnswer{
				{Source: "sys", Values: nil},
				{Source: "cf", Values: []string{}},
			},
		},
		// One resolver has values — AllEmpty must be false.
		{
			Name: "example.com",
			Type: netprobe.RecordTypeA,
			Results: []netprobe.DNSAnswer{
				{Source: "sys", Values: []string{"1.2.3.4"}},
				{Source: "cf", Values: []string{"1.2.3.4"}},
			},
		},
		// One resolver has LookupError — AllEmpty must be false (it's a failure, not empty).
		{
			Name: "broken.example",
			Type: netprobe.RecordTypeA,
			Results: []netprobe.DNSAnswer{
				{Source: "sys", LookupError: "no such host"},
				{Source: "cf", Values: []string{}},
			},
		},
	})
	ar := buildAnnotated(t, dr)
	if len(ar.DNS) != 3 {
		t.Fatalf("expected 3 DNS entries, got %d", len(ar.DNS))
	}
	// Entry 0: mx-less.example — all empty, no errors.
	if !ar.DNS[0].AllEmpty {
		t.Error("DNS[0] (all empty values, no errors): expected AllEmpty=true")
	}
	if ar.DNS[0].HasDivergence {
		t.Error("DNS[0]: all-empty must not be marked divergent")
	}
	// Entry 1: example.com — has values.
	if ar.DNS[1].AllEmpty {
		t.Error("DNS[1] (has values): expected AllEmpty=false")
	}
	// Entry 2: broken.example — LookupError present, not truly empty.
	if ar.DNS[2].AllEmpty {
		t.Error("DNS[2] (LookupError present): expected AllEmpty=false")
	}
}

// geoStubLocator returns a fixed non-zero GeoInfo for any IP so that
// HasLocation=true, allowing tests to exercise the geo branches in templates
// without requiring a real MaxMind database.
type geoStubLocator struct{}

func (geoStubLocator) LocateIP(ip string) (geo.GeoInfo, error) {
	return geo.GeoInfo{
		IP:          ip,
		Lat:         51.5,
		Lon:         -0.1,
		City:        "London",
		CountryCode: "GB",
		CountryName: "United Kingdom",
		ASN:         12345,
		OrgName:     "Test ISP",
		HasLocation: true,
	}, nil
}

func (geoStubLocator) Close() error { return nil }
