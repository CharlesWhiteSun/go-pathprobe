// Package report converts DiagReport data into table, JSON, and HTML outputs.
// Geo annotation is performed by the AnnotatedReport layer which wraps a
// *diag.DiagReport with geo.IPLocator results.
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

// HopEntry is a flat, geo-annotated view of a single traceroute hop.
// IP is empty when the hop timed out ("???"); in that case all geo fields
// are also empty.
type HopEntry struct {
	TTL      int
	IP       string
	Hostname string
	ASN      uint
	Country  string
	AvgRTT   string
	LossPct  float64
	HasGeo   bool
	Lat      float64
	Lon      float64
}

// DNSAnswerEntry is one resolver's answer for a domain + record type.
// LookupError is non-empty when the resolver failed; Values will be nil in that case.
type DNSAnswerEntry struct {
	Source      string
	Values      []string
	RTT         string
	LookupError string
}

// DNSEntry represents the cross-resolver comparison result for one
// domain + record-type pair.
//
// Status semantics (mutually exclusive, checked in order):
//
//	HasDivergence == true  → resolvers disagree (badge-fail)
//	AllEmpty      == true  → all resolvers found no records (badge-warn)
//	otherwise             → resolvers agree on a non-empty result (badge-ok)
type DNSEntry struct {
	Domain        string
	Type          string
	HasDivergence bool
	AllEmpty      bool
	Answers       []DNSAnswerEntry
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
	Route  []HopEntry
	DNS    []DNSEntry
}

// dnsTypeDisplayName maps an internal DNS RecordType to a user-friendly label.
// "A" is displayed as "IPv4" and "AAAA" as "IPv6" so the UI is consistent with
// the Record Types labels shown in the form.  Other types (e.g. MX) are kept as-is.
func dnsTypeDisplayName(t netprobe.RecordType) string {
	switch t {
	case netprobe.RecordTypeA:
		return "IPv4"
	case netprobe.RecordTypeAAAA:
		return "IPv6"
	default:
		return string(t)
	}
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

// toHopEntry converts a single netprobe.HopResult into a geo-annotated HopEntry.
// If loc is nil or hop.IP is empty (timed-out hop), geo fields are left at zero values.
func toHopEntry(hop netprobe.HopResult, loc geo.IPLocator) HopEntry {
	entry := HopEntry{
		TTL:      hop.TTL,
		IP:       hop.IP,
		Hostname: hop.Hostname,
		AvgRTT:   fmtDur(hop.Stats.AvgRTT),
		LossPct:  hop.Stats.LossPct,
	}
	if hop.IP == "" || loc == nil {
		return entry
	}
	info, err := loc.LocateIP(hop.IP)
	if err == nil {
		entry.ASN = info.ASN
		entry.Country = info.CountryCode
		entry.HasGeo = info.HasLocation
		entry.Lat = info.Lat
		entry.Lon = info.Lon
	}
	return entry
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
// geo.IPLocator for IP annotation.  A nil locator or NoopLocator leaves geo
// fields empty.
func Build(ctx context.Context, dr *diag.DiagReport, loc geo.IPLocator) (*AnnotatedReport, error) {
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

	// Process route hops (geo annotation applied below if loc != nil).
	if dr.Route != nil {
		for _, hop := range dr.Route.Hops {
			ar.Route = append(ar.Route, toHopEntry(hop, loc))
		}
	}

	// Flatten DNS comparison results into renderer-friendly entries.
	for _, comp := range dr.DNSComparisons {
		entry := DNSEntry{
			Domain:        comp.Name,
			Type:          dnsTypeDisplayName(comp.Type),
			HasDivergence: comp.HasDivergence(),
			AllEmpty:      comp.AllEmpty(),
		}
		for _, ans := range comp.Results {
			entry.Answers = append(entry.Answers, DNSAnswerEntry{
				Source:      ans.Source,
				Values:      ans.Values,
				RTT:         fmtDur(ans.RTT),
				LookupError: ans.LookupError,
			})
		}
		ar.DNS = append(ar.DNS, entry)
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
