package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-pathprobe/pkg/diag"
)

// TestStaticHandler_ServesIndexHTML verifies that GET / returns the embedded
// HTML page with the expected Content-Type and known content markers.
func TestStaticHandler_ServesIndexHTML(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
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
	if !strings.Contains(body, "style.css") {
		t.Error("index.html must reference style.css")
	}
}

// TestStaticHandler_ServesStyleCSS verifies that the CSS file is served with
// the correct Content-Type.
func TestStaticHandler_ServesStyleCSS(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
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
	h := newHandler(t, diag.NewDispatcher(nil))
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
	h := newHandler(t, diag.NewDispatcher(nil))
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

// TestStaticI18n_RunButtonLabels verifies that the embedded i18n.js separates
// the card-title key (run-diagnostic) from the button key (btn-run), and that
// the button uses an icon-only value (U+25B6) with no text label.
func TestStaticI18n_RunButtonLabels(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Card-title keys must carry the full section names (not the icon).
	for _, want := range []string{"'run-diagnostic'", "Run Diagnostic"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js en: missing %q for run-diagnostic key", want)
		}
	}
	if !strings.Contains(body, "\u57f7\u884c\u8a3a\u65b7") { // 執行診斷
		t.Error("i18n.js zh-TW: run-diagnostic must contain '\u57f7\u884c\u8a3a\u65b7'")
	}

	// btn-run must be the icon-only triangle (U+25B6); btn-running empty (spinner only).
	if !strings.Contains(body, "'btn-run'") {
		t.Error("i18n.js: btn-run key must be present")
	}
	if !strings.Contains(body, "\u25b6") { // ▶
		t.Error("i18n.js: btn-run must contain the play triangle '\u25b6'")
	}
	if !strings.Contains(body, "'btn-running'") {
		t.Error("i18n.js: btn-running key must be present")
	}
}

// TestStaticCSS_ButtonFixedDimensions verifies that the embedded style.css declares
// all fixed-dimension properties required to prevent layout shift on buttons.
func TestStaticCSS_ButtonFixedDimensions(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// .lang-btn: explicit fixed width prevents re-flow when EN/TW ↔ 英文/繁中.
	if !strings.Contains(body, "width: 2.8rem") {
		t.Error("style.css: .lang-btn must declare 'width: 2.8rem' to prevent locale-switch layout shift")
	}

	// #run-btn: square icon-only button — both width and height must be fixed.
	if !strings.Contains(body, "width: 2.75rem") {
		t.Error("style.css: #run-btn must declare 'width: 2.75rem' for icon-only square shape")
	}
	if !strings.Contains(body, "height: 2.75rem") {
		t.Error("style.css: #run-btn must declare 'height: 2.75rem' to prevent vertical layout shift")
	}
}

// TestStaticHandler_NotFound verifies that a non-existent path returns 404.
func TestStaticHandler_NotFound(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist.xyz", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist.xyz: want 404, got %d", rec.Code)
	}
}

// TestStaticHandler_DoesNotInterceptAPIHealth ensures that the static catch-all
// (GET /) does not shadow the dedicated API health handler.
func TestStaticHandler_DoesNotInterceptAPIHealth(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
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
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/diag", nil))

	// With GET / registered, Go 1.22+ ServeMux routes GET /api/diag to the
	// static handler which returns 404 (no matching file). We only assert it
	// is not a successful 2xx response (i.e. not served as the home page).
	if rec.Code < 400 {
		t.Fatalf("GET /api/diag: want 4xx error, got %d", rec.Code)
	}
}
