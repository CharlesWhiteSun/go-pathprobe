package main

import (
	"log/slog"
	"os"

	"go-pathprobe/pkg/cli"
	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
)

func main() {
	logger, levelVar := logging.NewLogger(slog.LevelInfo)

	dispatcher := diag.NewDispatcher(nil)
	for _, target := range diag.AllTargets {
		dispatcher.Register(target, diag.NewBasicRunner(target, logger))
	}

	rootCmd := cli.NewRootCommand(dispatcher, logger, levelVar)
	if err := rootCmd.Execute(); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}
