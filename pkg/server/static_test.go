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

// TestStaticHTML_ThemeSelector verifies that the embedded index.html contains
// the theme-bar container with five circular dot-buttons, ordered left-to-right
// as: forest-green, light-green, default, deep-blue, dark.
func TestStaticHTML_ThemeSelector(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The wrapper container must be present.
	if !strings.Contains(body, `class="theme-bar"`) {
		t.Error("index.html: missing .theme-bar container")
	}

	// All five theme dot-buttons must be present and in the prescribed order
	// (forest-green → light-green → default → deep-blue → dark, left-to-right).
	ordered := []string{"forest-green", "light-green", "default", "deep-blue", "dark"}
	prevIdx := -1
	for _, theme := range ordered {
		want := `data-theme="` + theme + `"`
		idx := strings.Index(body, want)
		if idx == -1 {
			t.Errorf("index.html: theme button for %q is missing", theme)
			continue
		}
		if idx <= prevIdx {
			t.Errorf("index.html: theme button %q is out of order", theme)
		}
		prevIdx = idx
	}

	// Buttons must NOT contain visible text (icon-only design).
	// Each button element should be self-contained (no child text node between
	// opening and closing tags beyond whitespace).
	if strings.Contains(body, `theme-select`) {
		t.Error("index.html: old <select id='theme-select'> must be removed in favour of dot-buttons")
	}
}

// TestStaticI18n_ThemeLabels verifies that both locales in the embedded i18n.js
// carry translations for all five theme IDs, ensuring the switcher options are
// localised correctly regardless of the active language.
func TestStaticI18n_ThemeLabels(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// All five theme keys must be present.
	for _, key := range []string{
		"'theme-default'", "'theme-deep-blue'", "'theme-light-green'",
		"'theme-forest-green'", "'theme-dark'",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing key %s", key)
		}
	}

	// en locale must carry English labels.
	for _, label := range []string{"Default", "Deep Blue", "Light Green", "Forest Green", "Dark"} {
		if !strings.Contains(body, label) {
			t.Errorf("i18n.js en: missing label %q", label)
		}
	}

	// zh-TW locale must carry Chinese labels.
	for _, label := range []string{"\u9810\u8a2d", "\u6df1\u85cd", "\u6de1\u7da0", "\u58a8\u7da0", "\u6697\u9ed1"} {
		if !strings.Contains(body, label) {
			t.Errorf("i18n.js zh-TW: missing label %q", label)
		}
	}
}

// TestStaticCSS_ThemeBarButtons verifies that the embedded style.css defines
// the circular dot-button styles for the .theme-bar switcher. It confirms:
//  1. .theme-btn uses border-radius: 50% to produce a circle.
//  2. Each of the five themes has a per-theme background rule targeting the
//     button element via .theme-btn[data-theme="..."], keeping button colours
//     independent of the active page theme.
//  3. Flat-design constraints: no linear-gradient (flashy half-split removed)
//     and no transform: scale in hover/active (no distracting zoom effect).
func TestStaticCSS_ThemeBarButtons(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Base circle shape.
	if !strings.Contains(body, "border-radius: 50%") {
		t.Error("style.css: .theme-btn must declare 'border-radius: 50%' for circular shape")
	}

	// Per-theme button colour rules (independent of page-level data-theme).
	for _, theme := range []string{"forest-green", "light-green", "default", "deep-blue", "dark"} {
		selector := `.theme-btn[data-theme="` + theme + `"]`
		if !strings.Contains(body, selector) {
			t.Errorf("style.css: missing per-button colour rule for theme %q (expected selector %s)", theme, selector)
		}
	}

	// Flat-design: buttons must use solid colour only (no hard-split gradient).
	if strings.Contains(body, "linear-gradient") {
		t.Error("style.css: .theme-btn must not use linear-gradient — flat solid colour only")
	}

	// Flat-design: no scale transform on hover/active (avoids flashy zoom).
	if strings.Contains(body, "transform: scale") {
		t.Error("style.css: .theme-btn must not use transform: scale — flat transition only")
	}

	// Old <select> style must be gone.
	if strings.Contains(body, "#theme-select") {
		t.Error("style.css: old #theme-select rule must be removed")
	}
}

