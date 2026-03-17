package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/report"
)

// writeReport builds an AnnotatedReport from diagReport and outputs it using
// the writer selected by opts (JSON, HTML file, or default table on stdout).
func writeReport(cmd *cobra.Command, opts *diag.GlobalOptions, diagReport *diag.DiagReport) error {
	locator, err := geo.Open(opts.GeoDBCity, opts.GeoDBASN)
	if err != nil {
		// Non-fatal: log the warning and proceed with NoopLocator.
		fmt.Fprintf(os.Stderr, "warning: geo db unavailable: %v\n", err)
		locator = geo.NoopLocator{}
	}
	defer locator.Close()

	ar, err := report.Build(cmd.Context(), diagReport, locator)
	if err != nil {
		return fmt.Errorf("build report: %w", err)
	}

	// HTML file report.
	if opts.Report != "" {
		hw := report.HTMLWriter{}
		if err := hw.WriteFile(opts.Report, ar); err != nil {
			return fmt.Errorf("write HTML report: %w", err)
		}
	}

	// JSON or table output to stdout.
	if opts.JSON {
		jw := report.JSONWriter{}
		return jw.Write(os.Stdout, ar)
	}
	tw := report.TableWriter{}
	return tw.Write(os.Stdout, ar)
}
