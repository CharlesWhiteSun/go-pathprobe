package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
)

// NewRootCommand constructs the CLI root with subcommands for diagnostics.
func NewRootCommand(dispatcher *diag.Dispatcher, logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
	opts := diag.GlobalOptions{MTRCount: diag.DefaultMTRCount, LogLevel: slog.LevelInfo}
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
	return &cobra.Command{
		Use:   target.String(),
		Short: fmt.Sprintf("Run %s diagnostics", target.String()),
		RunE: func(cmd *cobra.Command, _ []string) error {
			request := diag.Request{
				Target: target,
				Options: diag.Options{
					Global: *opts,
				},
			}
			if err := dispatcher.Dispatch(cmd.Context(), request); err != nil {
				return err
			}
			logger.Info("diagnostic completed", "target", target)
			return nil
		},
	}
}
