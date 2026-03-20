package app

import (
	"log/slog"
	"net/http"
)

// Deps holds shared infrastructure that protocol-specific runner factories and
// the application entrypoint require.  It is assembled once in main and
// threaded through every ProtocolPlugin.NewRunner call.
type Deps struct {
	Logger    *slog.Logger
	LevelVar  *slog.LevelVar
	HTTP      *http.Client
	ICMPAvail bool // true when raw ICMP sockets are available on this host
}