// TestStaticHTML_ThemeBarInHeaderInner verifies that the theme-bar sits inside
// the same .header-inner flex row as the brand and the language switcher,
// enabling the browser to vertically centre all three elements in one pass via
// align-items: center without a separate row above the title.
func TestStaticHTML_ThemeBarInHeaderInner(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// .header-brand wrapper must exist (wraps h1 + version-badge as flex: 1 left column).
	if !strings.Contains(body, `class="header-brand"`) {
		t.Error("index.html: missing .header-brand wrapper — required for 3-column header layout")
	}

	// 3-column order inside header-inner: header-brand THEN theme-bar THEN lang-switcher.
	brandIdx := strings.Index(body, `class="header-brand"`)
	themeIdx := strings.Index(body, `class="theme-bar"`)
	langIdx := strings.Index(body, `class="lang-switcher"`)
	if brandIdx == -1 || themeIdx == -1 || langIdx == -1 {
		t.Fatal("index.html: header-brand, theme-bar, or lang-switcher is missing")
	}
	if !(brandIdx < themeIdx && themeIdx < langIdx) {
		t.Errorf("index.html: 3-column order must be header-brand < theme-bar < lang-switcher, got positions %d %d %d",
			brandIdx, themeIdx, langIdx)
	}

	// theme-bar must appear AFTER the header-inner opening tag, confirming it is
	// inline (not a separate block before header-inner).
	headerInnerIdx := strings.Index(body, `class="header-inner"`)
	if themeIdx < headerInnerIdx {
		t.Error("index.html: theme-bar must be inside .header-inner, not above it")
	}
}

// TestStaticCSS_ThemeBarFlat verifies that .header-brand exists in CSS to
// support the left-column flex layout of the 3-column header.
func TestStaticCSS_ThemeBarFlat(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// .header-brand must be defined to anchor the left column with flex: 1.
	if !strings.Contains(body, ".header-brand") {
		t.Error("style.css: .header-brand rule must be defined for 3-column header layout")
	}
	// .lang-switcher must use flex: 1 (not margin-left: auto) to mirror the brand column.
	if strings.Contains(body, "margin-left: auto") {
		t.Error("style.css: lang-switcher must not use margin-left: auto in the 3-column layout")
	}
}

// TestStaticCSS_ThemeVariables verifies that the embedded style.css contains
// CSS variable override blocks for all four non-default themes. Each block is
// identified by the [data-theme="..."] attribute selector; the presence of the
// selector proves the theme can be activated purely via a data attribute, with
// no additional JavaScript style manipulation required.
func TestStaticCSS_ThemeVariables(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Each non-default theme must have its own [data-theme] block.
	for _, theme := range []string{"deep-blue", "light-green", "forest-green", "dark"} {
		selector := `[data-theme="` + theme + `"]`
		if !strings.Contains(body, selector) {
			t.Errorf("style.css: missing theme block for %q (expected selector %s)", theme, selector)
		}
	}

	// The key component-level token variables must be tokenised in :root so
	// theme overrides propagate through to all component rules.
	for _, token := range []string{
		"--input-bg", "--error-bg", "--error-border",
		"--badge-ok-bg", "--badge-fail-bg", "--focus-ring", "--surface-alt",
	} {
		if !strings.Contains(body, token) {
			t.Errorf("style.css: :root must declare the %q CSS variable for theme overrides to work", token)
		}
	}
}

// TestStaticHTML_DefaultThemeAttribute verifies that the embedded index.html
// declares a data-default-theme attribute on the <html> root element.
// This attribute acts as the server-side declaration of the intended startup
// theme: the JS initTheme() reads it on every page load and applies it as the
// fallback whenever no user preference is stored in localStorage.
// Asserting the attribute value is "default" (the third dot-button) ensures a
// service restart always presents a known, predictable starting state.
func TestStaticHTML_DefaultThemeAttribute(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The <html> tag must carry data-default-theme="default".
	const want = `data-default-theme="default"`
	if !strings.Contains(body, want) {
		t.Errorf("index.html: <html> tag must declare %s so initTheme() can read the server-declared default", want)
	}
}

// TestStaticJS_DefaultThemeConstant verifies that the embedded app.js declares
// the DEFAULT_THEME constant and that initTheme() reads the HTML
// data-default-theme attribute as its authoritative fallback source, rather
// than relying on a hardcoded string literal.
func TestStaticJS_DefaultThemeConstant(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// DEFAULT_THEME constant must be declared.
	if !strings.Contains(body, "const DEFAULT_THEME") {
		t.Error("app.js: DEFAULT_THEME constant must be declared")
	}

	// initTheme must read the HTML attribute for the server-declared default.
	if !strings.Contains(body, "dataset.defaultTheme") {
		t.Error("app.js: initTheme() must read document.documentElement.dataset.defaultTheme")
	}

	// The fallback chain must validate against THEMES before applying.
	if !strings.Contains(body, "THEMES.includes(htmlDefault)") {
		t.Error("app.js: initTheme() must validate htmlDefault against THEMES before use")
	}
}
