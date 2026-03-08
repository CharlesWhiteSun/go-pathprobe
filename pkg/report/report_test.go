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
