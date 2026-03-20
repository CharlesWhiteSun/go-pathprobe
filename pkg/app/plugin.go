package app

import (
	"go-pathprobe/pkg/cli"
	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/server"
)

// ProtocolPlugin bundles everything a diagnostic protocol needs to register
// itself with the application.  Adding a new protocol requires only creating a
// new ProtocolPlugin and including it in the plugin list — no other code needs
// to change (Open/Closed Principle).
type ProtocolPlugin struct {
	// Target identifies the diagnostic domain this plugin handles.
	Target diag.Target

	// NewRunner constructs the protocol's runner from shared infrastructure.
	// It is called once per application startup by BuildDispatcher.
	NewRunner func(deps Deps) diag.Runner

	// RegisterCLI registers protocol-specific CLI flags on the target's cobra
	// command.  nil means the target uses only the shared network flags
	// (--target-host, --port).
	RegisterCLI cli.FlagRegistrar

	// BuildOptions converts a server API request into fully-typed diag.Options
	// for this protocol.  nil means only global and network options are applied
	// (the default server.BuildGlobalOptions fallback is used).
	BuildOptions func(req server.ReqOptions) (diag.Options, error)
}
