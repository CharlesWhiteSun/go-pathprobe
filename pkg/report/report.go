// Package report converts DiagReport data into table, JSON, and HTML outputs.
// Geo annotation is performed by the AnnotatedReport layer which wraps a
// *diag.DiagReport with geo.Locator results.
package report

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/netprobe"
)

// Writer renders an AnnotatedReport to an io.Writer.
type Writer interface {
	Write(w io.Writer, r *AnnotatedReport) error
}

// GeoAnnotation holds location and network ownership data for a single IP.
type GeoAnnotation struct {
	IP          string
	Lat         float64
	Lon         float64
	City        string
	CountryCode string
	CountryName string
	ASN         uint
	OrgName     string
	HasLocation bool
}

// PortEntry is a flat view of a netprobe.PortProbeResult for renderers.
type PortEntry struct {
	Port     int
	Sent     int
	Received int
	LossPct  float64
	MinRTT   string
	AvgRTT   string
	MaxRTT   string
}

// ProtoEntry is a flat view of a diag.ProtoResult for renderers.
type ProtoEntry struct {
	Protocol string
	Host     string
	Port     int
	OK       bool
	Summary  string
}

// AnnotatedReport is a DiagReport enriched with geo information.
type AnnotatedReport struct {
	Target      string
	Host        string
	GeneratedAt string

	PublicGeo GeoAnnotation
	TargetGeo GeoAnnotation

	Ports  []PortEntry
	Protos []ProtoEntry
}

// fmtDur formats a duration for display, showing "—" for zero values.
func fmtDur(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Microseconds()))
	}
	return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
}

func toPortEntry(r netprobe.PortProbeResult) PortEntry {
	return PortEntry{
		Port:     r.Port,
		Sent:     r.Stats.Sent,
		Received: r.Stats.Received,
		LossPct:  r.Stats.LossPct,
		MinRTT:   fmtDur(r.Stats.MinRTT),
		AvgRTT:   fmtDur(r.Stats.AvgRTT),
		MaxRTT:   fmtDur(r.Stats.MaxRTT),
	}
}

func toGeoAnnotation(info geo.GeoInfo) GeoAnnotation {
	return GeoAnnotation{
		IP:          info.IP,
		Lat:         info.Lat,
		Lon:         info.Lon,
		City:        info.City,
		CountryCode: info.CountryCode,
		CountryName: info.CountryName,
		ASN:         info.ASN,
		OrgName:     info.OrgName,
		HasLocation: info.HasLocation,
	}
}

// resolveTargetIP resolves the first IP for host (which may itself be an IP).
func resolveTargetIP(host string) string {
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	addrs, err := net.LookupHost(host)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

// Build converts a DiagReport into an AnnotatedReport using the provided
// geo.Locator for IP annotation.  A nil locator or NoopLocator leaves geo
// fields empty.
func Build(ctx context.Context, dr *diag.DiagReport, loc geo.Locator) (*AnnotatedReport, error) {
	ar := &AnnotatedReport{
		Target:      string(dr.Target),
		Host:        dr.Host,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	for _, p := range dr.Ports {
		ar.Ports = append(ar.Ports, toPortEntry(p))
	}
	for _, p := range dr.Protos {
		ar.Protos = append(ar.Protos, ProtoEntry{
			Protocol: p.Protocol,
			Host:     p.Host,
			Port:     p.Port,
			OK:       p.OK,
			Summary:  p.Summary,
		})
	}

	if loc == nil {
		return ar, nil
	}

	// Annotate public IP.
	if dr.PublicIP != "" {
		info, err := loc.LocateIP(dr.PublicIP)
		if err == nil {
			info.IP = dr.PublicIP
			ar.PublicGeo = toGeoAnnotation(info)
		}
	}

	// Annotate target host.
	if dr.Host != "" {
		targetIP := resolveTargetIP(dr.Host)
		if targetIP != "" {
			info, err := loc.LocateIP(targetIP)
			if err == nil {
				info.IP = targetIP
				ar.TargetGeo = toGeoAnnotation(info)
			}
		}
	}

	return ar, nil
}
