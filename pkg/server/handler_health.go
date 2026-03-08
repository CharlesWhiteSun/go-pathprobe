package server

import (
	"log/slog"
	"net/http"

	"go-pathprobe/pkg/version"
)

// HealthHandler serves GET /api/health.
// It reports liveness status and embeds the build-time version information
// injected via ldflags.
type HealthHandler struct {
	logger *slog.Logger
}

// ServeHTTP handles GET /api/health.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: version.Version,
		BuiltAt: version.BuildTime,
	})
}
