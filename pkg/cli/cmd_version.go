package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/version"
)

// newVersionCommand builds the 'version' subcommand.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show PathProbe version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("PathProbe %s\n", version.Version)
		},
	}
}
