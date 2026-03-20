package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
)

// newDiagCommand builds the 'diag' subcommand container. Each supported
// diagnostic target is registered as a child command via newTargetCommand.
func newDiagCommand(opts *diag.GlobalOptions, dispatcher *diag.Dispatcher, registrars map[diag.Target]FlagRegistrar, logger *slog.Logger) *cobra.Command {
	diagCmd := &cobra.Command{
		Use:   "diag",
		Short: "Run diagnostics for web, mail, or file-transfer protocols",
	}
	for _, target := range diag.AllTargets {
		diagCmd.AddCommand(newTargetCommand(target, opts, dispatcher, registrars, logger))
	}
	return diagCmd
}

// newTargetCommand builds a subcommand for a single diagnostic target.
// Shared network flags (--target-host, --port) are registered for all targets.
// Protocol-specific flags are injected via the registrars map so that adding a
// new target requires only a new ProtocolPlugin — this function stays closed.
func newTargetCommand(target diag.Target, opts *diag.GlobalOptions, dispatcher *diag.Dispatcher, registrars map[diag.Target]FlagRegistrar, logger *slog.Logger) *cobra.Command {
	options := diag.Options{
		Net: diag.NetworkOptions{
			Host:  "example.com",
			Ports: diag.DefaultPorts(target),
		},
	}

	var preparer OptionsPreparer

	cmd := &cobra.Command{
		Use:   target.String(),
		Short: fmt.Sprintf("Run %s diagnostics", target.String()),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Merge the root-level GlobalOptions (already validated by
			// PersistentPreRunE) into the per-invocation options struct.
			options.Global = *opts

			// Run any deferred option transformation (e.g. parse record types).
			if preparer != nil {
				if err := preparer(&options); err != nil {
					return err
				}
			}

			diagReport := &diag.DiagReport{Target: target, Host: options.Net.Host}
			request := diag.Request{
				Target:  target,
				Options: options,
				Report:  diagReport,
			}
			if err := dispatcher.Dispatch(cmd.Context(), request); err != nil {
				return err
			}
			logger.Info("diagnostic completed", "target", target)
			return writeReport(cmd, opts, diagReport)
		},
	}

	// Network flags shared across all targets.
	cmd.Flags().StringVar(&options.Net.Host, "target-host", options.Net.Host, "host for connectivity probes")
	cmd.Flags().IntSliceVar(&options.Net.Ports, "port", options.Net.Ports, "ports to probe for reachability")

	// Delegate protocol-specific flags to the registrar, if one is registered.
	if registrar, ok := registrars[target]; ok {
		preparer = registrar(cmd, &options)
	}

	return cmd
}
