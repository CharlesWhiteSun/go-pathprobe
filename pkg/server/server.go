package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
)

// Config holds HTTP server tuning parameters.
type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns production-ready defaults.
// WriteTimeout is generous because diagnostic runs may take up to 30 s.
func DefaultConfig() Config {
	return Config{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Server wraps net/http.Server and owns the API route registration.
// Construct it with New; do not copy after first use.
type Server struct {
	cfg     Config
	httpSrv *http.Server
	handler http.Handler
	logger  *slog.Logger
}

// New builds a Server with all API routes registered.
// The caller owns locator's lifecycle and must close it after Shutdown returns.
func New(cfg Config, dispatcher *diag.Dispatcher, locator geo.Locator, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("GET /api/health", &HealthHandler{logger: logger})
	mux.Handle("POST /api/diag", &DiagHandler{
		dispatcher: dispatcher,
		locator:    locator,
		logger:     logger,
	})
	// Static web UI — registered last; GET / acts as catch-all for all paths
	// not claimed by more specific API patterns above.
	mux.Handle("GET /", newStaticHandler())

	httpSrv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{cfg: cfg, httpSrv: httpSrv, handler: mux, logger: logger}
}

// Handler exposes the underlying http.Handler for use with httptest in tests.
func (s *Server) Handler() http.Handler { return s.handler }

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.cfg.Addr }

// Start binds and serves requests.  It blocks until the server exits and
// returns http.ErrServerClosed on graceful shutdown.
func (s *Server) Start() error {
	s.logger.Info("API server listening", "addr", s.cfg.Addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully drains in-flight requests within ctx's deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// writeJSON encodes v as indented JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// writeError sends a JSON-encoded ErrorResponse.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
