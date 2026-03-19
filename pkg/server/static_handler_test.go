package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticHandler_ServesIndexHTML verifies that GET / returns the embedded
// HTML page with the expected Content-Type and known content markers.
func TestStaticHandler_ServesIndexHTML(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "PathProbe") {
		t.Error("index.html must contain the string 'PathProbe'")
	}
	if !strings.Contains(body, "app.js") {
		t.Error("index.html must reference app.js")
	}
	if !strings.Contains(body, "config.js") {
		t.Error("index.html must reference config.js")
	}
	if !strings.Contains(body, "style.css") {
		t.Error("index.html must reference style.css")
	}
}

// TestStaticHandler_ServesStyleCSS verifies that the CSS file is served with
// the correct Content-Type.
func TestStaticHandler_ServesStyleCSS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/css") {
		t.Fatalf("Content-Type = %q, want text/css", ct)
	}
}

// TestStaticHandler_ServesAppJS verifies that the JavaScript file is served
// with a JavaScript Content-Type.
func TestStaticHandler_ServesAppJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("Content-Type = %q, want javascript content type", ct)
	}
}

// TestStaticHandler_ServesI18nJS verifies that the i18n translation module is
// embedded and served with a JavaScript Content-Type.
func TestStaticHandler_ServesI18nJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("Content-Type = %q, want javascript content type", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "LOCALES") {
		t.Error("i18n.js must export the LOCALES object")
	}
	if !strings.Contains(body, "zh-TW") {
		t.Error("i18n.js must contain the zh-TW locale")
	}
}

// TestStaticHandler_NotFound verifies that a non-existent path returns 404.
func TestStaticHandler_NotFound(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist.xyz", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist.xyz: want 404, got %d", rec.Code)
	}
}

// TestStaticHandler_DoesNotInterceptAPIHealth ensures that the static catch-all
// (GET /) does not shadow the dedicated API health handler.
func TestStaticHandler_DoesNotInterceptAPIHealth(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("GET /api/health Content-Type = %q, want application/json", ct)
	}
}

// TestStaticHandler_DiagPathReturnsError verifies that GET /api/diag returns
// an error response (404 from the file server — the path doesn't exist in the
// embedded FS) and is NOT served as a 200 HTML page by the catch-all.
func TestStaticHandler_DiagPathReturnsError(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/diag", nil))

	// With GET / registered, Go 1.22+ ServeMux routes GET /api/diag to the
	// static handler which returns 404 (no matching file). We only assert it
	// is not a successful 2xx response (i.e. not served as the home page).
	if rec.Code < 400 {
		t.Fatalf("GET /api/diag: want 4xx error, got %d", rec.Code)
	}
}

// TestStaticHandler_ServesRendererJS verifies that the static file server
// serves renderer.js with HTTP 200 and that the file exports
// PathProbe.Renderer.
func TestStaticHandler_ServesRendererJS(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")
	if !strings.Contains(body, "PathProbe.Renderer") {
		t.Error("renderer.js: must export PathProbe.Renderer")
	}
	if !strings.Contains(body, "renderReport") {
		t.Error("renderer.js: must define renderReport")
	}
	if !strings.Contains(body, "rerenderLast") {
		t.Error("renderer.js: must define rerenderLast")
	}
}

// TestStaticHandler_ServesMapConnectorJS verifies that the static file server
// handles GET /map-connector.js with HTTP 200 and returns the module that
// exposes PathProbe.MapConnector.
func TestStaticHandler_ServesMapConnectorJS(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")
	if !strings.Contains(body, "PathProbe.MapConnector") {
		t.Error("/map-connector.js: response must export PathProbe.MapConnector")
	}
	if !strings.Contains(body, "buildConnectorLayer") {
		t.Error("/map-connector.js: response must expose buildConnectorLayer in PathProbe.MapConnector")
	}
}

// TestStaticHandler_ServesConfigJS verifies that GET /config.js returns HTTP 200
// with a JavaScript Content-Type.
func TestStaticHandler_ServesConfigJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("GET /config.js Content-Type = %q, want javascript content type", ct)
	}
}

// TestStaticHandler_ServesLocaleJS verifies that the embedded static file
// server correctly serves locale.js with an HTTP 200 response.  This confirms
// that the Go embed directive picks up the new file and that the route is
// reachable by the browser.
func TestStaticHandler_ServesLocaleJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "PathProbe.Locale") {
		t.Error("locale.js: must export PathProbe.Locale namespace")
	}
}

// TestStaticHandler_ServesThemeJS verifies that the embedded web filesystem
// serves theme.js with a 200 OK and that the file registers the
// PathProbe.Theme namespace, which all theme-aware tests depend on.
func TestStaticHandler_ServesThemeJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "PathProbe.Theme") {
		t.Error("theme.js: must register PathProbe.Theme namespace")
	}
}

// TestStaticHandler_ServesFormJS verifies that GET /form.js returns HTTP 200
// and that the response body registers the PathProbe.Form namespace so that
// dependent scripts can delegate form operations through it.
func TestStaticHandler_ServesFormJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/form.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /form.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "PathProbe.Form") {
		t.Error("form.js: must register PathProbe.Form namespace")
	}
}

// TestStaticHandler_ServesApiBuilderJS verifies that GET /api-builder.js returns
// HTTP 200 and that the response body registers the PathProbe.ApiBuilder namespace.
func TestStaticHandler_ServesApiBuilderJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api-builder.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api-builder.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "PathProbe.ApiBuilder") {
		t.Error("api-builder.js: must register PathProbe.ApiBuilder namespace")
	}
}

// TestStaticHandler_ServesMapJS verifies that the static file server handles
// GET /map.js with HTTP 200 and returns the module that exposes PathProbe.Map.
func TestStaticHandler_ServesMapJS(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")
	if !strings.Contains(body, "PathProbe.Map") {
		t.Error("/map.js: response must export PathProbe.Map")
	}
	for _, fn := range []string{"renderMap", "syncMapTileVariantToTheme", "setMapTileVariant", "refreshMapMarkers"} {
		if !strings.Contains(body, fn) {
			t.Errorf("/map.js: PathProbe.Map must expose %q", fn)
		}
	}
}
