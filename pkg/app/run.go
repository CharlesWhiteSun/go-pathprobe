package app

import (
	"go-pathprobe/pkg/cli"
	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
	"go-pathprobe/pkg/server"
	"log/slog"
)

// BuildDispatcher creates a new Dispatcher and registers each plugin's runner.
// Plugins with a nil NewRunner are silently skipped.
func BuildDispatcher(deps Deps, plugins []ProtocolPlugin) *diag.Dispatcher {
	d := diag.NewDispatcher(nil)
	for _, p := range plugins {
		if p.NewRunner != nil {
			d.Register(p.Target, p.NewRunner(deps))
		}
	}
	return d
}

// BuildRegistrars extracts CLI flag registrars from the plugin set.
// The returned map is safe to pass directly to cli.NewRootCommand.
// Plugins with a nil RegisterCLI are not included (they use only shared flags).
func BuildRegistrars(plugins []ProtocolPlugin) map[diag.Target]cli.FlagRegistrar {
	m := make(map[diag.Target]cli.FlagRegistrar, len(plugins))
	for _, p := range plugins {
		if p.RegisterCLI != nil {
			m[p.Target] = p.RegisterCLI
		}
	}
	return m
}

// BuildOptionsFunc creates a server.OptionsBuilder from the plugin set.
// When the request target matches a plugin's BuildOptions, it is called;
// otherwise server.BuildGlobalOptions is used as the fallback (global + net
// options only, no protocol-specific fields).
func BuildOptionsFunc(plugins []ProtocolPlugin) server.OptionsBuilder {
	builders := make(map[diag.Target]func(server.ReqOptions) (diag.Options, error), len(plugins))
	for _, p := range plugins {
		if p.BuildOptions != nil {
			builders[p.Target] = p.BuildOptions
		}
	}
	return func(target diag.Target, req server.ReqOptions) (diag.Options, error) {
		if fn, ok := builders[target]; ok {
			return fn(req)
		}
		opts := server.BuildGlobalOptions(req)
		if n := req.Net; n != nil {
			opts.Net.Host = n.Host
			opts.Net.Ports = n.Ports
		}
		return opts, nil
	}
}

// Run wires all plugins and starts the application by executing the root
// cobra command.  It is the single entry point called from main.
func Run(deps Deps, plugins []ProtocolPlugin) error {
	if deps.Logger == nil {
		deps.Logger, deps.LevelVar = logging.NewLogger(slog.LevelInfo)
	}

	dispatcher := BuildDispatcher(deps, plugins)
	registrars := BuildRegistrars(plugins)
	optBuilder := BuildOptionsFunc(plugins)

	rootCmd := cli.NewRootCommand(dispatcher, registrars, optBuilder, deps.Logger, deps.LevelVar)
	return rootCmd.Execute()
}
