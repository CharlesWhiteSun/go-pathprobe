package cli

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
	"go-pathprobe/pkg/netprobe"
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
	rootCmd.PersistentFlags().StringVar(&opts.Report, "report", "", "output report path (optional)")
	rootCmd.PersistentFlags().IntVar(&opts.MTRCount, "mtr-count", diag.DefaultMTRCount, "probe count per hop for traceroute/MTR")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "info", "log level: debug, info, warn, error")
	rootCmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", 5*time.Second, "overall timeout per diagnostic")
	rootCmd.PersistentFlags().BoolVar(&opts.Insecure, "insecure", false, "skip TLS verification (use with caution)")

	rootCmd.AddCommand(newDiagCommand(&opts, dispatcher, logger))

	return rootCmd
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

	cmd := &cobra.Command{
		Use:   target.String(),
		Short: fmt.Sprintf("Run %s diagnostics", target.String()),
		RunE: func(cmd *cobra.Command, _ []string) error {
			webTypes, err := netprobe.ParseRecordTypes(recordTypes)
			if err != nil {
				return err
			}
			request := diag.Request{
				Target: target,
				Options: diag.Options{
					Global: *opts,
					Web:    webOpts,
					Net:    netOpts,
					SMTP:   smtpOpts,
				},
			}
			if target == diag.TargetWeb {
				request.Options.Web.Types = webTypes
			}
			if err := dispatcher.Dispatch(cmd.Context(), request); err != nil {
				return err
			}
			logger.Info("diagnostic completed", "target", target)
			return nil
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

	return cmd
}
