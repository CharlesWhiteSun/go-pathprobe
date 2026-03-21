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
	// LocationPrecision mirrors geo.GeoInfo.LocationPrecision: "country" or "city".
	LocationPrecision string
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
// ErrorCategory holds the result of netprobe.ClassifyDNSLookupError and is empty
// when the resolver succeeded.
type DNSAnswerEntry struct {
	Source        string
	Values        []string
	RTT           string
	LookupError   string
	ErrorCategory string // "input" | "nxdomain" | "network" | "resolver" | "unknown" | ""
}

// DNSEntry represents the cross-resolver comparison result for one
// domain + record-type pair.
//
// Status semantics are mutually exclusive and checked in this priority order:
//
//	AllFailed     == true  → every resolver errored          (badge-fail, highest priority)
//	HasDivergence == true  → resolvers disagree              (badge-fail)
//	NoneFound     == true  → no records at all (errors+empty mix) (badge-warn)
//	AllEmpty      == true  → all resolvers found no records, no errors (badge-warn, subset of NoneFound)
//	otherwise              → resolvers agree on non-empty data (badge-ok)
//
// NoneFound is the canonical check in the renderer for the "no records" badge;
// AllEmpty is retained for backward-compatibility with existing tests.
//
// HintKey is the i18n key for a contextual explanation banner shown to the
// user when AllFailed is true.  It is empty when not all resolvers failed.
type DNSEntry struct {
	Domain        string
	Type          string
	HasDivergence bool
	AllEmpty      bool
	AllFailed     bool
	NoneFound     bool
	HintKey       string
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

// dnsHintKey selects an appropriate i18n hint key when all resolvers for an
// entry have failed.  If every answer shares the same actionable error category
// (input, nxdomain, network, or resolver) a category-specific key is returned;
// mixed categories or the "unknown" fallback both return "dns-hint-all-failed".
// An empty string is returned whenever at least one answer succeeded.
func dnsHintKey(answers []DNSAnswerEntry) string {
	if len(answers) == 0 {
		return ""
	}
	for _, a := range answers {
		if a.LookupError == "" {
			return ""
		}
	}
	// All answers are failures — determine dominant error category.
	first := answers[0].ErrorCategory
	for _, a := range answers[1:] {
		if a.ErrorCategory != first {
			return "dns-hint-all-failed"
		}
	}
	switch first {
	case string(netprobe.DNSErrInput), string(netprobe.DNSErrNXDomain),
		string(netprobe.DNSErrNetwork), string(netprobe.DNSErrResolver):
		return "dns-hint-" + first
	default:
		return "dns-hint-all-failed"
	}
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
		IP:                info.IP,
		Lat:               info.Lat,
		Lon:               info.Lon,
		City:              info.City,
		CountryCode:       info.CountryCode,
		CountryName:       info.CountryName,
		ASN:               info.ASN,
		OrgName:           info.OrgName,
		HasLocation:       info.HasLocation,
		LocationPrecision: info.LocationPrecision,
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

// wantsIPGeoAnnotation reports whether the web mode warrants PublicGeo and
// TargetGeo annotation.  Only WebModeAll (the legacy "all-in-one" empty mode)
// and WebModePublicIP perform a public-IP fetch, making IP-level geo sidebar
// decoration meaningful.  All other focused modes (dns, http, port, traceroute)
// suppress the IP-level geo sidebar to avoid misleading geographic context for
// operations that are not inherently IP-discovery oriented.
func wantsIPGeoAnnotation(mode diag.WebMode) bool {
	return mode == diag.WebModeAll || mode == diag.WebModePublicIP
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
			AllFailed:     comp.AllFailed(),
			HasDivergence: comp.HasDivergence(),
			AllEmpty:      comp.AllEmpty(),
			NoneFound:     comp.NoneFound(),
		}
		for _, ans := range comp.Results {
			cat := netprobe.ClassifyDNSLookupError(comp.Name, ans.LookupError)
			entry.Answers = append(entry.Answers, DNSAnswerEntry{
				Source:        ans.Source,
				Values:        ans.Values,
				RTT:           fmtDur(ans.RTT),
				LookupError:   ans.LookupError,
				ErrorCategory: string(cat),
			})
		}
		entry.HintKey = dnsHintKey(entry.Answers)
		ar.DNS = append(ar.DNS, entry)
	}

	if loc == nil {
		return ar, nil
	}

	// PublicGeo and TargetGeo only for IP-aware modes; route hop geo is unconditional.
	if wantsIPGeoAnnotation(dr.WebMode) {
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
	}

	return ar, nil
}
