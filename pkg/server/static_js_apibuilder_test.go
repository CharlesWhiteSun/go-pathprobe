package server_test

import (
	"strings"
	"testing"
)

// ── Phase 4: traceroute API field assertions ──────────────────────────────

// TestStaticJS_WebModeTracerouteBuildOpts verifies that the embedded api-builder.js
// handles the "traceroute" mode in buildWebOpts() and forwards max_hops into
// the API request payload so the server's WebOptions.MaxHops is populated.
func TestStaticJS_WebModeTracerouteBuildOpts(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// The traceroute mode string constant must appear in buildWebOpts.
	if !strings.Contains(body, `'traceroute'`) {
		t.Error("api-builder.js: 'traceroute' mode string must appear in buildWebOpts")
	}
	// The max_hops JSON field must be written into the request opts.
	if !strings.Contains(body, "max_hops") {
		t.Error("api-builder.js: buildWebOpts must include max_hops in the traceroute mode branch")
	}
	// The traceroute sub-panel ID must exist in config.js TARGET_MODE_PANELS (data layer).
	cfgBody := fetchBody(t, newStaticHandler(t), "/config.js")
	if !strings.Contains(cfgBody, "web-fields-traceroute") {
		t.Error("config.js: TARGET_MODE_PANELS.web must include 'web-fields-traceroute' entry")
	}
}

// TestStaticJS_TracerouteTimeoutAutoCompute verifies that app.js contains the
// logic to auto-compute a traceroute-appropriate timeout before sending the
// diagnostic request, preventing spurious deadline-exceeded errors.
func TestStaticJS_TracerouteTimeoutAutoCompute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// The traceroute-specific timeout guard must be present.
	if !strings.Contains(body, "traceroute") {
		t.Error("api-builder.js: must contain 'traceroute' reference for mode-specific timeout logic")
	}
	// parseTimeoutSec must be defined to compare user timeout vs worst-case minimum.
	if !strings.Contains(body, "parseTimeoutSec") {
		t.Error("api-builder.js: parseTimeoutSec helper must be defined for timeout comparison")
	}
	// The auto-compute formula (maxHops * mtrCount * 2 + 15) must be present.
	if !strings.Contains(body, "maxHops * mtrCount * 2 + 15") {
		t.Error("api-builder.js: traceroute timeout auto-compute must use formula maxHops * mtrCount * 2 + 15")
	}
}

// TestStaticJS_WebPortModeReadsTextInput verifies that app.js handles the
// web/port mode using the shared text input (val('ports')) instead of the
// removed checkbox picker.  getWebPorts() must no longer exist in the codebase.
func TestStaticJS_WebPortModeReadsTextInput(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// getWebPorts() has been removed; buildRequest reads _val('ports') for web/port.
	if strings.Contains(body, "function getWebPorts(") {
		t.Error("api-builder.js: getWebPorts() must be removed; web/port mode now uses the shared text input")
	}
	// buildRequest must use _webModesWithPorts() (runtime-resolved) to decide whether to read ports.
	if !strings.Contains(body, "_webModesWithPorts().includes(mode)") {
		t.Error("api-builder.js: buildRequest must guard web port reading with _webModesWithPorts().includes(mode)")
	}
	// ports-text-group: the shared text input read path for web/port mode must be documented.
	if !strings.Contains(body, "ports-text-group") {
		t.Error("api-builder.js: comment must reference 'ports-text-group' to document the shared text input read path")
	}
	// The removed picker elements must not be referenced in JS logic.
	if strings.Contains(body, "getElementById('port-other-cb')") {
		t.Error("api-builder.js: port-other-cb has been removed and must not be referenced")
	}
	if strings.Contains(body, "getElementById('port-other-num')") {
		t.Error("api-builder.js: port-other-num has been removed and must not be referenced")
	}
}

// TestStaticJS_BuildRequestFunction verifies that api-builder.js defines
// buildRequest() and assembles a payload with the expected { target, options }
// top-level structure, and that it is the sole function exported via the public API.
func TestStaticJS_BuildRequestFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// buildRequest must be defined as the main entry point.
	if !strings.Contains(body, "function buildRequest(") {
		t.Error("api-builder.js: buildRequest() function must be defined")
	}
	// The returned payload must contain both top-level fields.
	if !strings.Contains(body, "{ target, options: opts }") {
		t.Error("api-builder.js: buildRequest must return { target, options: opts } payload")
	}
	// Only buildRequest is exported — internal helpers remain private to the IIFE.
	needle := "PathProbe.ApiBuilder = {"
	exportStart := strings.Index(body, needle)
	if exportStart == -1 {
		t.Fatalf("api-builder.js: PathProbe.ApiBuilder = { ... } export block not found")
	}
	exportEnd := strings.Index(body[exportStart:], "};")
	if exportEnd == -1 {
		t.Fatalf("api-builder.js: closing }; not found after PathProbe.ApiBuilder export block")
	}
	exportBlock := body[exportStart : exportStart+exportEnd+2]
	if !strings.Contains(exportBlock, "buildRequest") {
		t.Error("api-builder.js: PathProbe.ApiBuilder must export buildRequest")
	}
	// Internal helpers must NOT be exported.
	for _, priv := range []string{"buildWebOpts", "buildSMTPOpts", "buildFTPOpts", "buildSFTPOpts", "parseTimeoutSec"} {
		if strings.Contains(exportBlock, priv) {
			t.Errorf("api-builder.js: %s must remain private (not exported in PathProbe.ApiBuilder)", priv)
		}
	}
}

// TestStaticJS_DNSModeEmitsEmptyHost verifies that buildRequest() sends an
// empty host value when the active web mode is in WEB_MODES_HIDE_HOST (i.e. dns).
// This prevents the backend from performing a spurious target-host geo lookup
// and showing a geo map to the user when they are running a DNS comparison.
func TestStaticJS_DNSModeEmitsEmptyHost(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// _webModesHideHost() config alias must be declared in the builder.
	if !strings.Contains(body, "_webModesHideHost()") {
		t.Error("api-builder.js: _webModesHideHost() alias must be declared")
	}
	// The host field must be conditionally suppressed when the mode is in WEB_MODES_HIDE_HOST.
	// Check that the ternary guard pattern exists (empty string for host in hide-host modes).
	if !strings.Contains(body, "_webModesHideHost().includes(") {
		t.Error("api-builder.js: buildRequest must use _webModesHideHost().includes() to suppress host in dns mode")
	}
	// The suppressed value must yield an empty string, not send a hidden field value.
	if !strings.Contains(body, "? ''") && !strings.Contains(body, `? ""`) {
		t.Error("api-builder.js: host suppression must produce an empty string ('') in dns mode")
	}
}
