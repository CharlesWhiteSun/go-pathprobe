package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"go-pathprobe/pkg/app"
	"go-pathprobe/pkg/logging"
	"go-pathprobe/pkg/syscheck"
)

func main() {
	logger, levelVar := logging.NewLogger(slog.LevelInfo)

	// Detect raw ICMP socket availability and warn the user when unavailable.
	avail := syscheck.RawICMPChecker{}.Check()
	if !avail.Available {
		logger.Warn(avail.Notice(), "error", avail.Err)
	}

	deps := app.Deps{
		Logger:    logger,
		LevelVar:  levelVar,
		HTTP:      &http.Client{Timeout: 5 * time.Second},
		ICMPAvail: avail.Available,
	}

	if err := app.Run(deps, app.AllPlugins); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}
