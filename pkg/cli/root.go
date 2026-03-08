package cli

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/logging"
	"go-pathprobe/pkg/netprobe"
	"go-pathprobe/pkg/report"
	"go-pathprobe/pkg/version"
)

// NewRootCommand constructs the CLI root with subcommands for diagnostics.
func NewRootCommand(dispatcher *diag.Dispatcher, logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
	opts := diag.GlobalOptions{MTRCount: diag.DefaultMTRCount, Timeout: 5 * time.Second, LogLevel: slog.LevelInfo}
	var logLevelFlag string

	rootCmd := &cobra.Command{
		Use:   "pathprobe",
		Short: "Network path and service diagnostic tool",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Update log level before executing any subcommand.
			level, err := logging.ParseLevel(logLevelFlag)
			if err != nil {
				return err
			}
			levelVar.Set(level)
			opts.LogLevel = level
			return opts.Validate()
		},
	}

	rootCmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "output results in JSON")
	rootCmd.PersistentFlags().StringVar(&opts.Report, "report", "", "write HTML report to this file path")
	rootCmd.PersistentFlags().IntVar(&opts.MTRCount, "mtr-count", diag.DefaultMTRCount, "probe count per hop for traceroute/MTR")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "info", "log level: debug, info, warn, error")
	rootCmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", 5*time.Second, "overall timeout per diagnostic")
	rootCmd.PersistentFlags().BoolVar(&opts.Insecure, "insecure", false, "skip TLS verification (use with caution)")
	rootCmd.PersistentFlags().StringVar(&opts.GeoDBCity, "geo-db-city", "", "path to GeoLite2-City.mmdb for location annotation")
	rootCmd.PersistentFlags().StringVar(&opts.GeoDBASN, "geo-db-asn", "", "path to GeoLite2-ASN.mmdb for ASN annotation")

	rootCmd.AddCommand(newDiagCommand(&opts, dispatcher, logger))
	rootCmd.AddCommand(newVersionCommand())

	return rootCmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show PathProbe version and build information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("PathProbe %s (built %s)\n", version.Version, version.BuildTime)
		},
	}
}

func newDiagCommand(opts *diag.GlobalOptions, dispatcher *diag.Dispatcher, logger *slog.Logger) *cobra.Command {
	diagCmd := &cobra.Command{
		Use:   "diag",
		Short: "Run diagnostics for web, mail, or file-transfer protocols",
	}

	for _, target := range diag.AllTargets {
		diagCmd.AddCommand(newTargetCommand(target, opts, dispatcher, logger))
	}

	return diagCmd
}

func newTargetCommand(target diag.Target, opts *diag.GlobalOptions, dispatcher *diag.Dispatcher, logger *slog.Logger) *cobra.Command {
	webOpts := diag.WebOptions{Domains: []string{"example.com"}}
	recordTypes := []string{"A", "AAAA", "MX"}
	netOpts := diag.NetworkOptions{Host: "example.com", Ports: diag.DefaultPorts(target)}
	smtpOpts := diag.SMTPOptions{}
	ftpOpts := diag.FTPOptions{}
	sftpOpts := diag.SFTPOptions{}

	cmd := &cobra.Command{
		Use:   target.String(),
		Short: fmt.Sprintf("Run %s diagnostics", target.String()),
		RunE: func(cmd *cobra.Command, _ []string) error {
			webTypes, err := netprobe.ParseRecordTypes(recordTypes)
			if err != nil {
				return err
			}

			// Create a report accumulator so runners can write structured results.
			diagReport := &diag.DiagReport{Target: target, Host: netOpts.Host}

			request := diag.Request{
				Target: target,
				Options: diag.Options{
					Global: *opts,
					Web:    webOpts,
					Net:    netOpts,
					SMTP:   smtpOpts,
					FTP:    ftpOpts,
					SFTP:   sftpOpts,
				},
				Report: diagReport,
			}
			if target == diag.TargetWeb {
				request.Options.Web.Types = webTypes
			}
			if err := dispatcher.Dispatch(cmd.Context(), request); err != nil {
				return err
			}
			logger.Info("diagnostic completed", "target", target)

			return writeReport(cmd, opts, diagReport)
		},
	}

	if target == diag.TargetWeb {
		cmd.Flags().StringSliceVar(&webOpts.Domains, "dns-domain", webOpts.Domains, "domains to compare across resolvers")
		cmd.Flags().StringSliceVar(&recordTypes, "dns-type", recordTypes, "record types to query (A, AAAA, MX)")
		cmd.Flags().StringVar(&webOpts.URL, "http-url", webOpts.URL, "HTTP/HTTPS URL for protocol probe")
	}

	// Network flags for connectivity/traceroute style probes
	cmd.Flags().StringVar(&netOpts.Host, "target-host", netOpts.Host, "host for connectivity probes")
	cmd.Flags().IntSliceVar(&netOpts.Ports, "port", netOpts.Ports, "ports to probe for reachability")

	if target == diag.TargetSMTP {
		cmd.Flags().StringVar(&smtpOpts.Domain, "smtp-domain", smtpOpts.Domain, "domain for MX lookup or EHLO")
		cmd.Flags().StringVar(&smtpOpts.Username, "smtp-user", smtpOpts.Username, "SMTP username for auth")
		cmd.Flags().StringVar(&smtpOpts.Password, "smtp-pass", smtpOpts.Password, "SMTP password or app password")
		cmd.Flags().StringVar(&smtpOpts.From, "smtp-from", smtpOpts.From, "MAIL FROM address")
		cmd.Flags().StringSliceVar(&smtpOpts.To, "smtp-to", smtpOpts.To, "RCPT TO addresses")
		cmd.Flags().BoolVar(&smtpOpts.UseTLS, "smtp-ssl", false, "use implicit SSL/TLS (SMTPS)")
		cmd.Flags().BoolVar(&smtpOpts.StartTLS, "smtp-starttls", true, "attempt STARTTLS when supported")
		cmd.Flags().StringSliceVar(&smtpOpts.AuthMethods, "smtp-auth-methods", nil, "auth mechanisms to try in order (PLAIN, LOGIN, XOAUTH2)")
		cmd.Flags().BoolVar(&smtpOpts.MXProbeAll, "smtp-mx-all", false, "probe all MX records for the domain")
	}

	if target == diag.TargetFTP {
		cmd.Flags().StringVar(&ftpOpts.Username, "ftp-user", ftpOpts.Username, "FTP username")
		cmd.Flags().StringVar(&ftpOpts.Password, "ftp-pass", ftpOpts.Password, "FTP password")
		cmd.Flags().BoolVar(&ftpOpts.UseTLS, "ftp-ssl", false, "use implicit FTPS (port 990)")
		cmd.Flags().BoolVar(&ftpOpts.AuthTLS, "ftp-auth-tls", false, "use explicit FTPS via AUTH TLS")
		cmd.Flags().BoolVar(&ftpOpts.RunLIST, "ftp-list", false, "attempt PASV + LIST after login")
	}

	if target == diag.TargetSFTP {
		cmd.Flags().StringVar(&sftpOpts.Username, "sftp-user", sftpOpts.Username, "SSH/SFTP username")
		cmd.Flags().StringVar(&sftpOpts.Password, "sftp-pass", sftpOpts.Password, "SSH/SFTP password")
		cmd.Flags().BoolVar(&sftpOpts.RunLS, "sftp-ls", false, "attempt to list remote default directory via SFTP subsystem")
	}

	return cmd
}

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
