package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/server"
	"go-pathprobe/pkg/store"
)

// newServeCommand builds the 'serve' subcommand that starts the embedded
// HTTP REST API server. It reuses the persistent --geo-db-city / --geo-db-asn
// flags already defined on the root command via opts.
//
// optBuilder is an optional server.OptionsBuilder derived from protocol plugins.
// When nil, the server uses its built-in option mapping (pkg/server.buildOptions).
//
// opener is called with the server URL just before srv.Start() so the browser
// opens while the server is warming up. Pass platformOpen for production;
// tests may inject a no-op or recording function.
func newServeCommand(dispatcher *diag.Dispatcher, opts *diag.GlobalOptions, optBuilder server.OptionsBuilder, logger *slog.Logger, opener func(string) error) *cobra.Command {
	cfg := server.DefaultConfig()
	var openBrowser bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the embedded PathProbe HTTP REST API server",
		Long: `Starts an HTTP server that exposes PathProbe diagnostics as a REST API.

Endpoints:
  GET  /api/health        — liveness probe (returns version + build time)
  POST /api/diag          — run a diagnostic and receive a JSON AnnotatedReport
  POST /api/diag/stream   — run a diagnostic with real-time SSE progress events

Geo annotation uses the --geo-db-city / --geo-db-asn flags from the root command.
The server shuts down gracefully on SIGINT (Ctrl-C).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			locator, err := geo.AutoLocator(opts.GeoDBCity, opts.GeoDBASN)
			if err != nil {
				logger.Warn("geo db unavailable, geo annotation disabled", "error", err)
				locator = geo.NoopLocator{}
			}
			defer locator.Close()

			st := store.NewMemoryStore(store.DefaultMaxHistory)
			var srvOpts []server.ServerOption
			if optBuilder != nil {
				srvOpts = append(srvOpts, server.WithOptionsBuilder(optBuilder))
			}
			srv := server.New(cfg, dispatcher, locator, st, logger, srvOpts...)
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer stop()

			go func() {
				<-ctx.Done()
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := srv.Shutdown(shutCtx); err != nil {
					logger.Warn("server shutdown error", "error", err)
				}
			}()

			// Launch the browser in a goroutine so srv.Start() is not delayed.
			// A brief sleep gives the HTTP listener time to bind before the
			// browser sends its first request.
			if openBrowser {
				url := serverURL(cfg.Addr)
				logger.Info("opening browser", "url", url)
				go func() {
					time.Sleep(300 * time.Millisecond)
					if err := opener(url); err != nil {
						logger.Warn("failed to open browser", "url", url, "error", err)
					}
				}()
			}

			if err := srv.Start(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("server error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.Addr, "addr", cfg.Addr,
		"listen address (host:port), e.g. :8080 or 127.0.0.1:9090")
	cmd.Flags().DurationVar(&cfg.WriteTimeout, "write-timeout", cfg.WriteTimeout,
		"HTTP write timeout (should be >= max diagnostic duration)")
	cmd.Flags().BoolVar(&openBrowser, "open", true,
		"open the web UI in the default browser on startup (use --open=false to disable)")

	return cmd
}
