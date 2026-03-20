package cli

import (
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
	"go-pathprobe/pkg/server"
)

// NewRootCommand constructs the CLI root with subcommands for diagnostics.
//
// registrars maps each diagnostic target to its protocol-specific FlagRegistrar.
// Pass DefaultRegistrars() for the built-in defaults, or use
// app.BuildRegistrars(plugins) when protocol plugins are in use.
//
// optBuilder is an optional OptionsBuilder forwarded to the HTTP serve command.
// Pass nil to use the server's built-in option mapping.
func NewRootCommand(dispatcher *diag.Dispatcher, registrars map[diag.Target]FlagRegistrar, optBuilder server.OptionsBuilder, logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
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

	serveCmd := newServeCommand(dispatcher, &opts, optBuilder, logger, platformOpen)
	rootCmd.AddCommand(newDiagCommand(&opts, dispatcher, registrars, logger))
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(serveCmd)

	// When invoked with no subcommand (e.g. double-clicking the binary),
	// behave as if the user ran "pathprobe serve".
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		serveCmd.SetContext(cmd.Context())
		return serveCmd.RunE(serveCmd, args)
	}

	return rootCmd
}
