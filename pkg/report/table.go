package report

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const separator = "═══════════════════════════════════════════════════════════════"
const thinLine = "  ─────────────────────────────────────────────────────────────"

// TableWriter renders an AnnotatedReport as a human-readable table on stdout.
type TableWriter struct{}

// Write outputs the report as a formatted text table.
func (TableWriter) Write(w io.Writer, r *AnnotatedReport) error {
	fmt.Fprintln(w, separator)
	fmt.Fprintf(w, "PathProbe Diagnostic Report  ·  %s  ·  %s\n", r.Target, r.Host)
	fmt.Fprintf(w, "Generated: %s\n", r.GeneratedAt)
	fmt.Fprintln(w, separator)
	fmt.Fprintln(w)

	// Geo cards row.
	writeGeoCard(w, "CLIENT (PUBLIC IP)", r.PublicGeo)
	writeGeoCard(w, "TARGET HOST", r.TargetGeo)
	fmt.Fprintln(w)

	// Port connectivity table.
	if len(r.Ports) > 0 {
		fmt.Fprintln(w, "PORT CONNECTIVITY")
		fmt.Fprintln(w, thinLine)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  PORT\tSENT\tRECV\tLOSS%\tMIN RTT\tAVG RTT\tMAX RTT")
		for _, p := range r.Ports {
			lossStr := fmt.Sprintf("%.1f%%", p.LossPct)
			fmt.Fprintf(tw, "  %d\t%d\t%d\t%s\t%s\t%s\t%s\n",
				p.Port, p.Sent, p.Received, lossStr, p.MinRTT, p.AvgRTT, p.MaxRTT)
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	// Protocol results table.
	if len(r.Protos) > 0 {
		fmt.Fprintln(w, "PROTOCOL RESULTS")
		fmt.Fprintln(w, thinLine)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  PROTOCOL\tHOST\tPORT\tSTATUS\tSUMMARY")
		for _, p := range r.Protos {
			status := "OK"
			if !p.OK {
				status = "FAIL"
			}
			portStr := fmt.Sprintf("%d", p.Port)
			if p.Port == 0 {
				portStr = "—"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
				p.Protocol, p.Host, portStr, status, p.Summary)
		}
		tw.Flush()
	}

	fmt.Fprintln(w, separator)
	return nil
}

func writeGeoCard(w io.Writer, label string, g GeoAnnotation) {
	fmt.Fprintf(w, "  %-20s  ", label)
	if g.IP != "" {
		fmt.Fprintf(w, "%-18s", g.IP)
		parts := []string{}
		if g.City != "" {
			parts = append(parts, g.City)
		}
		if g.CountryCode != "" {
			parts = append(parts, g.CountryCode)
		}
		if g.OrgName != "" {
			parts = append(parts, fmt.Sprintf("ASN%d %s", g.ASN, g.OrgName))
		}
		if len(parts) > 0 {
			fmt.Fprintf(w, "(%s)", strings.Join(parts, " | "))
		}
	} else {
		fmt.Fprint(w, "—")
	}
	fmt.Fprintln(w)
}
