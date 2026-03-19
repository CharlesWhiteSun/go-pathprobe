package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticJS_BrandSystemRemoved verifies that the brand style management
// system has been removed from app.js now that the logo style is fixed.
func TestStaticJS_BrandSystemRemoved(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	for _, absent := range []string{
		"BRAND_STYLES",
		"toggleBrandPicker",
		"initBrandStyle",
	} {
		if strings.Contains(body, absent) {
			t.Errorf("app.js: brand system symbol %q must not be present", absent)
		}
	}
}

// TestStaticJS_AppJsIsAssemblyEntryOnly verifies that app.js contains no
// function declarations — it must be a pure assembly entry point that only
// calls module APIs from a single DOMContentLoaded handler.
//
// All business logic must live in dedicated modules (form.js, theme.js,
// locale.js, api-client.js, history.js, etc.).  A stray function declaration
// in app.js indicates logic that should have been extracted.
func TestStaticJS_AppJsIsAssemblyEntryOnly(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// No function declarations — use arrow syntax in DOMContentLoaded instead.
	if strings.Contains(body, "function ") {
		t.Error("app.js: must not contain any 'function ' declarations — " +
			"app.js is a pure assembly entry point; extract logic into a dedicated module")
	}
	// Must wire up the DOMContentLoaded bootstrap.
	if !strings.Contains(body, "DOMContentLoaded") {
		t.Error("app.js: must register a DOMContentLoaded listener to bootstrap modules")
	}
	// Must call each module's init API.
	for _, call := range []string{
		"PathProbe.Form.init()",
		"PathProbe.ApiClient.fetchVersion()",
		"PathProbe.History.loadHistory()",
		"PathProbe.Theme.initTheme()",
		"PathProbe.Locale.initLocale()",
	} {
		if !strings.Contains(body, call) {
			t.Errorf("app.js: DOMContentLoaded must call %q", call)
		}
	}
}

// TestStaticJS_AllModulesServed is an integration test that verifies every
// JavaScript module referenced by index.html is served with HTTP 200 and a
// JavaScript Content-Type by the static handler.
func TestStaticJS_AllModulesServed(t *testing.T) {
	h := newStaticHandler(t)
	modules := []string{
		"/leaflet.js",
		"/i18n.js",
		"/config.js",
		"/locale.js",
		"/theme.js",
		"/form.js",
		"/api-builder.js",
		"/renderer.js",
		"/map-connector.js",
		"/map.js",
		"/api-client.js",
		"/history.js",
		"/app.js",
	}
	for _, path := range modules {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: want 200, got %d", path, rec.Code)
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.Contains(ct, "javascript") {
				t.Errorf("GET %s Content-Type = %q, want javascript", path, ct)
			}
		})
	}
}
