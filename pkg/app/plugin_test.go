package app_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"go-pathprobe/pkg/app"
	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/server"
)

// ---- test helpers -------------------------------------------------------

func silentDeps() app.Deps {
	return app.Deps{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		HTTP:      &http.Client{},
		ICMPAvail: false,
	}
}

// noopRunner is a Runner that always succeeds without I/O.
type noopRunner struct{}

func (noopRunner) Run(_ context.Context, _ diag.Request) error { return nil }

// ---- BuildDispatcher tests ---------------------------------------------

func TestBuildDispatcher_RegistersAllPlugins(t *testing.T) {
	deps := silentDeps()
	plugins := []app.ProtocolPlugin{
		{Target: diag.TargetIMAP, NewRunner: func(_ app.Deps) diag.Runner { return noopRunner{} }},
		{Target: diag.TargetPOP, NewRunner: func(_ app.Deps) diag.Runner { return noopRunner{} }},
	}
	d := app.BuildDispatcher(deps, plugins)

	for _, target := range []diag.Target{diag.TargetIMAP, diag.TargetPOP} {
		if err := d.Dispatch(context.Background(), diag.Request{Target: target}); err != nil {
			t.Errorf("expected runner for %s, got: %v", target, err)
		}
	}
}

func TestBuildDispatcher_SkipsNilRunner(t *testing.T) {
	deps := silentDeps()
	plugins := []app.ProtocolPlugin{
		{Target: diag.TargetIMAP, NewRunner: nil},
	}
	d := app.BuildDispatcher(deps, plugins)

	err := d.Dispatch(context.Background(), diag.Request{Target: diag.TargetIMAP})
	if err == nil {
		t.Error("expected ErrRunnerNotFound for plugin with nil NewRunner, got nil")
	}
}

func TestBuildDispatcher_InjectsDepsIntoRunner(t *testing.T) {
	var capturedDeps app.Deps
	deps := silentDeps()
	deps.ICMPAvail = true

	plugins := []app.ProtocolPlugin{
		{
			Target: diag.TargetIMAP,
			NewRunner: func(d app.Deps) diag.Runner {
				capturedDeps = d
				return noopRunner{}
			},
		},
	}
	app.BuildDispatcher(deps, plugins)

	if !capturedDeps.ICMPAvail {
		t.Error("expected deps.ICMPAvail=true to be forwarded to NewRunner factory")
	}
}

// ---- BuildRegistrars tests --------------------------------------------

func TestBuildRegistrars_OnlyIncludesNonNilRegistrars(t *testing.T) {
	plugins := []app.ProtocolPlugin{
		{Target: diag.TargetWeb, RegisterCLI: nil},
		{Target: diag.TargetIMAP, RegisterCLI: nil},
	}
	result := app.BuildRegistrars(plugins)
	if len(result) != 0 {
		t.Errorf("expected empty registrar map when all RegisterCLI are nil, got %d entries", len(result))
	}
}

func TestBuildRegistrars_CountMatchesNonNilRegistrars(t *testing.T) {
	plugins := app.AllPlugins
	result := app.BuildRegistrars(plugins)

	// Web, SMTP, FTP, SFTP have registrars; IMAP, POP do not.
	expected := 4
	if len(result) != expected {
		t.Errorf("expected %d registrar entries, got %d", expected, len(result))
	}
}

func TestBuildRegistrars_IMAPAndPOPAbsent(t *testing.T) {
	result := app.BuildRegistrars(app.AllPlugins)
	for _, target := range []diag.Target{diag.TargetIMAP, diag.TargetPOP} {
		if _, ok := result[target]; ok {
			t.Errorf("unexpected registrar for %s; these targets use only shared flags", target)
		}
	}
}

// ---- BuildOptionsFunc tests -------------------------------------------

func TestBuildOptionsFunc_DelegatesToPlugin(t *testing.T) {
	called := false
	plugins := []app.ProtocolPlugin{
		{
			Target: diag.TargetSMTP,
			BuildOptions: func(req server.ReqOptions) (diag.Options, error) {
				called = true
				return server.BuildGlobalOptions(req), nil
			},
		},
	}
	builder := app.BuildOptionsFunc(plugins)
	if _, err := builder(diag.TargetSMTP, server.ReqOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected plugin's BuildOptions to be called for TargetSMTP")
	}
}

func TestBuildOptionsFunc_FallbackForUnknownTarget(t *testing.T) {
	builder := app.BuildOptionsFunc([]app.ProtocolPlugin{}) // no builders
	opts, err := builder(diag.TargetIMAP, server.ReqOptions{MTRCount: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Global.MTRCount != 7 {
		t.Errorf("expected MTRCount=7 from global defaults, got %d", opts.Global.MTRCount)
	}
}

func TestBuildOptionsFunc_NetOptionsAppliedInFallback(t *testing.T) {
	builder := app.BuildOptionsFunc([]app.ProtocolPlugin{})
	host := "example.com"
	opts, err := builder(diag.TargetIMAP, server.ReqOptions{
		Net: &server.ReqNet{Host: host, Ports: []int{143}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Net.Host != host {
		t.Errorf("expected Net.Host=%q in fallback, got %q", host, opts.Net.Host)
	}
	if len(opts.Net.Ports) != 1 || opts.Net.Ports[0] != 143 {
		t.Errorf("expected IMAP port 143 in fallback, got %v", opts.Net.Ports)
	}
}

// ---- AllPlugins invariant tests ----------------------------------------

func TestAllPlugins_UniqueTargets(t *testing.T) {
	seen := make(map[diag.Target]bool, len(app.AllPlugins))
	for _, p := range app.AllPlugins {
		if seen[p.Target] {
			t.Errorf("duplicate target %q in AllPlugins", p.Target)
		}
		seen[p.Target] = true
	}
}

func TestAllPlugins_AllHaveNonNilNewRunner(t *testing.T) {
	for _, p := range app.AllPlugins {
		if p.NewRunner == nil {
			t.Errorf("plugin for target %q has nil NewRunner", p.Target)
		}
	}
}

func TestAllPlugins_CoversAllDiagTargets(t *testing.T) {
	pluginTargets := make(map[diag.Target]bool, len(app.AllPlugins))
	for _, p := range app.AllPlugins {
		pluginTargets[p.Target] = true
	}
	for _, target := range diag.AllTargets {
		if !pluginTargets[target] {
			t.Errorf("diag.AllTargets contains %q but no matching entry in AllPlugins", target)
		}
	}
}

func TestAllPlugins_WebSMTPFTPSFTPHaveRegistrarAndBuildOptions(t *testing.T) {
	needsFull := map[diag.Target]bool{
		diag.TargetWeb:  true,
		diag.TargetSMTP: true,
		diag.TargetFTP:  true,
		diag.TargetSFTP: true,
	}
	for _, p := range app.AllPlugins {
		if !needsFull[p.Target] {
			continue
		}
		if p.RegisterCLI == nil {
			t.Errorf("plugin %q must have non-nil RegisterCLI", p.Target)
		}
		if p.BuildOptions == nil {
			t.Errorf("plugin %q must have non-nil BuildOptions", p.Target)
		}
	}
}

func TestAllPlugins_BuildDispatcher_AllTargetsReachable(t *testing.T) {
	deps := silentDeps()
	d := app.BuildDispatcher(deps, app.AllPlugins)

	for _, target := range diag.AllTargets {
		if err := d.Dispatch(context.Background(), diag.Request{Target: target}); err == diag.ErrRunnerNotFound {
			t.Errorf("no runner registered for target %q after BuildDispatcher", target)
		}
	}
}
