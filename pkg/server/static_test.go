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

	// Card-title keys must carry the section names (not the icon).
	for _, want := range []string{"'run-diagnostic'", "Diagnostic"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js en: missing %q for run-diagnostic key", want)
		}
	}
	if !strings.Contains(body, "\u8a3a\u65b7") { // 診斷
		t.Error("i18n.js zh-TW: run-diagnostic must contain '\u8a3a\u65b7'")
	}

	// History-title key — section 2 label.
	for _, want := range []string{"'history-title'", "History"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js en: missing %q for history-title key", want)
		}
	}
	if !strings.Contains(body, "\u8a18\u9304") { // 記錄
		t.Error("i18n.js zh-TW: history-title must contain '\u8a18\u9304'")
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
	// Scope the check to only the theme-bar section to avoid false positives
	// from other components that legitimately use scale transforms (e.g.
	// the custom-select popup uses scaleY for its entrance animation).
	// The section comment may carry an inline description after the mark;
	// search only for the fixed prefix that will always be present.
	themeSectionMark := "/* \u2500\u2500 Theme bar"
	themeSecStart := strings.Index(body, themeSectionMark)
	if themeSecStart == -1 {
		t.Fatalf("style.css: '/* ── Theme bar …' section comment not found; snippet around 'theme-btn':\n%s",
			func() string {
				idx := strings.Index(body, ".theme-btn")
				if idx == -1 {
					return "(.theme-btn not found)"
				}
				start := idx - 120
				if start < 0 {
					start = 0
				}
				end := idx + 120
				if end > len(body) {
					end = len(body)
				}
				return body[start:end]
			}())
	}
	themeBarSec := body[themeSecStart:]
	if nextSec := strings.Index(themeBarSec[len(themeSectionMark):], "/* \u2500\u2500"); nextSec != -1 {
		themeBarSec = themeBarSec[:len(themeSectionMark)+nextSec]
	}
	if strings.Contains(themeBarSec, "transform: scale") {
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

// TestStaticHTML_BrandMarkup verifies that the embedded index.html renders
// the "PathProbe" logotype as two separate spans so that CSS can apply
// independent weight/opacity to each half.
func TestStaticHTML_BrandMarkup(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `class="brand-path"`) {
		t.Error(`index.html: expected <span class="brand-path"> inside h1`)
	}
	if !strings.Contains(body, `class="brand-probe"`) {
		t.Error(`index.html: expected <span class="brand-probe"> inside h1`)
	}
	// The plain text logotype must no longer appear as a bare text node.
	if strings.Contains(body, `<h1>PathProbe</h1>`) {
		t.Error("index.html: h1 must use brand-path/brand-probe spans, not bare text")
	}
}

// TestStaticCSS_BrandTypography verifies that the embedded style.css contains
// the --brand-font token, individual brand-span rules, and the commented-out
// @font-face swap-point template so future custom fonts require only updating
// that one CSS variable.
func TestStaticCSS_BrandTypography(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	checks := []struct {
		needle string
		msg    string
	}{
		{"--brand-font", "style.css: --brand-font token must be declared in :root"},
		{".brand-path", "style.css: .brand-path rule must exist"},
		{".brand-probe", "style.css: .brand-probe rule must exist"},
		{"@font-face", "style.css: @font-face swap-point template must be present (as a comment)"},
		{"font-display: swap", "style.css: @font-face template must include font-display: swap"},
		{"brand.woff2", "style.css: @font-face template must reference brand.woff2"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.needle) {
			t.Error(c.msg)
		}
	}
}

// TestStaticCSS_HeaderPaddingToken verifies that the embedded style.css uses a
// --header-py CSS custom property for vertical header padding.  This makes
// header height adjustments a single-token change with no selector hunting.
func TestStaticCSS_HeaderPaddingToken(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "--header-py") {
		t.Error("style.css: --header-py token must be declared in :root")
	}
	if !strings.Contains(body, "var(--header-py)") {
		t.Error("style.css: .site-header must consume var(--header-py) for vertical padding")
	}
}

// TestStaticCSS_BrandLogoSizeTokens verifies that style.css declares a unified
// --brand-logo-size token in :root and that both .brand-path and .brand-probe
// consume it via var(), so both glyphs always share the same size.
func TestStaticCSS_BrandLogoSizeTokens(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "--brand-logo-size") {
		t.Error("style.css: --brand-logo-size token must be declared in :root")
	}
	// Both glyphs must reference the unified token — no separate size tokens.
	if strings.Contains(body, "--brand-path-size") {
		t.Error("style.css: --brand-path-size must not exist; use --brand-logo-size instead")
	}
	if strings.Contains(body, "--brand-probe-size") {
		t.Error("style.css: --brand-probe-size must not exist; use --brand-logo-size instead")
	}
	// Count occurrences of var(--brand-logo-size): must appear for .brand-path AND .brand-probe.
	count := strings.Count(body, "var(--brand-logo-size)")
	if count < 2 {
		t.Errorf("style.css: var(--brand-logo-size) must be used at least twice (brand-path + brand-probe), got %d", count)
	}
}

// TestStaticHTML_BrandNoPicker verifies that the embedded index.html no longer
// contains the picker markup now that the logo style is fixed.
func TestStaticHTML_BrandNoPicker(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, absent := range []string{
		"brand-type-wrapper",
		"brand-style-btn",
		"brand-style-picker",
	} {
		if strings.Contains(body, absent) {
			t.Errorf("index.html: picker markup %q must not be present", absent)
		}
	}
}

// ── Web mode radio-button tests ───────────────────────────────────────────

// TestStaticHTML_WebModeRadioButtons verifies the four radio buttons exist.
func TestStaticHTML_WebModeRadioButtons(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, mode := range []string{"public-ip", "dns", "http", "port"} {
		if !strings.Contains(body, `value="`+mode+`"`) {
			t.Errorf("index.html: missing radio button with value=%q", mode)
		}
	}
	// One of the radio buttons must be pre-checked.
	if !strings.Contains(body, `name="web-mode"`) {
		t.Error("index.html: radio buttons must carry name=\"web-mode\"")
	}
}

// TestStaticHTML_WebModeDNSSubpanel verifies that the DNS sub-panel exists with
// the placeholder attribute (no hard-coded value).
func TestStaticHTML_WebModeDNSSubpanel(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `id="web-fields-dns"`) {
		t.Error("index.html: DNS sub-panel #web-fields-dns must exist")
	}
	if !strings.Contains(body, `data-i18n-placeholder="ph-dns-domains"`) {
		t.Error("index.html: dns-domains input must use data-i18n-placeholder")
	}
	// Must NOT have a hard-coded value="example.com"
	if strings.Contains(body, `value="example.com"`) {
		t.Error("index.html: dns-domains must not have hard-coded value=\"example.com\"")
	}
}

// TestStaticHTML_WebModeRecordTypeLabels verifies i18n labels for A/AAAA/MX.
func TestStaticHTML_WebModeRecordTypeLabels(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, key := range []string{"dns-type-A", "dns-type-AAAA", "dns-type-MX"} {
		if !strings.Contains(body, `data-i18n="`+key+`"`) {
			t.Errorf("index.html: missing data-i18n=%q for record type label", key)
		}
	}
}

// TestStaticI18n_WebModeKeys verifies required web-mode and dns-type keys exist
// in both locales.
func TestStaticI18n_WebModeKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	requiredKeys := []string{
		"label-web-mode",
		"web-mode-public-ip",
		"web-mode-dns",
		"web-mode-http",
		"web-mode-port",
		"dns-type-A",
		"dns-type-AAAA",
		"dns-type-MX",
		"ph-dns-domains",
	}
	for _, k := range requiredKeys {
		if !strings.Contains(body, `'`+k+`'`) {
			t.Errorf("i18n.js: missing key '%s'", k)
		}
	}
}

// TestStaticCSS_ModeSelector verifies the .mode-selector and .mode-option style rules exist.
func TestStaticCSS_ModeSelector(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, rule := range []string{".mode-selector", ".mode-option"} {
		if !strings.Contains(body, rule) {
			t.Errorf("style.css: %s rule must be defined", rule)
		}
	}
}

// TestStaticHTML_SMTPModeSelector verifies SMTP mode-selector and sub-panels exist.
func TestStaticHTML_SMTPModeSelector(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, mode := range []string{"handshake", "auth", "send"} {
		if !strings.Contains(body, `name="smtp-mode" value="`+mode+`"`) {
			t.Errorf("index.html: missing SMTP radio with value=%q", mode)
		}
	}
	for _, panel := range []string{"smtp-fields-auth", "smtp-fields-send"} {
		if !strings.Contains(body, `id="`+panel+`"`) {
			t.Errorf("index.html: missing sub-panel #%s", panel)
		}
	}
}

// TestStaticHTML_FTPModeSelector verifies FTP mode-selector exists and ftp-list checkbox is absent.
func TestStaticHTML_FTPModeSelector(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, mode := range []string{"login", "list"} {
		if !strings.Contains(body, `name="ftp-mode" value="`+mode+`"`) {
			t.Errorf("index.html: missing FTP radio with value=%q", mode)
		}
	}
	if strings.Contains(body, `id="ftp-list"`) {
		t.Error("index.html: ftp-list checkbox must be removed (replaced by mode selector)")
	}
}

// TestStaticHTML_SFTPModeSelector verifies SFTP mode-selector exists and sftp-ls checkbox is absent.
func TestStaticHTML_SFTPModeSelector(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, mode := range []string{"auth", "ls"} {
		if !strings.Contains(body, `name="sftp-mode" value="`+mode+`"`) {
			t.Errorf("index.html: missing SFTP radio with value=%q", mode)
		}
	}
	if strings.Contains(body, `id="sftp-ls"`) {
		t.Error("index.html: sftp-ls checkbox must be removed (replaced by mode selector)")
	}
}

// TestStaticI18n_SMTPFTPSFTPModeKeys verifies SMTP/FTP/SFTP mode i18n keys in both locales.
func TestStaticI18n_SMTPFTPSFTPModeKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	required := []string{
		"label-smtp-mode", "smtp-mode-handshake", "smtp-mode-auth", "smtp-mode-send",
		"label-ftp-mode", "ftp-mode-login", "ftp-mode-list",
		"label-sftp-mode", "sftp-mode-auth", "sftp-mode-ls",
	}
	for _, k := range required {
		if !strings.Contains(body, `'`+k+`'`) {
			t.Errorf("i18n.js: missing key '%s'", k)
		}
	}
}

// TestStaticI18n_ModeLabelsDetectionMode verifies that all protocol mode labels
// use 'Detection Mode' in en and '偵測模式' in zh-TW — consistent with the
// Web/DNS fieldset wording.  'Test Mode' must not appear for any mode label.
func TestStaticI18n_ModeLabelsDetectionMode(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Every mode-label key must map to 'Detection Mode' (en) somewhere in the file.
	for _, k := range []string{"label-smtp-mode", "label-ftp-mode", "label-sftp-mode"} {
		if !strings.Contains(body, `'`+k+`':`) {
			t.Errorf("i18n.js: key '%s' missing", k)
		}
	}
	// 'Detection Mode' value must appear at least three times (smtp/ftp/sftp).
	count := strings.Count(body, "'Detection Mode'")
	if count < 3 {
		t.Errorf("i18n.js: expected at least 3 occurrences of 'Detection Mode', got %d", count)
	}
	// zh-TW '偵測模式' must appear at least four times (web + smtp + ftp + sftp).
	zhCount := strings.Count(body, "'偵測模式'")
	if zhCount < 4 {
		t.Errorf("i18n.js: expected at least 4 occurrences of '偵測模式' (zh-TW), got %d", zhCount)
	}
	// Old wording 'Test Mode' must not appear anywhere.
	if strings.Contains(body, "'Test Mode'") {
		t.Error("i18n.js: 'Test Mode' must be replaced by 'Detection Mode'")
	}
}

// TestStaticI18n_ZhTWModeTranslations verifies that the zh-TW locale has
// proper Chinese translations for all SMTP/FTP/SFTP mode option values.
func TestStaticI18n_ZhTWModeTranslations(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Check zh-TW mode option translations are present.
	zhTranslations := []struct {
		key  string
		want string
	}{
		{"label-smtp-mode", "偵測模式"},
		{"smtp-mode-handshake", "無驗證"}, // partial match is sufficient
		{"smtp-mode-auth", "身分驗證"},
		{"smtp-mode-send", "傳送"},
		{"label-ftp-mode", "偵測模式"},
		{"ftp-mode-login", "連線並登入"},
		{"ftp-mode-list", "目錄列表"},
		{"label-sftp-mode", "偵測模式"},
		{"sftp-mode-auth", "身分驗證"},
		{"sftp-mode-ls", "列出目錄"},
	}
	for _, tc := range zhTranslations {
		if !strings.Contains(body, tc.want) {
			t.Errorf("i18n.js zh-TW: key '%s' — expected Chinese translation containing %q", tc.key, tc.want)
		}
	}
}

// TestStaticHTML_ModeLabelFallbackText verifies that the fallback text for all
// mode-selector labels in index.html is 'Detection Mode' (not 'Test Mode').
func TestStaticHTML_ModeLabelFallbackText(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if strings.Contains(body, ">Test Mode<") {
		t.Error("index.html: fallback text 'Test Mode' must be replaced by 'Detection Mode'")
	}
	// Each of the three protocol fieldsets must carry the correct fallback text.
	for _, key := range []string{"label-smtp-mode", "label-ftp-mode", "label-sftp-mode"} {
		want := `data-i18n="` + key + `">Detection Mode`
		if !strings.Contains(body, want) {
			t.Errorf("index.html: label with data-i18n=%q must have fallback text 'Detection Mode'", key)
		}
	}
}

// TestStaticJS_BrandSystemRemoved verifies that the brand style management
// system has been removed from app.js now that the logo style is fixed.
func TestStaticJS_BrandSystemRemoved(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

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

// TestStaticCSS_HeaderShadow verifies that the embedded style.css declares a
// --header-shadow CSS token in :root and that .site-header consumes it via
// var(--header-shadow), keeping the shadow value a single-token change.
func TestStaticCSS_HeaderShadow(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "--header-shadow") {
		t.Error("style.css: --header-shadow token must be declared in :root")
	}
	if !strings.Contains(body, "var(--header-shadow)") {
		t.Error("style.css: .site-header must consume var(--header-shadow)")
	}
}

// TestStaticCSS_StickyHeader verifies that the embedded style.css makes the
// site header stick to the top of the viewport while the page is scrolled.
// position: sticky + top: 0 achieves this without removing the header from
// normal document flow (unlike position: fixed), so .main requires no extra
// margin-top compensation.  z-index ensures the header layers above all cards.
func TestStaticCSS_StickyHeader(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "position: sticky") {
		t.Error("style.css: .site-header must declare 'position: sticky' to stay visible during scroll")
	}
	if !strings.Contains(body, "top: 0") {
		t.Error("style.css: .site-header must declare 'top: 0' to anchor at the viewport top")
	}
	if !strings.Contains(body, "z-index: 100") {
		t.Error("style.css: .site-header must declare 'z-index: 100' to layer above page content")
	}
}

// TestStaticCSS_SelectCustomChevron verifies that the embedded style.css
// removes the native OS dropdown arrow and replaces it with a custom chevron
// that follows the active theme's --primary colour via CSS mask-image.
// Both .select-wrap and .cs-wrap must carry this chevron so legacy native
// selects and the new custom-select widget look identical.
func TestStaticCSS_SelectCustomChevron(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Native arrow must be suppressed.
	if !strings.Contains(body, "appearance: none") {
		t.Error("style.css: select must declare 'appearance: none' to remove the native OS arrow")
	}
	if !strings.Contains(body, "-webkit-appearance: none") {
		t.Error("style.css: select must declare '-webkit-appearance: none' for Safari/Chrome compat")
	}

	// Both wrapper classes must be defined.
	for _, cls := range []string{".select-wrap", ".cs-wrap"} {
		if !strings.Contains(body, cls) {
			t.Errorf("style.css: %s rule must exist as a positioning context for the chevron", cls)
		}
	}

	// Custom chevron uses mask-image so background-color: var(--primary) provides
	// the colour — automatically correct for every theme.
	if !strings.Contains(body, "mask-image") {
		t.Error("style.css: chevron must use mask-image for the theme-aware colouring")
	}
	if !strings.Contains(body, "background-color: var(--primary)") {
		t.Error("style.css: chevron must use background-color: var(--primary) so colour tracks the active theme")
	}
	// Rotation signal: cs-wrap.open must rotate the chevron 180°.
	if !strings.Contains(body, "rotate(180deg)") {
		t.Error("style.css: .cs-wrap.open::after must rotate the chevron 180deg to indicate open state")
	}
}

// TestStaticHTML_CustomSelectMarkup verifies that the target <select> in
// index.html has been replaced with the custom .cs-wrap widget and that the
// hidden native <select id="target"> is still present so val('target')
// continues to work without any other JS changes.
func TestStaticHTML_CustomSelectMarkup(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Custom-select wrapper must be present.
	if !strings.Contains(body, `class="cs-wrap"`) {
		t.Error(`index.html: <div class="cs-wrap"> must be present for the custom dropdown`)
	}
	if !strings.Contains(body, `class="cs-trigger"`) {
		t.Error(`index.html: .cs-trigger button must be present inside .cs-wrap`)
	}
	if !strings.Contains(body, `class="cs-list"`) {
		t.Error(`index.html: .cs-list popup must be present inside .cs-wrap`)
	}
	if !strings.Contains(body, `class="cs-label"`) {
		t.Error(`index.html: .cs-label span must be present inside .cs-trigger`)
	}

	// All six target values must appear as cs-item options.
	for _, v := range []string{"web", "smtp", "imap", "pop", "ftp", "sftp"} {
		want := `data-value="` + v + `"`
		if !strings.Contains(body, want) {
			t.Errorf("index.html: cs-item with %s not found", want)
		}
	}

	// The hidden native select must still be present for val('target') compat.
	if !strings.Contains(body, `id="target"`) {
		t.Fatal("index.html: hidden <select id=\"target\"> must be present for val() compatibility")
	}

	// cs-wrap must precede the hidden select in source order.
	csIdx := strings.Index(body, `class="cs-wrap"`)
	selIdx := strings.Index(body, `id="target"`)
	if csIdx == -1 || selIdx == -1 {
		t.Fatal("index.html: .cs-wrap or #target is missing")
	}
	if csIdx > selIdx {
		t.Error("index.html: .cs-wrap must appear before the hidden #target in source order")
	}

	// Accessibility: trigger must have aria-haspopup and aria-expanded.
	if !strings.Contains(body, `aria-haspopup="listbox"`) {
		t.Error(`index.html: .cs-trigger must carry aria-haspopup="listbox" for screen-reader disclosure`)
	}
	if !strings.Contains(body, `aria-expanded="false"`) {
		t.Error(`index.html: .cs-trigger must start with aria-expanded="false"`)
	}
	// cs-list must have role=listbox.
	if !strings.Contains(body, `role="listbox"`) {
		t.Error(`index.html: .cs-list must carry role="listbox"`)
	}
}

// TestStaticCSS_CustomSelectPopup verifies that style.css defines the
// cs-* component rules with theme-aware tokens for the popup's visual style.
// Specifically: rounded corners (--select-popup-r), layered shadow
// (--select-popup-shadow), surface background, and a smooth opacity+scale
// entrance transition that is impossible with the OS-native dropdown.
func TestStaticCSS_CustomSelectPopup(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Token declarations in :root.
	for _, token := range []string{"--select-popup-shadow", "--select-popup-r"} {
		if !strings.Contains(body, token) {
			t.Errorf("style.css: %s token must be declared in :root", token)
		}
	}

	// Component rules.
	for _, rule := range []string{".cs-wrap", ".cs-trigger", ".cs-list", ".cs-item"} {
		if !strings.Contains(body, rule) {
			t.Errorf("style.css: %s rule must be defined for the custom-select component", rule)
		}
	}

	// Popup uses theme tokens for background and shadow.
	if !strings.Contains(body, "var(--select-popup-shadow)") {
		t.Error("style.css: .cs-list must consume var(--select-popup-shadow)")
	}
	if !strings.Contains(body, "var(--select-popup-r)") {
		t.Error("style.css: .cs-list must consume var(--select-popup-r) for themed corner radius")
	}

	// Popup entrance is driven by opacity + transform transitions.
	if !strings.Contains(body, "scaleY") {
		t.Error("style.css: .cs-list entrance animation must include a scaleY transform for a natural dropdown feel")
	}
	// Popup animation duration and scale are driven by CSS tokens.
	if !strings.Contains(body, "var(--cs-popup-anim-dur)") {
		t.Error("style.css: .cs-list transition must consume var(--cs-popup-anim-dur) instead of a hard-coded value")
	}
	if !strings.Contains(body, "var(--cs-popup-anim-scale)") {
		t.Error("style.css: .cs-list transform must consume var(--cs-popup-anim-scale) instead of a hard-coded value")
	}
	// .cs-wrap.open reveals the list.
	if !strings.Contains(body, ".cs-wrap.open .cs-list") {
		t.Error("style.css: .cs-wrap.open .cs-list selector must make the popup visible")
	}
	// Selected item uses primary colour.
	if !strings.Contains(body, `.cs-item[aria-selected="true"]`) {
		t.Error("style.css: cs-item[aria-selected=\"true\"] must be styled for the active selection")
	}
}

// TestStaticCSS_PanelTransition verifies that style.css declares the
// panel-appear @keyframes and the .target-fields.panel-entering rule so
// onTargetChange() can trigger the fade-in animation without extra CSS.
func TestStaticCSS_PanelTransition(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "@keyframes panel-appear") {
		t.Error("style.css: @keyframes panel-appear must be declared for the target fieldset entrance animation")
	}
	if !strings.Contains(body, ".target-fields.panel-entering") {
		t.Error("style.css: .target-fields.panel-entering must consume the panel-appear animation")
	}
	// Exit animation: departing panel must also animate out.
	if !strings.Contains(body, "@keyframes panel-leave") {
		t.Error("style.css: @keyframes panel-leave must be declared for the target fieldset exit animation")
	}
	if !strings.Contains(body, ".target-fields.panel-leaving") {
		t.Error("style.css: .target-fields.panel-leaving must consume the panel-leave animation")
	}
	// Animation must use opacity (fade) and a vertical transform (slide).
	if !strings.Contains(body, "translateY") {
		t.Error("style.css: panel animations must include translateY for the entrance/exit slide effect")
	}
	// Duration and distance must be driven by CSS tokens (not hard-coded values).
	if !strings.Contains(body, "var(--panel-anim-dur)") {
		t.Error("style.css: panel transition must consume var(--panel-anim-dur) instead of a hard-coded duration")
	}
	if !strings.Contains(body, "var(--panel-anim-dist)") {
		t.Error("style.css: panel-appear keyframe must consume var(--panel-anim-dist) instead of a hard-coded pixel offset")
	}
	// The panel-stage wrapper must clip exiting panels and animate its own
	// height smoothly so the card never jumps when switching between panels of
	// different heights.
	if !strings.Contains(body, ".panel-stage") {
		t.Error("style.css: .panel-stage rule must be declared to wrap all .target-fields fieldsets")
	}
	if !strings.Contains(body, "overflow: hidden") {
		t.Error("style.css: .panel-stage must set overflow: hidden to clip the exit animation")
	}
	if !strings.Contains(body, "transition: height var(--panel-anim-dur)") {
		t.Error("style.css: .panel-stage must animate height via transition: height var(--panel-anim-dur)")
	}
}

// TestStaticJS_CustomSelectFunctions verifies that app.js defines
// initCustomSelect(), selectItem() logic, and the _initTargetDone guard that
// prevents the entrance animation from firing on the cold page load.
func TestStaticJS_CustomSelectFunctions(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "initCustomSelect") {
		t.Error("app.js: initCustomSelect function must be defined")
	}
	if !strings.Contains(body, "_initTargetDone") {
		t.Error("app.js: _initTargetDone guard must be present to skip animation on cold page load")
	}
	if !strings.Contains(body, "panel-entering") {
		t.Error("app.js: onTargetChange must manage the panel-entering CSS class for the entrance animation")
	}
	// Custom select must sync the hidden native select so val('target') stays valid.
	if !strings.Contains(body, "select.value") {
		t.Error("app.js: initCustomSelect must sync the hidden native select .value")
	}
	// Keyboard navigation arrows must be wired.
	if !strings.Contains(body, "ArrowDown") || !strings.Contains(body, "ArrowUp") {
		t.Error("app.js: initCustomSelect must handle ArrowDown and ArrowUp keyboard navigation")
	}
	// has-selection class must be managed to give persistent primary-border indicator.
	if !strings.Contains(body, "has-selection") {
		t.Error("app.js: initCustomSelect must add 'has-selection' class to .cs-wrap for persistent selection indicator")
	}
	// close() must accept a restoreFocus parameter so outside clicks don't steal focus.
	if !strings.Contains(body, "restoreFocus") {
		t.Error("app.js: close() in initCustomSelect must accept a restoreFocus parameter")
	}
	// Document click handler must call close(false) to avoid stealing focus on outside click.
	if !strings.Contains(body, "close(false)") {
		t.Error("app.js: outside-click document listener must call close(false) to avoid focus theft")
	}
}

// ── Footer tests ─────────────────────────────────────────────────────────

// TestStaticHTML_FooterPresent verifies that the embedded index.html contains
// a <footer class="site-footer"> element with the .footer-inner wrapper.
// The footer must appear after </main> so the HTML document structure follows
// the natural reading order: header → main content → footer.
func TestStaticHTML_FooterPresent(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `class="site-footer"`) {
		t.Error(`index.html: <footer class="site-footer"> must be present`)
	}
	if !strings.Contains(body, `class="footer-inner"`) {
		t.Error(`index.html: .footer-inner wrapper must be present inside .site-footer`)
	}
	if !strings.Contains(body, `class="footer-copy"`) {
		t.Error(`index.html: .footer-copy paragraph must be present inside .footer-inner`)
	}

	// Footer must appear after the closing </main> tag.
	mainIdx := strings.Index(body, "</main>")
	footerIdx := strings.Index(body, `class="site-footer"`)
	if mainIdx == -1 || footerIdx == -1 {
		t.Fatal("index.html: </main> or .site-footer is missing")
	}
	if footerIdx < mainIdx {
		t.Error("index.html: .site-footer must appear after </main> in source order")
	}
}

// TestStaticHTML_FooterCopyright verifies that the footer element contains the
// copyright notice with the data-i18n key and the expected English fallback
// text.  The copyright text must include the © symbol, the year, and the
// author name "Charles" so the notice is legally unambiguous.
func TestStaticHTML_FooterCopyright(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `data-i18n="footer-copyright"`) {
		t.Error("index.html: footer copyright paragraph must carry data-i18n=\"footer-copyright\"")
	}
	// The fallback text must contain the essential copyright elements.
	for _, want := range []string{"\u00a9", "2026", "Charles"} {
		if !strings.Contains(body, want) {
			t.Errorf("index.html: footer fallback text must contain %q for a valid copyright notice", want)
		}
	}
}

// TestStaticCSS_FooterStyles verifies that the embedded style.css defines the
// three footer component rules (.site-footer, .footer-inner, .footer-copy) and
// the --footer-shadow design token.  This ensures the footer can be restyled by
// changing a single token just like the header.
func TestStaticCSS_FooterStyles(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// All three footer component selectors must be defined.
	for _, rule := range []string{".site-footer", ".footer-inner", ".footer-copy"} {
		if !strings.Contains(body, rule) {
			t.Errorf("style.css: %s rule must be defined", rule)
		}
	}
	// Footer must reuse the same --header-py token for vertical rhythm parity.
	if !strings.Contains(body, "var(--header-py)") {
		t.Error("style.css: .site-footer must reuse var(--header-py) for vertically consistent rhythm with the header")
	}
	// The --footer-shadow token must be declared and consumed.
	if !strings.Contains(body, "--footer-shadow") {
		t.Error("style.css: --footer-shadow token must be declared in :root")
	}
	if !strings.Contains(body, "var(--footer-shadow)") {
		t.Error("style.css: .site-footer must consume var(--footer-shadow)")
	}
	// Footer must NOT be sticky or fixed — it should flow with the document.
	// We narrow the check to only the footer CSS section by using the section
	// comment marker "/* ── Footer" as the start boundary and the next "/* ──"
	// section marker as the end boundary.  This avoids false positives from
	// the header section which legitimately declares position: sticky.
	sectionMark := "/* \u2500\u2500 Footer"
	footerSecStart := strings.Index(body, sectionMark)
	if footerSecStart == -1 {
		t.Fatal("style.css: '/* ── Footer' section comment not found")
	}
	footerSec := body[footerSecStart+len(sectionMark):]
	if nextSec := strings.Index(footerSec, "/* \u2500\u2500"); nextSec != -1 {
		footerSec = footerSec[:nextSec]
	}
	if strings.Contains(footerSec, "position: sticky") || strings.Contains(footerSec, "position: fixed") {
		t.Error("style.css: .site-footer must NOT be sticky or fixed — it must flow with the document")
	}
}

// TestStaticCSS_BodyFlushBottom verifies that the embedded style.css configures
// the body as a flex-column container with min-height: 100vh, and that .main
// carries flex: 1 and width: 100%.  Together these rules guarantee:
//   - the footer is always pressed to the viewport bottom on short pages, and
//   - .main fills the full available width (up to max-width: 960px) instead of
//     shrinking to its intrinsic content width (flex cross-axis shrink-to-fit).
func TestStaticCSS_BodyFlushBottom(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "min-height: 100vh") {
		t.Error("style.css: body must declare 'min-height: 100vh' so the footer reaches the bottom on short pages")
	}
	if !strings.Contains(body, "flex-direction: column") {
		t.Error("style.css: body must declare 'flex-direction: column' for the header-main-footer stack")
	}
	if !strings.Contains(body, "flex: 1") {
		t.Error("style.css: .main must declare 'flex: 1' to fill remaining space above the footer")
	}
	// width: 100% is required so that margin: auto on the cross axis of the body
	// flex container does not trigger shrink-to-fit, which would squeeze the
	// diagnostic and history cards narrower than their intended 960px maximum.
	if !strings.Contains(body, "width: 100%") {
		t.Error("style.css: .main must declare 'width: 100%' to prevent shrink-to-fit inside the body flex container")
	}
}

// TestStaticCSS_ChromeHeightParity verifies that the embedded style.css
// declares a --chrome-inner-h design token and applies it as min-height to
// both .header-inner and .footer-inner.  This single token guarantees the
// visible chrome bars (header + footer) have identical height regardless of
// their text content size difference, producing a visually balanced bookend.
func TestStaticCSS_ChromeHeightParity(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Token must be declared in :root so themes can override it.
	if !strings.Contains(body, "--chrome-inner-h") {
		t.Error("style.css: --chrome-inner-h token must be declared in :root for header/footer height parity")
	}
	// Both inner containers must consume the token.
	count := strings.Count(body, "var(--chrome-inner-h)")
	if count < 2 {
		t.Errorf("style.css: var(--chrome-inner-h) must appear at least twice (header-inner + footer-inner), got %d", count)
	}
}

// TestStaticI18n_FooterCopyrightKey verifies that the embedded i18n.js carries
// the footer-copyright key in both the en and zh-TW locales, and that each
// value contains the required legal elements (© symbol, year, author name).
func TestStaticI18n_FooterCopyrightKey(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "'footer-copyright'") {
		t.Error("i18n.js: 'footer-copyright' key must be present")
	}
	// Both locales must include the mandatory copyright elements.
	for _, want := range []string{"\u00a9", "2026", "Charles"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js: footer-copyright value must contain %q", want)
		}
	}
	// en locale must carry the All Rights Reserved statement.
	if !strings.Contains(body, "All Rights Reserved") {
		t.Error("i18n.js en: footer-copyright must contain 'All Rights Reserved'")
	}
	// zh-TW locale must have a Chinese-language variant using the corresponding phrase.
	if !strings.Contains(body, "保留所有權利") {
		t.Error("i18n.js zh-TW: footer-copyright must contain '保留所有權利'")
	}
}

// ── Select option theming tests ───────────────────────────────────────────

// TestStaticCSS_SelectOptionTheming verifies that style.css defines theme-aware
// option styling using only CSS custom-property tokens.  A single pair of rules
// (select option + option:checked) automatically covers every theme because
// each [data-theme] block overrides the tokens they reference — no per-theme
// CSS duplication is needed.
func TestStaticCSS_SelectOptionTheming(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Locate the option-theming section so assertions are scoped to it.
	sectionMark := "/* \u2500\u2500 Select option theming"
	secStart := strings.Index(body, sectionMark)
	if secStart == -1 {
		t.Fatal("style.css: '/* ── Select option theming' section comment not found")
	}
	sec := body[secStart+len(sectionMark):]
	if nextSec := strings.Index(sec, "/* \u2500\u2500"); nextSec != -1 {
		sec = sec[:nextSec]
	}

	// Base rule: options must display the theme's input-surface background and text colour.
	if !strings.Contains(sec, "select option") {
		t.Error("style.css: 'select option' selector must be present in the option-theming section")
	}
	if !strings.Contains(sec, "var(--input-bg)") {
		t.Error("style.css: select option background-color must reference var(--input-bg) to track the theme's input surface")
	}
	if !strings.Contains(sec, "var(--text)") {
		t.Error("style.css: select option color must reference var(--text) for legible text across all themes")
	}

	// Checked/selected state must highlight using the primary colour.
	if !strings.Contains(sec, "option:checked") {
		t.Error("style.css: 'option:checked' selector must be defined for the selected-option highlight")
	}
	if !strings.Contains(sec, "var(--primary)") {
		t.Error("style.css: option:checked background-color must reference var(--primary)")
	}
	// Foreground must use the --option-checked-fg token so themes with a light
	// primary colour can override it for adequate contrast without a new CSS block.
	if !strings.Contains(sec, "var(--option-checked-fg)") {
		t.Error("style.css: option:checked color must reference var(--option-checked-fg) for per-theme contrast control")
	}
}

// TestStaticCSS_OptionCheckedFgToken verifies that --option-checked-fg is
// declared in :root (defaulting to #fff) and that [data-theme="dark"] overrides
// it to a dark tint.  The dark theme's primary is #bb86fc (light purple), so
// white text would give only ~2.8:1 contrast; the surface override raises this
// to ~7.5:1, well above the WCAG AA threshold.
func TestStaticCSS_OptionCheckedFgToken(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Token must be declared somewhere in the stylesheet.
	if !strings.Contains(body, "--option-checked-fg") {
		t.Error("style.css: --option-checked-fg token must be declared in :root")
	}

	// Locate the dark-theme block and verify it overrides the token.
	// Search for the standalone block (prefixed with newline + selector + space)
	// to avoid accidentally matching .theme-btn[data-theme="dark"] which appears
	// earlier in the CSS for the header swatch buttons.
	darkMark := "\n[data-theme=\"dark\"] {"
	darkIdx := strings.Index(body, darkMark)
	if darkIdx == -1 {
		t.Fatalf("style.css: standalone [data-theme=\"dark\"] { block not found")
	}
	darkBlock := body[darkIdx:]
	// Trim to just this block (ends at the first bare closing brace on its own line).
	if closeIdx := strings.Index(darkBlock, "\n}"); closeIdx != -1 {
		darkBlock = darkBlock[:closeIdx+2]
	}
	if !strings.Contains(darkBlock, "--option-checked-fg") {
		t.Errorf("style.css: [data-theme=\"dark\"] must override --option-checked-fg for legible text on the light-purple primary (#bb86fc)")
	}
}

// ── Animation control tests ───────────────────────────────────────────────

// TestStaticCSS_AnimationTokens verifies that style.css declares the four
// animation design tokens in :root and implements the [data-anim="vivid"] and
// [data-anim="off"] override blocks so JS can switch animation intensity by
// toggling a single HTML attribute.
func TestStaticCSS_AnimationTokens(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// All four tokens must be declared in :root.
	for _, token := range []string{
		"--panel-anim-dur",
		"--panel-anim-dist",
		"--cs-popup-anim-dur",
		"--cs-popup-anim-scale",
	} {
		if !strings.Contains(body, token) {
			t.Errorf("style.css: animation token %s must be declared in :root", token)
		}
	}

	// vivid and off mode blocks must exist.
	if !strings.Contains(body, `[data-anim="vivid"]`) {
		t.Error(`style.css: [data-anim="vivid"] override block must be present`)
	}
	if !strings.Contains(body, `[data-anim="off"]`) {
		t.Error(`style.css: [data-anim="off"] override block must be present`)
	}
}

// TestStaticCSS_CustomSelectHasSelection verifies that style.css defines a
// persistent visual indicator for .cs-wrap.has-selection .cs-trigger so the
// widget looks "selected" even when it does not have keyboard focus — mirroring
// the always-visible highlight of a checked radio button.
func TestStaticCSS_CustomSelectHasSelection(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, ".cs-wrap.has-selection .cs-trigger") {
		t.Error("style.css: .cs-wrap.has-selection .cs-trigger rule must be defined for persistent selection indicator")
	}
	if !strings.Contains(body, "border-color: var(--primary)") {
		t.Error("style.css: .cs-wrap.has-selection .cs-trigger must set border-color: var(--primary)")
	}
	// Background tint uses color-mix for accessible, theme-aware contrast.
	if !strings.Contains(body, "color-mix") {
		t.Error("style.css: .cs-wrap.has-selection .cs-trigger should use color-mix() for a subtle primary background tint")
	}
}

// TestStaticHTML_VividAnimDefault verifies that index.html permanently sets
// data-anim="vivid" on the <html> element so the vivid animation intensity is
// active from first paint without any JS initialization.
func TestStaticHTML_VividAnimDefault(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Vivid animation must be the declared default on the root element.
	if !strings.Contains(body, `data-anim="vivid"`) {
		t.Error(`index.html: <html> must carry data-anim="vivid" to apply the vivid animation intensity by default`)
	}
	// The temporary toggle button must be absent — it was a developer tool only.
	if strings.Contains(body, `id="anim-toggle"`) {
		t.Error(`index.html: temporary anim-toggle button must be removed; vivid mode is now the permanent default`)
	}
	if strings.Contains(body, `cycleAnim()`) {
		t.Error(`index.html: cycleAnim() onclick must be removed along with the toggle button`)
	}
}

// TestStaticCSS_PanelLeaveAnimation verifies that style.css defines both
// halves of the panel cross-fade: @keyframes panel-leave and the
// .target-fields.panel-leaving rule.  The leave direction (upward slide) must
// be the mirror of the enter direction so the transition feels directional.
func TestStaticCSS_PanelLeaveAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "@keyframes panel-leave") {
		t.Error("style.css: @keyframes panel-leave must be declared for the target fieldset exit animation")
	}
	if !strings.Contains(body, ".target-fields.panel-leaving") {
		t.Error("style.css: .target-fields.panel-leaving rule must consume panel-leave so onTargetChange() can trigger it")
	}
	// Leave animation must move upward — opposite direction to the enter slide.
	if !strings.Contains(body, "calc(-1 * var(--panel-anim-dist))") {
		t.Error("style.css: panel-leave must use calc(-1 * var(--panel-anim-dist)) for the mirrored upward slide")
	}
	// Interaction must be blocked during the fade-out to prevent stray clicks.
	if !strings.Contains(body, "pointer-events: none") {
		t.Error("style.css: .target-fields.panel-leaving must declare pointer-events: none to block stray clicks during fade-out")
	}
}

// TestStaticJS_PanelLeaveAnimation verifies that app.js manages the
// panel-leaving class in onTargetChange() so the departing panel animates out
// before being hidden.  Also checks that the _pendingReveal cleanup guard
// exists for safe rapid target-switching.
func TestStaticJS_PanelLeaveAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "panel-leaving") {
		t.Error("app.js: onTargetChange must add 'panel-leaving' class to the departing panel for the exit animation")
	}
	// animationend fires after the CSS animation completes; hiding then keeps layout clean.
	if !strings.Contains(body, "animationend") {
		t.Error("app.js: onTargetChange must listen for animationend to hide the departing panel after its exit animation")
	}
	// Rapid-switch guard: _pendingReveal cleanup cancels in-flight transitions.
	if !strings.Contains(body, "_pendingReveal") {
		t.Error("app.js: _pendingReveal must be defined to cancel in-flight transitions on rapid target switching")
	}
	// Toggle functions must be absent — vivid mode is now the HTML-level default.
	for _, sym := range []string{"cycleAnim", "initAnim", "applyAnim", "ANIM_MODES"} {
		if strings.Contains(body, sym) {
			t.Errorf("app.js: %s must be removed; animation mode is now a static HTML attribute, not a runtime toggle", sym)
		}
	}
}

// TestStaticJS_PanelSequentialTransition verifies that app.js implements a
// strictly sequential panel transition in onTargetChange(): the incoming panel
// is kept hidden (incoming.hidden = true) while the departing panel is still
// animating, ensuring the two panels never coexist in the layout flow.  The
// test also confirms the revealIncoming helper function exists to decouple the
// "show new panel" step from the departure listener, and that _pendingReveal
// stores a cleanup callback that can be invoked by a subsequent call to cancel
// the in-flight transition and prevent a stale reveal from running.
func TestStaticJS_PanelSequentialTransition(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The incoming panel must be explicitly hidden while the outgoing panel
	// is animating so both panels never occupy layout space at the same time.
	if !strings.Contains(body, "incoming.hidden = true") {
		t.Error("app.js: incoming.hidden must be set to true during the departure phase to prevent simultaneous layout overlap")
	}
	// revealIncoming encapsulates the deferred show+animate step and is the
	// sole entry point for making the incoming panel visible.
	if !strings.Contains(body, "revealIncoming") {
		t.Error("app.js: revealIncoming helper must be defined to decouple the reveal step from the animationend listener")
	}
	// _pendingReveal stores the listener cleanup for the in-flight transition
	// so that a rapid switch can cancel the previous departure and reveal.
	if !strings.Contains(body, "_pendingReveal") {
		t.Error("app.js: _pendingReveal cleanup variable must store the cancel function for the active transition")
	}
	// removeEventListener must be called inside the cleanup to stop stale
	// animationend handlers from triggering an outdated revealIncoming.
	if !strings.Contains(body, "removeEventListener") {
		t.Error("app.js: cleanup must call removeEventListener to prevent stale animationend handlers from triggering on rapid switch")
	}
	// Height animation: measurePanelHeight must exist to off-screen-measure the
	// incoming panel before revealing it.
	if !strings.Contains(body, "measurePanelHeight") {
		t.Error("app.js: measurePanelHeight function must be defined to measure the incoming panel height off-screen")
	}
	// measurePanelHeight must include CSS margins in the returned value so the
	// stage height transition target matches the panel's true occupied layout
	// space and does not jump when height: auto is restored afterwards.
	if !strings.Contains(body, "getComputedStyle") || !strings.Contains(body, "marginBottom") {
		t.Error("app.js: measurePanelHeight must use getComputedStyle to include marginTop/marginBottom in the height total")
	}
	// measurePanelHeight must use clone.offsetHeight (not clone.scrollHeight).
	// offsetHeight includes the element's border, while scrollHeight does not;
	// the parent stage's scrollHeight accounts for the child's full offsetHeight,
	// so using scrollHeight would leave the stage 2 px short (border top+bottom),
	// causing a visible snap when height:auto is restored at animation end.
	if !strings.Contains(body, "clone.offsetHeight") {
		t.Error("app.js: measurePanelHeight must use clone.offsetHeight (includes border) not clone.scrollHeight to avoid a 2px snap at animation end")
	}
	// stage.scrollHeight captures the current panel height before locking it.
	if !strings.Contains(body, "stage.scrollHeight") {
		t.Error("app.js: stage.scrollHeight must be read to capture current height before locking for the transition")
	}
	// stage.offsetWidth is passed to measurePanelHeight to simulate the correct layout width.
	if !strings.Contains(body, "stage.offsetWidth") {
		t.Error("app.js: stage.offsetWidth must be passed to measurePanelHeight to simulate the correct layout width")
	}
	// stage.style.height must be set and then cleared after the transition.
	if !strings.Contains(body, "stage.style.height = ") {
		t.Error("app.js: stage.style.height must be set during the height transition")
	}
	if !strings.Contains(body, "stage.style.height = ''") {
		t.Error("app.js: stage.style.height must be cleared to auto after the panel transition completes")
	}
}

// TestStaticHTML_ImapPopFieldsets verifies that index.html contains hidden
// fieldsets for the imap and pop targets so onTargetChange() can always find
// them via getElementById and cleanly hide any previously active panel.
func TestStaticHTML_ImapPopFieldsets(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `id="fields-imap"`) {
		t.Error("index.html: fieldset id=fields-imap must be present for the imap target type")
	}
	if !strings.Contains(body, `id="fields-pop"`) {
		t.Error("index.html: fieldset id=fields-pop must be present for the pop target type")
	}
	// Both fieldsets must start hidden so they are invisible until selected.
	imapHiddenIdx := strings.Index(body, `id="fields-imap"`)
	popHiddenIdx := strings.Index(body, `id="fields-pop"`)
	if imapHiddenIdx == -1 || !strings.Contains(body[imapHiddenIdx:imapHiddenIdx+200], "hidden") {
		t.Error("index.html: fields-imap fieldset must carry the hidden attribute")
	}
	if popHiddenIdx == -1 || !strings.Contains(body[popHiddenIdx:popHiddenIdx+200], "hidden") {
		t.Error("index.html: fields-pop fieldset must carry the hidden attribute")
	}
	// Both fieldsets must carry data-panel-empty="true" so JS skips the reveal
	// step and never presents an empty bordered box to the user when imap/pop
	// is selected.
	if imapHiddenIdx == -1 || !strings.Contains(body[imapHiddenIdx:imapHiddenIdx+300], `data-panel-empty="true"`) {
		t.Error(`index.html: fields-imap fieldset must carry data-panel-empty="true" to suppress the blank reveal`)
	}
	if popHiddenIdx == -1 || !strings.Contains(body[popHiddenIdx:popHiddenIdx+300], `data-panel-empty="true"`) {
		t.Error(`index.html: fields-pop fieldset must carry data-panel-empty="true" to suppress the blank reveal`)
	}
	// legend keys must be referenced so i18n can label the fieldsets.
	if !strings.Contains(body, "legend-imap") {
		t.Error("index.html: fields-imap fieldset must reference legend-imap i18n key")
	}
	if !strings.Contains(body, "legend-pop") {
		t.Error("index.html: fields-pop fieldset must reference legend-pop i18n key")
	}
}

// TestStaticI18n_ImapPopLegendKeys verifies that i18n.js defines legend-imap
// and legend-pop translation keys for both the English and zh-TW locales.
func TestStaticI18n_ImapPopLegendKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "legend-imap") {
		t.Error("i18n.js: legend-imap key must be defined (needed by fields-imap fieldset)")
	}
	if !strings.Contains(body, "legend-pop") {
		t.Error("i18n.js: legend-pop key must be defined (needed by fields-pop fieldset)")
	}
	// English translations.
	if !strings.Contains(body, "IMAP Options") {
		t.Error("i18n.js: English translation for legend-imap must be 'IMAP Options'")
	}
	if !strings.Contains(body, "POP3 Options") {
		t.Error("i18n.js: English translation for legend-pop must be 'POP3 Options'")
	}
	// Traditional Chinese translations.
	if !strings.Contains(body, "IMAP \u9078\u9805") {
		t.Error("i18n.js: zh-TW translation for legend-imap must be 'IMAP \u9078\u9805'")
	}
	if !strings.Contains(body, "POP3 \u9078\u9805") {
		t.Error("i18n.js: zh-TW translation for legend-pop must be 'POP3 \u9078\u9805'")
	}
}

// TestStaticJS_EmptyPanelHandling verifies that app.js honours the
// data-panel-empty attribute: when the target resolves to a content-free
// fieldset (imap, pop) all departing panels are still hidden and the stage
// height collapses smoothly, but the blank fieldset is never made visible so
// the user is never presented with an empty bordered box.
func TestStaticJS_EmptyPanelHandling(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// JS reads dataset.panelEmpty to decide whether to reveal the incoming panel.
	if !strings.Contains(body, "dataset.panelEmpty") {
		t.Error("app.js: onTargetChange must read dataset.panelEmpty to detect content-free panels")
	}
	// isEmptyPanel is the local flag derived from the attribute.
	if !strings.Contains(body, "isEmptyPanel") {
		t.Error("app.js: onTargetChange must define isEmptyPanel flag to branch the reveal path")
	}
	// When the incoming panel is empty the stage height target must be 0 so the
	// stage collapses smoothly rather than leaving residual whitespace.
	if !strings.Contains(body, "isEmptyPanel ? 0") {
		t.Error("app.js: empty panel transition must use incomingH=0 to collapse the stage smoothly")
	}
	// revealIncoming must guard on isEmptyPanel and return early without
	// unhiding the blank fieldset.
	if !strings.Contains(body, "if (isEmptyPanel)") {
		t.Error("app.js: revealIncoming must check isEmptyPanel and return early without showing the blank fieldset")
	}
}

// TestStaticJS_EmptyToContentTransition verifies that app.js smoothly
// animates the stage height from 0 to the incoming panel height when
// switching from a content-free panel (e.g. pop → ftp).  Without this
// branch the stage jumps directly from height:0 to height:auto, causing
// the card border to appear instantly instead of growing in.
func TestStaticJS_EmptyToContentTransition(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The grow-from-empty branch must lock the stage at '0px' before
	// triggering the transition, so CSS has an explicit start value to
	// animate from (auto→auto never animates).
	if !strings.Contains(body, "stage.style.height = '0px'") {
		t.Error("app.js: grow-from-empty branch must set stage.style.height = '0px' to give CSS transition an explicit start value")
	}
	// The branch must measure the incoming panel so the stage knows its
	// target height before the transition starts.
	if !strings.Contains(body, "!isEmptyPanel && stage") {
		t.Error("app.js: grow-from-empty branch must guard on !isEmptyPanel && stage to ensure it only runs for content panels")
	}
}

// TestStaticHTML_AdvancedOptsStructure verifies that index.html wraps the
// Advanced Options content inside .adv-body / .adv-inner elements so that
// JS-driven height + fade animations work correctly.
func TestStaticHTML_AdvancedOptsStructure(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The details element must have a stable id so initAdvancedOpts() can find it.
	if !strings.Contains(body, `id="advanced-opts"`) {
		t.Error(`index.html: <details> must have id="advanced-opts" for JS to wire up the animation`)
	}
	// .adv-body is the height-transition container (mirrors .panel-stage).
	if !strings.Contains(body, `class="adv-body"`) {
		t.Error("index.html: Advanced Options content must be wrapped in <div class=\"adv-body\"> for height animation")
	}
	// .adv-inner is the opacity+slide animation target (mirrors .target-fields inside .panel-stage).
	if !strings.Contains(body, `class="adv-inner"`) {
		t.Error("index.html: Advanced Options content must be wrapped in <div class=\"adv-inner\"> for fade+slide animation")
	}
}

// TestStaticCSS_AdvancedOptsAnimation verifies that style.css declares the
// rules required for the Advanced Options animated expand/collapse, and that
// they reuse the shared panel-appear / panel-leave keyframes and
// --panel-anim-dur token so vivid / off modes apply automatically.
func TestStaticCSS_AdvancedOptsAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The height-animated container must have overflow:hidden to clip the content.
	if !strings.Contains(body, ".advanced-opts .adv-body") {
		t.Error("style.css: .advanced-opts .adv-body rule must be declared as the height-transition container")
	}
	// Height transition must consume the shared token, not a hard-coded value.
	if !strings.Contains(body, "transition: height var(--panel-anim-dur)") {
		t.Error("style.css: .adv-body must use transition: height var(--panel-anim-dur) so vivid/off modes apply")
	}
	// Entering animation must reuse panel-appear so the feel matches panel transitions.
	if !strings.Contains(body, "adv-entering") {
		t.Error("style.css: .adv-body.adv-entering rule must be declared to trigger the entrance animation")
	}
	if !strings.Contains(body, "adv-leaving") {
		t.Error("style.css: .adv-body.adv-leaving rule must be declared to trigger the exit animation")
	}
	// Both states must delegate to the shared keyframes to avoid duplication.
	if !strings.Contains(body, "panel-appear") {
		t.Error("style.css: adv-entering animation must reuse the panel-appear keyframes")
	}
	if !strings.Contains(body, "panel-leave") {
		t.Error("style.css: adv-leaving animation must reuse the panel-leave keyframes")
	}
	// The native browser triangle marker must be suppressed so the custom
	// ::before chevron is the only visible indicator.
	if !strings.Contains(body, "::-webkit-details-marker") {
		t.Error("style.css: .advanced-opts > summary::-webkit-details-marker must be hidden to suppress the native Chrome/Safari triangle")
	}
	if !strings.Contains(body, "list-style: none") {
		t.Error("style.css: .advanced-opts > summary must set list-style:none to suppress the native Firefox triangle marker")
	}
	// summary::before must carry the animated chevron.
	if !strings.Contains(body, "summary::before") {
		t.Error("style.css: .advanced-opts > summary::before rule must exist to render the custom animated chevron")
	}
	// The native Firefox ::marker must also be suppressed (belt-and-suspenders).
	if !strings.Contains(body, "summary::marker") {
		t.Error("style.css: .advanced-opts > summary::marker must blank the native Firefox arrow")
	}
	// Chevron rotation must use ease-in-out for an elegant deceleration.
	if !strings.Contains(body, "ease-in-out") {
		t.Error("style.css: summary::before transition must use ease-in-out for a graceful rotation feel")
	}
	// Duration is driven by --panel-anim-dur (via calc) so vivid/off cascade.
	// The multiplier must be 1.2 (= original 1.8 ÷ 1.5, i.e. 50% faster).
	if !strings.Contains(body, "* 1.2") {
		t.Error("style.css: summary::before transition duration multiplier must be 1.2 (50% faster than the original 1.8x setting)")
	}
	if !strings.Contains(body, "var(--panel-anim-dur)") {
		t.Error("style.css: summary::before transition duration must consume var(--panel-anim-dur) so vivid/off modes apply automatically")
	}
	// .adv-is-open class (not [open] attribute) drives the open rotation so
	// the chevron is always in sync with the height transition direction.
	if !strings.Contains(body, "adv-is-open") {
		t.Error("style.css: .adv-is-open class must be declared to rotate the chevron in sync with the height animation")
	}
}

// TestStaticJS_AdvancedOptsAnimation verifies that app.js defines
// initAdvancedOpts() and implements the expected animated expand/collapse
// behaviour: intercepts summary clicks, drives height transition and
// adv-entering / adv-leaving CSS classes, and calls transitionend cleanup.
func TestStaticJS_AdvancedOptsAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Core function must be defined and called from DOMContentLoaded.
	if !strings.Contains(body, "initAdvancedOpts") {
		t.Error("app.js: initAdvancedOpts function must be defined to wire up the Advanced Options animation")
	}
	// The function must look up the details element by id.
	if !strings.Contains(body, `getElementById('advanced-opts')`) {
		t.Error("app.js: initAdvancedOpts must find the details element via getElementById('advanced-opts')")
	}
	// CSS classes adv-entering and adv-leaving drive the animations.
	if !strings.Contains(body, "adv-entering") {
		t.Error("app.js: initAdvancedOpts must apply adv-entering class on expand")
	}
	if !strings.Contains(body, "adv-leaving") {
		t.Error("app.js: initAdvancedOpts must apply adv-leaving class on collapse")
	}
	// details.open must be managed manually so the browser does not instantly
	// show/hide content before the animation can run.
	if !strings.Contains(body, "details.open") {
		t.Error("app.js: initAdvancedOpts must manage details.open manually to prevent instant browser toggle")
	}
	// transitionend cleanup ensures height:auto is restored after the animation
	// so the panel can resize naturally (e.g. if the viewport width changes).
	if !strings.Contains(body, "transitionend") {
		t.Error("app.js: initAdvancedOpts must listen for transitionend to restore height:auto after animation")
	}
	// e.preventDefault() prevents the browser from toggling open/closed natively.
	if !strings.Contains(body, "e.preventDefault") {
		t.Error("app.js: initAdvancedOpts click handler must call e.preventDefault() to suppress native toggle")
	}
	// adv-is-open class controls the chevron rotation and must be added at
	// expand-start and removed at collapse-start (not at transitionend) so
	// the chevron rotation is always in sync with the height animation.
	if !strings.Contains(body, "adv-is-open") {
		t.Error("app.js: initAdvancedOpts must manage adv-is-open class to drive the chevron rotation in sync with height animation")
	}
}

// TestStaticCSS_CustomCheckbox verifies that style.css replaces the native
// checkbox appearance with a fully themed custom box driven by design tokens.
// Specifically it checks:
//   - The native input is hidden (appearance:none + position:absolute + opacity:0)
//   - span::before draws the custom box sized by --cb-size token
//   - The --cb-radius token controls the corner radius
//   - The --cb-anim-dur token drives the transition so vivid/off modes apply
//   - The checked state applies the primary background colour
//   - A white SVG checkmark is embedded as a background-image data-URI
//   - A focus-visible rule adds the focus ring via box-shadow + --focus-ring token
//   - Hover states exist for both unchecked (border highlight) and checked (darken)
func TestStaticCSS_CustomCheckbox(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Design tokens must be declared so they can be overridden per theme / anim mode.
	if !strings.Contains(body, "--cb-size") {
		t.Error("style.css: --cb-size token must be declared in :root for the custom checkbox box dimensions")
	}
	if !strings.Contains(body, "--cb-radius") {
		t.Error("style.css: --cb-radius token must be declared in :root for the custom checkbox corner radius")
	}
	if !strings.Contains(body, "--cb-anim-dur") {
		t.Error("style.css: --cb-anim-dur token must be declared in :root so vivid/off animation modes apply to checkboxes")
	}
	// Vivid and off modes must each override --cb-anim-dur so the token
	// system is consistent with panel and popup animation tokens.
	// Search from the opening brace of each selector to avoid matching the
	// inline comment in :root that also contains the literal text.
	vividStart := strings.Index(body, "[data-anim=\"vivid\"] {")
	offStart := strings.Index(body, "[data-anim=\"off\"] {")
	if vividStart == -1 || !strings.Contains(body[vividStart:vividStart+400], "--cb-anim-dur") {
		t.Error("style.css: [data-anim=\"vivid\"] must override --cb-anim-dur")
	}
	if offStart == -1 || !strings.Contains(body[offStart:offStart+400], "--cb-anim-dur") {
		t.Error("style.css: [data-anim=\"off\"] must override --cb-anim-dur")
	}
	// Native checkbox must be visually hidden.
	if !strings.Contains(body, "appearance: none") {
		t.Error("style.css: .checkbox-row input[type=checkbox] must set appearance:none to suppress native rendering")
	}
	// span::before must be declared as the custom visual box target.
	if !strings.Contains(body, "input[type=checkbox] + span::before") {
		t.Error("style.css: input[type=checkbox] + span::before selector must exist to draw the custom checkbox box")
	}
	// Box dimensions must reference the --cb-size token.
	if !strings.Contains(body, "var(--cb-size)") {
		t.Error("style.css: span::before must use var(--cb-size) for width/height so the box dimension is token-driven")
	}
	// Corner radius must reference the --cb-radius token.
	if !strings.Contains(body, "var(--cb-radius)") {
		t.Error("style.css: span::before must use var(--cb-radius) for border-radius so the shape is token-driven")
	}
	// Transition must consume --cb-anim-dur so speed is token-controlled.
	if !strings.Contains(body, "var(--cb-anim-dur)") {
		t.Error("style.css: span::before transition must reference var(--cb-anim-dur)")
	}
	// Checked state must apply the primary colour.
	if !strings.Contains(body, "input[type=checkbox]:checked + span::before") {
		t.Error("style.css: :checked + span::before selector must exist to fill the box with the primary colour")
	}
	if !strings.Contains(body, "background-color: var(--primary)") {
		t.Error("style.css: checked state must set background-color: var(--primary)")
	}
	// White SVG checkmark embedded as a data-URI background-image.
	if !strings.Contains(body, "data:image/svg+xml") {
		t.Error("style.css: checked span::before must embed an SVG checkmark via background-image data-URI")
	}
	// Keyboard focus ring via :focus-visible.
	if !strings.Contains(body, "focus-visible + span::before") {
		t.Error("style.css: :focus-visible + span::before rule must be declared to show the keyboard focus ring on the custom box")
	}
	if !strings.Contains(body, "var(--focus-ring)") {
		t.Error("style.css: focus-visible rule must use var(--focus-ring) for the box-shadow so focus colour matches the global token")
	}
}

// ── Phase 4: traceroute API field assertions ──────────────────────────────

// TestStaticHTML_WebModeTracerouteRadio verifies that the embedded index.html
// includes a radio button for the "traceroute" web sub-mode so users can
// initiate a route-trace diagnostic from the UI.
func TestStaticHTML_WebModeTracerouteRadio(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The traceroute radio value must be present.
	if !strings.Contains(body, `value="traceroute"`) {
		t.Error("index.html: missing radio button with value=\"traceroute\" for route-trace mode")
	}
	// Its i18n key must be declared.
	if !strings.Contains(body, `data-i18n="web-mode-traceroute"`) {
		t.Error("index.html: traceroute radio must carry data-i18n=\"web-mode-traceroute\"")
	}
}

// TestStaticHTML_WebModeTracerouteMaxHopsPanel verifies that the traceroute
// sub-panel exists in index.html and exposes a max-hops number input so the
// user can control the maximum TTL depth.
func TestStaticHTML_WebModeTracerouteMaxHopsPanel(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The traceroute sub-panel must exist and be initially hidden.
	if !strings.Contains(body, `id="web-fields-traceroute"`) {
		t.Error("index.html: traceroute sub-panel #web-fields-traceroute must exist")
	}
	// The max-hops number input must be present inside the panel.
	if !strings.Contains(body, `id="traceroute-max-hops"`) {
		t.Error("index.html: traceroute sub-panel must contain input#traceroute-max-hops")
	}
	// Its label must use the i18n key.
	if !strings.Contains(body, `data-i18n="label-max-hops"`) {
		t.Error("index.html: max-hops label must use data-i18n=\"label-max-hops\"")
	}
}

// TestStaticI18n_WebModeTracerouteKeys verifies that both the English and
// zh-TW locales in i18n.js declare the required traceroute mode keys so the
// UI can be localised without fallback gaps.
func TestStaticI18n_WebModeTracerouteKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Both locale sections must contain the traceroute mode key.
	for _, key := range []string{"'web-mode-traceroute'", "'label-max-hops'"} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing key %s", key)
		}
	}
	// zh-TW locale must carry Chinese label for route trace.
	if !strings.Contains(body, "路由追蹤") {
		t.Error("i18n.js zh-TW: web-mode-traceroute must contain '路由追蹤'")
	}
	// zh-TW locale must carry Chinese label for max-hops.
	if !strings.Contains(body, "最大躍點數") {
		t.Error("i18n.js zh-TW: label-max-hops must contain '最大躍點數'")
	}
}

// TestStaticJS_WebModeTracerouteBuildOpts verifies that the embedded app.js
// handles the "traceroute" mode in buildWebOpts() and forwards max_hops into
// the API request payload so the server's WebOptions.MaxHops is populated.
func TestStaticJS_WebModeTracerouteBuildOpts(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The traceroute mode string constant must appear in buildWebOpts / TARGET_MODE_PANELS.
	if !strings.Contains(body, `'traceroute'`) {
		t.Error("app.js: 'traceroute' mode string must appear in buildWebOpts or TARGET_MODE_PANELS")
	}
	// The max_hops JSON field must be written into the request opts.
	if !strings.Contains(body, "max_hops") {
		t.Error("app.js: buildWebOpts must include max_hops in the traceroute mode branch")
	}
	// The traceroute sub-panel must be wired to TARGET_MODE_PANELS.
	if !strings.Contains(body, "web-fields-traceroute") {
		t.Error("app.js: TARGET_MODE_PANELS.web must include 'web-fields-traceroute' entry")
	}
}

// ── Phase 5: traceroute result rendering assertions ───────────────────────

// TestStaticJS_RenderRouteSection verifies that app.js defines a
// renderRouteSection function and wires it into renderReport so route hops
// are shown in the results pane when a traceroute diagnostic is returned.
func TestStaticJS_RenderRouteSection(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The render function must be defined.
	if !strings.Contains(body, "renderRouteSection") {
		t.Error("app.js: renderRouteSection function must be defined")
	}
	// It must be invoked from renderReport with the Route field.
	if !strings.Contains(body, "renderRouteSection(r.Route)") {
		t.Error("app.js: renderReport must call renderRouteSection(r.Route)")
	}
	// The route section heading i18n key must be referenced.
	if !strings.Contains(body, "'section-route'") {
		t.Error("app.js: renderRouteSection must reference i18n key 'section-route'")
	}
	// Timed-out hop indicator must be present.
	if !strings.Contains(body, "hop-timedout") {
		t.Error("app.js: renderRouteSection must apply 'hop-timedout' class to timed-out hops")
	}
}

// TestStaticCSS_RouteTable verifies that style.css contains the CSS rules
// needed to style the route-trace hop table and distinguish timed-out hops.
func TestStaticCSS_RouteTable(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The route-table modifier class must be present.
	if !strings.Contains(body, ".route-table") {
		t.Error("style.css: .route-table modifier class must exist for the route hop table")
	}
	// The hop-timedout rule must be present to style unresponsive hops.
	if !strings.Contains(body, ".hop-timedout") {
		t.Error("style.css: .hop-timedout rule must exist for timed-out traceroute hops")
	}
}

// TestStaticI18n_RouteSectionKeys verifies that both the English and zh-TW
// locales in i18n.js declare all keys required by renderRouteSection to
// produce a fully-localised hop table without fallback gaps.
func TestStaticI18n_RouteSectionKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, key := range []string{
		"'section-route'", "'th-ttl'", "'th-ip-host'", "'th-asn'", "'th-country'",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing route section key %s", key)
		}
	}
	// zh-TW locale must carry a Chinese section title.
	if !strings.Contains(body, "路由路徑") {
		t.Error("i18n.js zh-TW: section-route must contain '路由路徑'")
	}
	// zh-TW locale must carry Chinese column header for IP / Host.
	if !strings.Contains(body, "IP / 主機") {
		t.Error("i18n.js zh-TW: th-ip-host must contain 'IP / 主機'")
	}
}

// ── animation & error-message tests ──────────────────────────────────────

// TestStaticCSS_RunAnimation verifies that style.css defines the dots
// run-button animation class and its associated @keyframes.
func TestStaticCSS_RunAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Dots animation must be defined (used as the run-button loading state).
	if !strings.Contains(body, ".anim-dots") {
		t.Error("style.css: .anim-dots animation class must be defined")
	}
	if !strings.Contains(body, "@keyframes anim-dots-bounce") {
		t.Error("style.css: @keyframes anim-dots-bounce must be declared")
	}
	// Spinner must also be present (used elsewhere in the UI).
	if !strings.Contains(body, ".spinner") {
		t.Error("style.css: .spinner class must be defined")
	}
	if !strings.Contains(body, "@keyframes spin") {
		t.Error("style.css: @keyframes spin must be declared")
	}
	// The temporary animation picker and its removed sibling animations must
	// no longer exist in the stylesheet.
	for _, removed := range []string{".anim-picker", ".anim-opt", ".anim-pulse", ".anim-wave"} {
		if strings.Contains(body, removed) {
			t.Errorf("style.css: removed animation/picker rule %q must not be present", removed)
		}
	}
}

// TestStaticCSS_AutofillTheme verifies that style.css overrides the browser
// autofill background colour so the site theme is preserved when the browser
// fills in a previously entered value for the target-host input.
func TestStaticCSS_AutofillTheme(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, ":-webkit-autofill") {
		t.Error("style.css: :-webkit-autofill rules must be present to prevent browser autofill overriding the theme background")
	}
	// The override must use a box-shadow inset trick (the only cross-browser
	// approach that defeats the UA fill colour without disabling autofill).
	if !strings.Contains(body, "inset !important") {
		t.Error("style.css: autofill override must use 'inset !important' box-shadow technique")
	}
	// Text colour must also be explicitly restored.
	if !strings.Contains(body, "-webkit-text-fill-color") {
		t.Error("style.css: autofill override must set -webkit-text-fill-color to restore text colour")
	}
}

// TestStaticJS_DotsRunAnimation verifies that app.js always injects the
// dots animation markup into #run-btn and that the picker system has been
// removed in favour of the fixed dots choice.
func TestStaticJS_DotsRunAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// getRunningHTML must exist and emit the dots markup.
	if !strings.Contains(body, "getRunningHTML") {
		t.Error("app.js: getRunningHTML must be defined")
	}
	if !strings.Contains(body, "anim-dots") {
		t.Error("app.js: getRunningHTML must return anim-dots markup")
	}
	// Picker management functions must have been removed.
	for _, removed := range []string{"RUN_ANIMATIONS", "initRunAnimation", "setRunAnimation", "_syncAnimPicker"} {
		if strings.Contains(body, removed) {
			t.Errorf("app.js: removed animation picker symbol %q must not be present", removed)
		}
	}
	// picker HTML must not be present.
	if strings.Contains(body, "id=\"anim-picker\"") {
		t.Error("index.html: #anim-picker must have been removed")
	}
}

// TestStaticJS_ErrorClearsProgressLog verifies that app.js clears and hides
// the progress log both on SSE error events and on network-level failures, so
// partial traceroute output does not remain visible below the error banner.
func TestStaticJS_ErrorClearsProgressLog(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Both the catch block and the SSE error handler must clear innerHTML.
	// We verify by counting occurrences of the clear pattern.
	clearPattern := "progressEl.innerHTML = ''"
	count := strings.Count(body, clearPattern)
	if count < 2 {
		t.Errorf("app.js: progressEl.innerHTML='' must appear in both the catch block and the SSE error handler; found %d occurrence(s)", count)
	}
}

// TestStaticJS_TracerouteTimeoutAutoCompute verifies that app.js contains the
// logic to auto-compute a traceroute-appropriate timeout before sending the
// diagnostic request, preventing spurious deadline-exceeded errors.
func TestStaticJS_TracerouteTimeoutAutoCompute(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The traceroute-specific timeout guard must be present.
	if !strings.Contains(body, "traceroute") {
		t.Error("app.js: must contain 'traceroute' reference for mode-specific timeout logic")
	}
	if !strings.Contains(body, "parseTimeoutSec") {
		t.Error("app.js: parseTimeoutSec helper must be defined for timeout comparison")
	}
}

// TestStaticJS_LocalizeError verifies that app.js defines the localizeError
// function to map raw server error strings to user-friendly i18n messages,
// replacing opaque Go internal strings like "context deadline exceeded".
func TestStaticJS_LocalizeError(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "localizeError") {
		t.Error("app.js: localizeError function must be defined")
	}
	// Must check for the deadline exceeded pattern.
	if !strings.Contains(body, "deadline exceeded") {
		t.Error("app.js: localizeError must handle 'deadline exceeded' error pattern")
	}
	// Must use the err-timeout i18n key for timeout errors.
	if !strings.Contains(body, "err-timeout") {
		t.Error("app.js: localizeError must reference 'err-timeout' i18n key for timeout errors")
	}
}

// TestStaticI18n_ErrorMessageKeys verifies that the embedded i18n.js contains
// user-friendly error message keys in both English and zh-TW locales.
func TestStaticI18n_ErrorMessageKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// All error keys must be present in the file.
	for _, key := range []string{"'err-timeout'", "'err-no-runner'", "'err-unknown'"} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing error key %s", key)
		}
	}
	// English locale must carry user-friendly timeout text (not raw Go error).
	if !strings.Contains(body, "timed out") {
		t.Error("i18n.js en: err-timeout must contain 'timed out' for user-friendly display")
	}
	// zh-TW locale must carry a Chinese timeout message.
	if !strings.Contains(body, "診斷逾時") {
		t.Error("i18n.js zh-TW: err-timeout must contain '診斷逾時'")
	}
}

// TestStaticCSS_ErrorBannerFlex verifies that the updated error-banner uses
// flexbox layout (with .error-icon and .error-text children) for better visual
// separation between the icon and the message text.
func TestStaticCSS_ErrorBannerFlex(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// error-banner must use flex layout for icon + text alignment.
	if !strings.Contains(body, ".error-banner") {
		t.Error("style.css: .error-banner rule must be defined")
	}
	if !strings.Contains(body, ".error-icon") {
		t.Error("style.css: .error-icon rule must be defined inside .error-banner")
	}
	if !strings.Contains(body, ".error-text") {
		t.Error("style.css: .error-text rule must be defined inside .error-banner")
	}
}

// TestStaticHTML_ErrorBannerStructure verifies that index.html contains the
// structured error banner with role="alert" and separate icon/text spans.
func TestStaticHTML_ErrorBannerStructure(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `id="error-banner"`) {
		t.Error("index.html: #error-banner must be present")
	}
	if !strings.Contains(body, `role="alert"`) {
		t.Error("index.html: #error-banner must declare role=\"alert\" for screen-reader accessibility")
	}
	if !strings.Contains(body, `class="error-icon"`) {
		t.Error("index.html: .error-icon span must be present inside #error-banner")
	}
	if !strings.Contains(body, `id="error-text"`) {
		t.Error("index.html: #error-text span must be present inside #error-banner")
	}
}

// TestStaticHTML_ErrorBannerHiddenByDefault verifies that the error banner in
// index.html carries the `hidden` attribute so it is invisible on page load and
// only becomes visible when JS calls showError().
func TestStaticHTML_ErrorBannerHiddenByDefault(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The banner element with its hidden attribute must appear together.
	if !strings.Contains(body, `id="error-banner" hidden`) {
		t.Error("index.html: #error-banner must carry the `hidden` attribute so it is invisible on load")
	}
}

// TestStaticCSS_HiddenAttributeEnforced verifies that style.css declares a
// [hidden] reset rule with !important so that component-level display
// properties (e.g. display:flex on .error-banner) cannot override the HTML
// hidden attribute and show elements that should be invisible.
func TestStaticCSS_HiddenAttributeEnforced(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The reset rule must use !important so it wins over component display rules.
	if !strings.Contains(body, "[hidden]") {
		t.Error("style.css: [hidden] reset rule must be declared")
	}
	if !strings.Contains(body, "display: none !important") {
		t.Error("style.css: [hidden] rule must use 'display: none !important' to override component display values")
	}
}

// TestStaticCSS_RunBtnCentering verifies that style.css correctly centres both
// the run-button resting state (▶ glyph) and its loading state (dots animation)
// by enforcing line-height:1 on #run-btn and removing the margin offset from
// .anim-dots when it is a child of #run-btn.
func TestStaticCSS_RunBtnCentering(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// line-height:1 must be set so the inherited body line-height (1.5) does
	// not add extra leading that shifts the glyph off the vertical centre.
	if !strings.Contains(body, "line-height: 1") {
		t.Error("style.css: #run-btn must set line-height: 1 for pixel-perfect vertical centering")
	}
	// The context-specific margin reset ensures the dots animation is not
	// shifted horizontally by its default margin-right value.
	if !strings.Contains(body, "#run-btn .anim-dots") {
		t.Error("style.css: #run-btn .anim-dots override must be defined to remove the inline-context margin")
	}
	if !strings.Contains(body, "margin: 0") {
		t.Error("style.css: #run-btn .anim-dots must set margin: 0 to restore flex centering symmetry")
	}
}

// TestStaticHTML_PortsFieldGroup verifies that the redesigned form layout places
// target-type, host, and port-group in ONE unified form-grid row.  The port-group
// hosts a shared text input used by both web/port mode and non-web targets.
func TestStaticHTML_PortsFieldGroup(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The unified port-group column must exist, initially hidden
	// (default target = web/public-ip which doesn't need port selection).
	if !strings.Contains(body, `id="port-group" hidden`) {
		t.Error("index.html: #port-group must be present and initially hidden (default web/public-ip needs no ports)")
	}
	// The shared text-input variant must exist inside port-group.
	if !strings.Contains(body, `id="ports-text-group" hidden`) {
		t.Error("index.html: #ports-text-group must be present inside #port-group")
	}
	// The removed checkbox picker must NOT appear in the HTML.
	if strings.Contains(body, `id="web-port-picker"`) {
		t.Error("index.html: #web-port-picker checkbox picker has been removed; it must not appear in the HTML")
	}
	// host and ports inputs must still be reachable by their existing IDs.
	if !strings.Contains(body, `id="host"`) {
		t.Error("index.html: #host input must be present")
	}
	if !strings.Contains(body, `id="ports"`) {
		t.Error("index.html: #ports input must be present")
	}
}

// TestStaticHTML_PortGroupLabelHint verifies that the #port-group label displays
// the "Ports" text and "(comma-separated)" hint inline as a <small> element
// inside the <label> — matching the same visual pattern used by other fields
// (e.g. DNS Domains, SMTP RCPT TO).  The hint must NOT appear as a standalone
// sibling of the <input> inside #ports-text-group.
func TestStaticHTML_PortGroupLabelHint(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The label inside #port-group must embed the hint as a <small> element.
	const wantInlineHint = `<span data-i18n="label-ports">Ports</span> <small data-i18n="label-ports-hint">(comma-separated)</small></label>`
	if !strings.Contains(body, wantInlineHint) {
		t.Error(`index.html: #port-group label must contain inline <small data-i18n="label-ports-hint"> hint`)
	}
	// The hint must NOT appear as a standalone sibling of the <input> inside
	// #ports-text-group (it would duplicate the inline label hint).
	portTextGroupStart := strings.Index(body, `id="ports-text-group"`)
	if portTextGroupStart == -1 {
		t.Fatal("index.html: #ports-text-group element not found")
	}
	// Find the closing </div> of #ports-text-group (next </div> after its open tag).
	portTextGroupEnd := strings.Index(body[portTextGroupStart:], "</div>")
	if portTextGroupEnd == -1 {
		t.Fatal("index.html: closing </div> for #ports-text-group not found")
	}
	textGroupBody := body[portTextGroupStart : portTextGroupStart+portTextGroupEnd]
	if strings.Contains(textGroupBody, `data-i18n="label-ports-hint"`) {
		t.Error(`index.html: <small data-i18n="label-ports-hint"> must not appear inside #ports-text-group (it belongs in the parent <label> instead)`)
	}
}

// TestStaticJS_WebPortModeReadsTextInput verifies that app.js handles the
// web/port mode using the shared text input (val('ports')) instead of the
// removed checkbox picker.  getWebPorts() must no longer exist in the codebase.
func TestStaticJS_WebPortModeReadsTextInput(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// getWebPorts() has been removed; buildRequest reads val('ports') for web/port.
	if strings.Contains(body, "function getWebPorts(") {
		t.Error("app.js: getWebPorts() must be removed; web/port mode now uses the shared text input")
	}
	// buildRequest must use WEB_MODES_WITH_PORTS to decide whether to read ports.
	if !strings.Contains(body, "WEB_MODES_WITH_PORTS.includes(mode)") {
		t.Error("app.js: buildRequest must guard web port reading with WEB_MODES_WITH_PORTS.includes(mode)")
	}
	// The removed picker elements must not be referenced in JS logic.
	if strings.Contains(body, "getElementById('port-other-cb')") {
		t.Error("app.js: port-other-cb has been removed and must not be referenced")
	}
	if strings.Contains(body, "getElementById('port-other-num')") {
		t.Error("app.js: port-other-num has been removed and must not be referenced")
	}
}

// TestStaticJS_WebTargetPortDefaults verifies that TARGET_PORTS.web includes
// both port 80 (HTTP) and port 443 (HTTPS) as the auto-fill defaults shown
// when the user selects web target + port connectivity mode.
func TestStaticJS_WebTargetPortDefaults(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// TARGET_PORTS.web must include both HTTP (80) and HTTPS (443) defaults.
	if !strings.Contains(body, "web:  [80, 443]") {
		t.Error("app.js: TARGET_PORTS.web must be [80, 443] (HTTP + HTTPS defaults for port-connectivity mode)")
	}
}

// TestStaticJS_PortGroupModeAutoFill verifies that app.js auto-fills the ports
// text input when the user switches a web radio to the port-connectivity mode
// (mirrors the auto-fill onTargetChange() already does for target switches).
func TestStaticJS_PortGroupModeAutoFill(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The radio change handler must auto-fill ports for web/port mode.
	if !strings.Contains(body, "WEB_MODES_WITH_PORTS.includes(mode)") {
		t.Error("app.js: radio change handler must check WEB_MODES_WITH_PORTS to auto-fill ports")
	}
	// Must respect the userEdited guard so manual entries are preserved.
	if !strings.Contains(body, `dataset.userEdited !== 'true'`) {
		t.Error("app.js: radio change handler must respect dataset.userEdited guard before auto-filling")
	}
}

// TestStaticJS_PortGroupToggle verifies that app.js manages #port-group
// visibility via updatePortGroup(), which is driven by the WEB_MODES_WITH_PORTS
// constant so logic is data-driven rather than hardcoded per-mode.
func TestStaticJS_PortGroupToggle(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The unified port-group ID must be referenced.
	if !strings.Contains(body, "port-group") {
		t.Error("app.js: must reference 'port-group' to toggle Ports column visibility")
	}
	// updatePortGroup must be callable from both onTargetChange and the radio handler.
	if !strings.Contains(body, "updatePortGroup(") {
		t.Error("app.js: updatePortGroup() must be called from onTargetChange and radio change handler")
	}
}

// TestStaticJS_WEB_MODES_WITH_PORTS verifies that app.js declares the
// WEB_MODES_WITH_PORTS constant used to drive port-group visibility in a
// data-driven, non-hardcoded manner.
func TestStaticJS_WEB_MODES_WITH_PORTS(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "WEB_MODES_WITH_PORTS") {
		t.Error("app.js: WEB_MODES_WITH_PORTS constant must be declared")
	}
	// Port connectivity mode must be listed as requiring port selection.
	if !strings.Contains(body, "'port'") {
		t.Error("app.js: WEB_MODES_WITH_PORTS must include 'port' mode")
	}
}

// TestStaticJS_UpdatePortGroup verifies that app.js declares the updatePortGroup()
// function which manages visibility of #port-group and its inner variants.
func TestStaticJS_UpdatePortGroup(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function updatePortGroup(") {
		t.Error("app.js: updatePortGroup() function must be defined")
	}
	// Must reference all three DOM elements it manages.
	for _, id := range []string{"port-group", "ports-text-group"} {
		if !strings.Contains(body, id) {
			t.Errorf("app.js: updatePortGroup() must reference element #%s", id)
		}
	}
	// The removed checkbox picker must no longer be referenced in updatePortGroup.
	if strings.Contains(body, "getElementById('web-port-picker')") {
		t.Error("app.js: web-port-picker has been removed; updatePortGroup must not reference it")
	}
}

// TestStaticJS_RenderMapInvalidateSize verifies that renderMap() defers a call
// to _map.invalidateSize() via requestAnimationFrame so Leaflet re-projects all
// tiles after the #results section transitions from display:none to display:block.
// Without this, Leaflet sees a 0×0 container at init time and leaves large
// blank grey areas on the OpenStreetMap canvas.
func TestStaticJS_RenderMapInvalidateSize(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// renderMap must call invalidateSize to correct blank-tile regression.
	if !strings.Contains(body, "invalidateSize") {
		t.Error("app.js: renderMap must call _map.invalidateSize() to fix blank tile regression when container was hidden")
	}
	// The call must be deferred via requestAnimationFrame so it runs after the
	// browser has re-laid-out the newly visible container.
	if !strings.Contains(body, "requestAnimationFrame") {
		t.Error("app.js: invalidateSize must be deferred via requestAnimationFrame so layout is complete before tiles repaint")
	}
}

// TestStaticJS_SSEResultRevealOrder verifies that in the SSE 'result' event
// handler, resultEl.hidden = false is set BEFORE renderMap() is called.
// Leaflet initialises by reading the container's layout dimensions; if the
// parent #results section is still hidden (display:none) at that point, the
// map gets a 0×0 size and tiles are blank.
func TestStaticJS_SSEResultRevealOrder(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Locate handleSSEMessage and the evtName==='result' branch within it.
	fnStart := strings.Index(body, "function handleSSEMessage(")
	if fnStart == -1 {
		t.Fatal("app.js: handleSSEMessage function not found")
	}
	resultBranchIdx := strings.Index(body[fnStart:], "evtName === 'result'")
	if resultBranchIdx == -1 {
		t.Fatal("app.js: evtName === 'result' branch not found in handleSSEMessage")
	}
	// Inspect a window large enough to cover the result branch body.
	windowStart := fnStart + resultBranchIdx
	window := body[windowStart : windowStart+600]

	hiddenIdx := strings.Index(window, "resultEl.hidden = false")
	renderMapIdx := strings.Index(window, "renderMap(")
	if hiddenIdx == -1 {
		t.Fatal("app.js: resultEl.hidden = false not found in SSE result branch")
	}
	if renderMapIdx == -1 {
		t.Fatal("app.js: renderMap( not found in SSE result branch")
	}
	if hiddenIdx > renderMapIdx {
		t.Error("app.js: resultEl.hidden = false must appear BEFORE renderMap() in the SSE result handler — " +
			"#results must be visible so the Leaflet container has layout dimensions")
	}
}

// TestStaticJS_HistoryEntryRevealOrder verifies that in loadHistoryEntry(),
// resultEl.hidden = false is set BEFORE renderMap() for the same reason as
// in the SSE handler.
func TestStaticJS_HistoryEntryRevealOrder(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	fnStart := strings.Index(body, "function loadHistoryEntry(")
	if fnStart == -1 {
		t.Fatal("app.js: loadHistoryEntry function not found")
	}
	// Bound the search to the function body (next top-level function boundary).
	// nextFn is relative to body[fnStart:], so add fnStart to get absolute end.
	nextFn := strings.Index(body[fnStart+1:], "\nasync function ")
	var fnBody string
	if nextFn != -1 && (fnStart+1+nextFn) <= len(body) {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		// Fallback: take up to 1200 chars, capped at body length.
		end := fnStart + 1200
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	hiddenIdx := strings.Index(fnBody, "resultEl.hidden = false")
	renderMapIdx := strings.Index(fnBody, "renderMap(")
	if hiddenIdx == -1 {
		t.Fatal("app.js: resultEl.hidden = false not found in loadHistoryEntry")
	}
	if renderMapIdx == -1 {
		t.Fatal("app.js: renderMap( not found in loadHistoryEntry")
	}
	if hiddenIdx > renderMapIdx {
		t.Error("app.js: resultEl.hidden = false must appear BEFORE renderMap() in loadHistoryEntry — " +
			"#results must be visible so the Leaflet container has layout dimensions")
	}
}

// TestStaticJS_MapPointConfigs verifies that app.js declares MAP_POINT_CONFIGS
// with 'origin' and 'target' keys, forming a data-driven foundation for all
// map marker styling.  Callers derive visual behaviour from this object rather
// than hardcoding logic inside renderMap().
func TestStaticJS_MapPointConfigs(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "MAP_POINT_CONFIGS") {
		t.Error("app.js: MAP_POINT_CONFIGS constant not found — map marker config must be data-driven")
	}
	if !strings.Contains(body, "'origin'") {
		t.Error("app.js: MAP_POINT_CONFIGS must include an 'origin' key for the public-IP marker")
	}
	if !strings.Contains(body, "'target'") {
		t.Error("app.js: MAP_POINT_CONFIGS must include a 'target' key for the destination marker")
	}
}

// TestStaticJS_HaversineKm verifies that app.js defines a haversineKm()
// helper for computing the great-circle distance.  This powers the distance
// badge displayed below the map between origin and target markers.
func TestStaticJS_HaversineKm(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function haversineKm(") {
		t.Error("app.js: haversineKm function not found — distance calculation must be a named helper")
	}
	// Earth radius constant must appear to confirm correct formula.
	if !strings.Contains(body, "6371") {
		t.Error("app.js: haversineKm must use Earth radius constant 6371 km")
	}
}

// TestStaticJS_BuildMarkerIcon verifies that app.js defines buildMarkerIcon()
// which creates L.divIcon instances driven by MAP_POINT_CONFIGS, replacing the
// default Leaflet marker pin with a role-coloured dot.
func TestStaticJS_BuildMarkerIcon(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function buildMarkerIcon(") {
		t.Error("app.js: buildMarkerIcon function not found — marker icon creation must be a named helper")
	}
	if !strings.Contains(body, "L.divIcon(") {
		t.Error("app.js: buildMarkerIcon must use L.divIcon for custom marker styling")
	}
}

// TestStaticJS_BuildPopupHtml verifies that app.js defines buildPopupHtml()
// which constructs a rich HTML popup from a GeoAnnotation, using the
// geo-popup__role badge to clearly identify origin vs target.
func TestStaticJS_BuildPopupHtml(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function buildPopupHtml(") {
		t.Error("app.js: buildPopupHtml function not found")
	}
	if !strings.Contains(body, "geo-popup__role") {
		t.Error("app.js: buildPopupHtml must emit .geo-popup__role element for visual role identification")
	}
	if !strings.Contains(body, "geo-popup__ip") {
		t.Error("app.js: buildPopupHtml must emit .geo-popup__ip element for the IP address")
	}
}

// TestStaticJS_RenderMapPolyline verifies that renderMap() draws a L.polyline
// between origin and target to give users a clear visual probe direction.
func TestStaticJS_RenderMapPolyline(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "L.polyline(") {
		t.Error("app.js: renderMap must draw a L.polyline to connect origin and target markers")
	}
	if !strings.Contains(body, "dashArray") {
		t.Error("app.js: the connection polyline must use dashArray for a visual route style")
	}
}

// TestStaticHTML_GeoDistanceElement verifies that index.html includes the
// #geo-distance element, which renderMap() populates with the great-circle
// distance between origin and target.
func TestStaticHTML_GeoDistanceElement(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `id="geo-distance"`) {
		t.Error("index.html: #geo-distance element not found — required for the map distance badge")
	}
}

// TestStaticCSS_GeoMarkerStyles verifies that style.css defines the custom
// marker dot classes used by buildMarkerIcon() via L.divIcon.
func TestStaticCSS_GeoMarkerStyles(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, cls := range []string{".geo-marker--origin", ".geo-marker--target", ".geo-marker__dot"} {
		if !strings.Contains(body, cls) {
			t.Errorf("style.css: class %q not found — required for custom Leaflet divIcon styling", cls)
		}
	}
}

// TestStaticCSS_GeoLegendAndDistance verifies that style.css defines the
// .geo-legend and .geo-distance classes used by the in-map legend control and
// the distance badge below the map.
func TestStaticCSS_GeoLegendAndDistance(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, cls := range []string{".geo-legend", ".geo-legend__item", ".geo-legend__dot", ".geo-distance"} {
		if !strings.Contains(body, cls) {
			t.Errorf("style.css: class %q not found — required for map legend / distance badge", cls)
		}
	}
}

// TestStaticI18n_MapOriginAndDistanceKeys verifies that both the 'en' and 'zh'
// locales in i18n.js expose the 'map-origin' and 'map-distance' keys introduced
// for the enhanced map UX.  Each key must appear at least twice (once per locale).
func TestStaticI18n_MapOriginAndDistanceKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, key := range []string{"'map-origin'", "'map-distance'"} {
		first := strings.Index(body, key)
		if first == -1 {
			t.Errorf("i18n.js: key %s not found in any locale", key)
			continue
		}
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh", key)
		}
	}
}

// TestStaticJS_TileLayerConfigs verifies that app.js declares TILE_LAYER_CONFIGS
// with both 'light' and 'dark' variants pointing to the CARTO basemap service.
// Tile URLs must not use the raw OSM URL so theme-aware switching works correctly.
func TestStaticJS_TileLayerConfigs(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "TILE_LAYER_CONFIGS") {
		t.Error("app.js: TILE_LAYER_CONFIGS constant not found — tile URLs must be data-driven")
	}
	if !strings.Contains(body, "'light'") {
		t.Error("app.js: TILE_LAYER_CONFIGS must include a 'light' variant")
	}
	if !strings.Contains(body, "'dark'") {
		t.Error("app.js: TILE_LAYER_CONFIGS must include a 'dark' variant")
	}
	// CARTO attribution must be present to satisfy the tile provider's terms.
	if !strings.Contains(body, "carto.com/attributions") {
		t.Error("app.js: CARTO attribution URL must be present in TILE_LAYER_CONFIGS")
	}
	// OSM is now a supported variant inside TILE_LAYER_CONFIGS; its URL is data-driven
	// and must appear inside that config block, not hardcoded in renderMap.
	if !strings.Contains(body, "tile.openstreetmap.org") {
		t.Error("app.js: tile.openstreetmap.org URL must appear in TILE_LAYER_CONFIGS as the osm variant")
	}
}

// TestStaticJS_MapDarkThemes verifies that app.js declares MAP_DARK_THEMES as
// the authoritative set of theme IDs that map to the dark tile variant.
func TestStaticJS_MapDarkThemes(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "MAP_DARK_THEMES") {
		t.Error("app.js: MAP_DARK_THEMES constant not found — dark/light tile selection must be data-driven")
	}
	// The known dark themes must be listed.
	for _, id := range []string{"'dark'", "'deep-blue'", "'forest-green'"} {
		cfg := strings.Index(body, "MAP_DARK_THEMES")
		if cfg == -1 {
			break
		}
		// look for the id somewhere after MAP_DARK_THEMES declaration
		if !strings.Contains(body[cfg:cfg+200], id) {
			t.Errorf("app.js: MAP_DARK_THEMES must include theme id %s", id)
		}
	}
}

// TestStaticJS_GetMapTileVariant verifies that app.js exposes a named
// getMapTileVariant() function which is the single decision point for
// mapping the active application theme to a tile-layer variant string.
func TestStaticJS_GetMapTileVariant(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function getMapTileVariant(") {
		t.Error("app.js: getMapTileVariant function not found")
	}
}

// TestStaticJS_RefreshMapTiles verifies that app.js exposes a named
// refreshMapTiles() function that swaps the tile layer on the live map
// with a fade-out/fade-in animation.  It is called only from
// setMapTileVariant() (user-driven tile changes).  Theme-triggered tile swaps
// are handled silently by syncMapTileVariantToTheme().
func TestStaticJS_RefreshMapTiles(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function refreshMapTiles(") {
		t.Error("app.js: refreshMapTiles function not found")
	}
}

// TestStaticJS_ApplyThemeCallsRefreshMapTiles verifies that applyTheme() ensures
// map tiles are refreshed when the colour theme changes.  The function may do
// this directly (refreshMapTiles()) or via syncMapTileVariantToTheme(), which
// itself calls refreshMapTiles() internally.
func TestStaticJS_ApplyThemeCallsRefreshMapTiles(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("app.js: applyTheme function not found")
	}
	// Find the closing brace of applyTheme by scanning for the next top-level function.
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// applyTheme must trigger a tile refresh either directly or via syncMapTileVariantToTheme.
	if !strings.Contains(fnBody, "refreshMapTiles()") && !strings.Contains(fnBody, "syncMapTileVariantToTheme(") {
		t.Error("app.js: applyTheme must call refreshMapTiles() or syncMapTileVariantToTheme() so tile layer updates on theme change")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 fix tests — theme fade / input colour / map-bar visibility / tile swap
// ---------------------------------------------------------------------------

// TestStaticCSS_BodyIncludesOpacityTransition verifies that the theme-fade
// opacity transition is applied to .main (not body) so that applyTheme()'s
// transitionend listener fires correctly when only the main content area fades.
// The body rule itself must NOT carry opacity, since header and footer must
// remain visible during theme switches and use their own colour transitions.
func TestStaticCSS_BodyIncludesOpacityTransition(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// .main must include an opacity transition so body.theme-transitioning .main
	// triggers a CSS transition (and thus fires transitionend on the element).
	mainIdx := strings.Index(css, ".main {")
	if mainIdx == -1 {
		t.Fatal("style.css: .main rule not found")
	}
	endIdx := strings.Index(css[mainIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: .main rule closing brace not found")
	}
	mainBlock := css[mainIdx : mainIdx+endIdx+1]
	if !strings.Contains(mainBlock, "opacity") {
		t.Error("style.css: .main transition must include 'opacity' so theme-transitioning fade works (transitionend fires on .main)")
	}
}

// TestStaticCSS_InputBaseCaretColor verifies that the base input rule (outside
// of the :-webkit-autofill override) explicitly sets caret-color so the text
// insertion cursor stays theme-coloured even in dark themes.
func TestStaticCSS_InputBaseCaretColor(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	autofillIdx := strings.Index(css, ":-webkit-autofill")
	if autofillIdx == -1 {
		t.Fatal("style.css: :-webkit-autofill rule not found")
	}
	// caret-color must appear BEFORE the autofill override so we know it is in
	// the base input rule, not only as part of the autofill emergency patch.
	beforeAutofill := css[:autofillIdx]
	if !strings.Contains(beforeAutofill, "caret-color: var(--text)") {
		t.Error("style.css: base input rule must set caret-color: var(--text) — not only in the autofill override — to keep the cursor visible in dark themes")
	}
}

// TestStaticCSS_InputBaseTextFillColor verifies that the base input rule sets
// -webkit-text-fill-color so dark-theme text remains readable when the browser
// applies autocomplete suggestion overlay styles.
func TestStaticCSS_InputBaseTextFillColor(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	autofillIdx := strings.Index(css, ":-webkit-autofill")
	if autofillIdx == -1 {
		t.Fatal("style.css: :-webkit-autofill rule not found")
	}
	beforeAutofill := css[:autofillIdx]
	if !strings.Contains(beforeAutofill, "-webkit-text-fill-color: var(--text)") {
		t.Error("style.css: base input rule must set -webkit-text-fill-color: var(--text) to prevent dark-theme text appearing black")
	}
}

// TestStaticCSS_RadiusTokenDefined verifies that --radius is defined in :root
// so all component rules that use var(--radius) resolve to a valid value.
// A missing token causes silent fallback to 'initial' (no border-radius).
func TestStaticCSS_RadiusTokenDefined(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// --radius must be assigned inside :root.
	rootStart := strings.Index(css, ":root {")
	if rootStart == -1 {
		t.Fatal("style.css: :root block not found")
	}
	rootEnd := strings.Index(css[rootStart:], "\n}")
	if rootEnd == -1 {
		t.Fatal("style.css: :root closing brace not found")
	}
	rootBlock := css[rootStart : rootStart+rootEnd]
	if !strings.Contains(rootBlock, "--radius") {
		t.Error("style.css: --radius must be defined inside :root so var(--radius) components resolve correctly")
	}
}

// TestStaticJS_ApplyThemeFiltersOpacityEvent verifies that applyTheme() uses
// e.propertyName to guard the transitionend handler so only the body's own
// opacity transition — not background/color transitions or bubbling child
// events — triggers the theme swap.
func TestStaticJS_ApplyThemeFiltersOpacityEvent(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("app.js: applyTheme function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1200
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "propertyName") {
		t.Error("app.js: applyTheme transitionend handler must check e.propertyName to filter the correct transition event")
	}
	if !strings.Contains(fnBody, "'opacity'") {
		t.Error("app.js: applyTheme must guard transitionend with e.propertyName === 'opacity'")
	}
}

// TestStaticJS_MapBarHiddenToggled verifies that renderMap() removes the hidden
// attribute from #geo-map-outer when the map is shown and sets it when hidden,
// so the tile-variant selector bar (inside the outer wrapper) is visible exactly
// when the map is visible.
func TestStaticJS_MapBarHiddenToggled(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("app.js: renderMap function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 3000
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	// Both show and hide paths must reference the outer wrapper element and toggle hidden.
	if !strings.Contains(fnBody, "geo-map-outer") {
		t.Error("app.js: renderMap must reference geo-map-outer to toggle its visibility")
	}
	if !strings.Contains(fnBody, "hidden = false") && !strings.Contains(fnBody, "removeAttribute('hidden')") {
		t.Error("app.js: renderMap must reveal #geo-map-outer (hidden = false) when map is shown")
	}
	if !strings.Contains(fnBody, "hidden = true") && !strings.Contains(fnBody, "setAttribute('hidden'") {
		t.Error("app.js: renderMap must hide #geo-map-outer (hidden = true) when map is hidden")
	}
}

// TestStaticJS_RefreshMapTilesRequestAnimationFrame verifies that the updated
// refreshMapTiles() uses requestAnimationFrame to remove the fading class after
// the tile swap, rather than registering a second transitionend listener that
// would never fire (since removing the class triggers the transition, not ends it).
func TestStaticJS_RefreshMapTilesRequestAnimationFrame(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function refreshMapTiles(")
	if fnStart == -1 {
		t.Fatal("app.js: refreshMapTiles function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "requestAnimationFrame") {
		t.Error("app.js: refreshMapTiles must use requestAnimationFrame to remove geo-map--fading after tile swap")
	}
	if !strings.Contains(fnBody, "propertyName") {
		t.Error("app.js: refreshMapTiles transitionend handler must filter by e.propertyName to avoid acting on bubbling child events")
	}
}

// TestStaticJS_SyncMapTileVariantNoFadeAnimation verifies that
// syncMapTileVariantToTheme() does NOT call refreshMapTiles(), ensuring the
// theme-driven tile swap is always silent (no map fade animation).  The fade
// would be redundant because the body is already invisible during a theme
// transition, and the second transitionend listener in the old refreshMapTiles
// would leave geo-map--fading stuck permanently.
func TestStaticJS_SyncMapTileVariantNoFadeAnimation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function syncMapTileVariantToTheme(")
	if fnStart == -1 {
		t.Fatal("app.js: syncMapTileVariantToTheme function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 600
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	// Must NOT delegate to animated refreshMapTiles — silent swap only.
	if strings.Contains(fnBody, "refreshMapTiles()") {
		t.Error("app.js: syncMapTileVariantToTheme must NOT call refreshMapTiles() — tile swap must be silent during theme transitions")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 — theme-fade / map-tile-bar tests
// ---------------------------------------------------------------------------

// TestStaticJS_MapThemeToTileVariant verifies that app.js declares
// MAP_THEME_TO_TILE_VARIANT mapping all five supported theme IDs to either
// 'light' or 'dark', providing the default tile variant for each theme.
func TestStaticJS_MapThemeToTileVariant(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "MAP_THEME_TO_TILE_VARIANT") {
		t.Fatal("app.js: MAP_THEME_TO_TILE_VARIANT constant not found")
	}
	for _, themeID := range []string{"'default'", "'light-green'", "'deep-blue'", "'forest-green'", "'dark'"} {
		if !strings.Contains(body, themeID) {
			t.Errorf("app.js: MAP_THEME_TO_TILE_VARIANT must include theme %s", themeID)
		}
	}
}

// TestStaticJS_MapTileVariants verifies that MAP_TILE_VARIANTS is declared in
// app.js as an ordered array containing all three supported tile variants.
func TestStaticJS_MapTileVariants(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "MAP_TILE_VARIANTS") {
		t.Fatal("app.js: MAP_TILE_VARIANTS constant not found")
	}
	// All three variants must be listed.
	for _, v := range []string{"'light'", "'osm'", "'dark'"} {
		if !strings.Contains(body, v) {
			t.Errorf("app.js: MAP_TILE_VARIANTS must contain variant %s", v)
		}
	}
}

// TestStaticJS_OsmTileInConfigs verifies that the osm tile variant entry in
// TILE_LAYER_CONFIGS points to tile.openstreetmap.org.
func TestStaticJS_OsmTileInConfigs(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// 'osm' key must exist as a variant key.
	if !strings.Contains(body, "osm:") && !strings.Contains(body, "'osm'") {
		t.Error("app.js: TILE_LAYER_CONFIGS must declare an osm variant")
	}
	if !strings.Contains(body, "tile.openstreetmap.org") {
		t.Error("app.js: osm variant must use tile.openstreetmap.org URL")
	}
}

// TestStaticJS_SetMapTileVariant verifies that app.js exposes a named
// setMapTileVariant() function which is called from the map bar buttons.
func TestStaticJS_SetMapTileVariant(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function setMapTileVariant(") {
		t.Error("app.js: setMapTileVariant function not found")
	}
}

// TestStaticJS_RenderMapBar verifies that app.js exposes a named renderMapBar()
// function that builds the three tile-variant buttons above the map.
func TestStaticJS_RenderMapBar(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function renderMapBar(") {
		t.Error("app.js: renderMapBar function not found")
	}
}

// TestStaticJS_SyncMapTileVariantToTheme verifies that app.js exposes a named
// syncMapTileVariantToTheme() function which is called by applyTheme() and
// initTheme() to align the tile variant with the active colour theme.
func TestStaticJS_SyncMapTileVariantToTheme(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "function syncMapTileVariantToTheme(") {
		t.Error("app.js: syncMapTileVariantToTheme function not found")
	}
}

// TestStaticJS_ThemeTransitioning verifies that applyTheme() adds the
// 'theme-transitioning' class to body to drive the fade-out/in animation.
func TestStaticJS_ThemeTransitioning(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "theme-transitioning") {
		t.Error("app.js: 'theme-transitioning' class not found — theme fade animation requires it")
	}
	// The class must be both added and removed within applyTheme.
	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("app.js: applyTheme function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	if !strings.Contains(fnBody, "theme-transitioning") {
		t.Error("app.js: applyTheme must reference 'theme-transitioning' class")
	}
}

// TestStaticCSS_ThemeTransitioning verifies that style.css defines the
// body.theme-transitioning rule which snaps opacity to 0 for the theme fade.
func TestStaticCSS_ThemeTransitioning(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "body.theme-transitioning") {
		t.Error("style.css: body.theme-transitioning rule not found")
	}
	if !strings.Contains(body, "--theme-fade-dur") {
		t.Error("style.css: --theme-fade-dur CSS custom property not found")
	}
}

// TestStaticCSS_GeoMapFading verifies that style.css defines the
// #geo-map.geo-map--fading rule used during tile-swap fade animation.
func TestStaticCSS_GeoMapFading(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "geo-map--fading") {
		t.Error("style.css: geo-map--fading modifier class not found")
	}
	if !strings.Contains(body, "--map-fade-dur") {
		t.Error("style.css: --map-fade-dur CSS custom property not found")
	}
}

// TestStaticCSS_MapTileBar verifies that style.css declares the .geo-map-bar
// and .map-tile-btn rules required for the tile-variant dot selector bar.
func TestStaticCSS_MapTileBar(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, selector := range []string{".geo-map-bar", ".map-tile-btn", ".map-tile-btn.active"} {
		if !strings.Contains(body, selector) {
			t.Errorf("style.css: selector %q not found — map tile bar requires it", selector)
		}
	}
}

// TestStaticHTML_GeoMapBar verifies that index.html contains #geo-map-bar
// inside a .geo-map-outer wrapper element.
func TestStaticHTML_GeoMapBar(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `id="geo-map-bar"`) {
		t.Error(`index.html: element with id="geo-map-bar" not found`)
	}
}

// ---------------------------------------------------------------------------
// Phase 6 fix-2 tests — color-scheme / dot buttons / overlay wrapper
// ---------------------------------------------------------------------------

// TestStaticCSS_DarkThemeColorScheme verifies that all three dark themes
// declare color-scheme: dark so Chrome/Safari use dark-mode form-control
// rendering and do not revert focused-input text to the UA default (black).
func TestStaticCSS_DarkThemeColorScheme(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	for _, theme := range []string{"dark", "deep-blue", "forest-green"} {
		themeIdx := strings.Index(css, `[data-theme="`+theme+`"]`)
		if themeIdx == -1 {
			t.Errorf("style.css: [data-theme=%q] block not found", theme)
			continue
		}
		// Find the closing brace of the block (next '}' at column 0).
		blockEnd := strings.Index(css[themeIdx:], "\n}")
		if blockEnd == -1 {
			t.Errorf("style.css: [data-theme=%q] block closing brace not found", theme)
			continue
		}
		block := css[themeIdx : themeIdx+blockEnd]
		if !strings.Contains(block, "color-scheme: dark") {
			t.Errorf("style.css: [data-theme=%q] must declare `color-scheme: dark` to fix dark-theme input text color", theme)
		}
	}
}

// TestStaticCSS_RootColorSchemeLight verifies that :root declares
// color-scheme: light as the baseline so light themes' form controls default
// to light-mode UA rendering.
func TestStaticCSS_RootColorSchemeLight(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	rootStart := strings.Index(css, ":root {")
	if rootStart == -1 {
		t.Fatal("style.css: :root block not found")
	}
	rootEnd := strings.Index(css[rootStart:], "\n}")
	if rootEnd == -1 {
		t.Fatal("style.css: :root closing brace not found")
	}
	rootBlock := css[rootStart : rootStart+rootEnd]
	if !strings.Contains(rootBlock, "color-scheme: light") {
		t.Error("style.css: :root must declare `color-scheme: light` as the default for light themes")
	}
}

// TestStaticCSS_MapTileBarOverlay verifies that .geo-map-bar uses
// position: absolute so it overlays the map instead of sitting above it.
func TestStaticCSS_MapTileBarOverlay(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	barIdx := strings.Index(css, ".geo-map-bar {")
	if barIdx == -1 {
		t.Fatal("style.css: .geo-map-bar rule not found")
	}
	blockEnd := strings.Index(css[barIdx:], "\n}")
	if blockEnd == -1 {
		t.Fatal("style.css: .geo-map-bar closing brace not found")
	}
	block := css[barIdx : barIdx+blockEnd]
	if !strings.Contains(block, "position: absolute") {
		t.Error("style.css: .geo-map-bar must use position:absolute to overlay the map")
	}
}

// TestStaticCSS_GeoMapOuterRelative verifies that .geo-map-outer has
// position: relative, providing the positioning context for .geo-map-bar.
func TestStaticCSS_GeoMapOuterRelative(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	outerIdx := strings.Index(css, ".geo-map-outer {")
	if outerIdx == -1 {
		t.Fatal("style.css: .geo-map-outer rule not found")
	}
	blockEnd := strings.Index(css[outerIdx:], "\n}")
	if blockEnd == -1 {
		t.Fatal("style.css: .geo-map-outer closing brace not found")
	}
	block := css[outerIdx : outerIdx+blockEnd]
	if !strings.Contains(block, "position: relative") {
		t.Error("style.css: .geo-map-outer must have position:relative to contain absolute .geo-map-bar")
	}
}

// TestStaticCSS_MapTileBtnCircle verifies that .map-tile-btn is styled as a
// circle (border-radius: 50%) matching the .theme-btn visual language.
func TestStaticCSS_MapTileBtnCircle(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	btnIdx := strings.Index(css, ".map-tile-btn {")
	if btnIdx == -1 {
		t.Fatal("style.css: .map-tile-btn rule not found")
	}
	blockEnd := strings.Index(css[btnIdx:], "\n}")
	if blockEnd == -1 {
		t.Fatal("style.css: .map-tile-btn closing brace not found")
	}
	block := css[btnIdx : btnIdx+blockEnd]
	if !strings.Contains(block, "border-radius: 50%") {
		t.Error("style.css: .map-tile-btn must use border-radius:50% (circle) to match .theme-btn style")
	}
}

// TestStaticCSS_MapTileBtnVariantColors verifies that per-variant colour swatches
// are declared for all three tile variants (light, osm, dark).
func TestStaticCSS_MapTileBtnVariantColors(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	for _, variant := range []string{"light", "osm", "dark"} {
		selector := `.map-tile-btn[data-tile-variant="` + variant + `"]`
		if !strings.Contains(css, selector) {
			t.Errorf("style.css: per-variant swatch rule %q not found", selector)
		}
	}
}

// TestStaticHTML_GeoMapOuter verifies that index.html wraps #geo-map-bar and
// #geo-map in a .geo-map-outer element which provides the overlay context.
func TestStaticHTML_GeoMapOuter(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `id="geo-map-outer"`) {
		t.Error(`index.html: element with id="geo-map-outer" not found`)
	}
	if !strings.Contains(body, `class="geo-map-outer"`) {
		t.Error(`index.html: element with class="geo-map-outer" not found`)
	}
}

// TestStaticJS_RenderMapUsesOuterWrapper verifies that renderMap() references
// geo-map-outer to toggle the entire map area (wrapper + bar + map) as one unit.
func TestStaticJS_RenderMapUsesOuterWrapper(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("app.js: renderMap function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 3000
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "geo-map-outer") {
		t.Error("app.js: renderMap must reference geo-map-outer to toggle map area visibility")
	}
}

// TestStaticJS_RenderMapBarNoTextContent verifies that renderMapBar() produces
// buttons without text content — dot-only style, accessible via aria-label.
func TestStaticJS_RenderMapBarNoTextContent(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function renderMapBar(")
	if fnStart == -1 {
		t.Fatal("app.js: renderMapBar function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	// Must have aria-label for accessibility.
	if !strings.Contains(fnBody, "aria-label") {
		t.Error("app.js: renderMapBar buttons must include aria-label for accessibility")
	}
	// Must have title for native tooltip.
	if !strings.Contains(fnBody, "title=") {
		t.Error("app.js: renderMapBar buttons should include title attribute for tooltip")
	}
	// The button closing tag must immediately follow the opening tag (no text node).
	// Check that the inner text is NOT rendered (no i18nKey value as text content).
	if strings.Contains(fnBody, ">'"+"\n") || strings.Contains(fnBody, "> +\n      esc(t(") {
		t.Error("app.js: renderMapBar must not render i18n text inside the button element")
	}
}

// declare translation keys for all three tile variants: light, osm, and dark.
func TestStaticI18n_MapTileVariantKeys(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/i18n.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /i18n.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	for _, key := range []string{"'map-tile-light'", "'map-tile-osm'", "'map-tile-dark'"} {
		first := strings.Index(body, key)
		if first == -1 {
			t.Errorf("i18n.js: key %s not found in any locale", key)
			continue
		}
		// Key must appear at least twice (en + zh).
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh", key)
		}
	}
}

// Phase 7 fix tests — map z-index isolation / header+footer fade / copyright year
// ---------------------------------------------------------------------------

// TestStaticCSS_GeoMapIsolation verifies that #geo-map has isolation: isolate so
// that Leaflet's internal pane z-indices (200, 400…) are contained within the
// map's own stacking context and cannot bleed into .geo-map-outer, where the
// .geo-map-bar overlay sits at z-index: 10.  Without this, Leaflet's tile pane
// (z-index 200) would render above the dot-button overlay.
func TestStaticCSS_GeoMapIsolation(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	geoMapIdx := strings.Index(css, "#geo-map {")
	if geoMapIdx == -1 {
		t.Fatal("style.css: #geo-map rule not found")
	}
	endIdx := strings.Index(css[geoMapIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: #geo-map rule closing brace not found")
	}
	geoMapBlock := css[geoMapIdx : geoMapIdx+endIdx+1]
	if !strings.Contains(geoMapBlock, "isolation") {
		t.Error("style.css: #geo-map must have isolation: isolate to contain Leaflet's internal z-indices")
	}
	if !strings.Contains(geoMapBlock, "isolate") {
		t.Error("style.css: #geo-map isolation must be set to 'isolate'")
	}
}

// TestStaticCSS_HeaderHasColorTransition verifies that .site-header explicitly
// defines CSS transitions for background and color so the chrome strip smoothly
// cross-fades between theme palettes without ever disappearing (no opacity fade).
func TestStaticCSS_HeaderHasColorTransition(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	headerIdx := strings.Index(css, ".site-header  {")
	if headerIdx == -1 {
		t.Fatal("style.css: .site-header rule not found")
	}
	endIdx := strings.Index(css[headerIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: .site-header rule closing brace not found")
	}
	headerBlock := css[headerIdx : headerIdx+endIdx+1]
	if !strings.Contains(headerBlock, "transition") {
		t.Error("style.css: .site-header must have a transition property for smooth theme colour changes")
	}
	if !strings.Contains(headerBlock, "background") {
		t.Error("style.css: .site-header transition must include background")
	}
}

// TestStaticCSS_FooterHasColorTransition verifies that .site-footer explicitly
// defines CSS transitions for background and color, mirroring .site-header,
// so both chrome strips transition in visual unison on every theme change.
func TestStaticCSS_FooterHasColorTransition(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	footerIdx := strings.Index(css, ".site-footer  {")
	if footerIdx == -1 {
		t.Fatal("style.css: .site-footer rule not found")
	}
	endIdx := strings.Index(css[footerIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: .site-footer rule closing brace not found")
	}
	footerBlock := css[footerIdx : footerIdx+endIdx+1]
	if !strings.Contains(footerBlock, "transition") {
		t.Error("style.css: .site-footer must have a transition property for smooth theme colour changes")
	}
	if !strings.Contains(footerBlock, "background") {
		t.Error("style.css: .site-footer transition must include background")
	}
}

// TestStaticCSS_ThemeTransitioningMainOpacity verifies that
// body.theme-transitioning targets .main with opacity: 0 so only the main
// content area fades out during a theme switch (header and footer stay visible).
func TestStaticCSS_ThemeTransitioningMainOpacity(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// The rule that should appear is:  body.theme-transitioning .main { opacity: 0; }
	if !strings.Contains(css, "body.theme-transitioning .main") {
		t.Error("style.css: expected 'body.theme-transitioning .main' selector — only .main must fade, not the whole body")
	}
	ttIdx := strings.Index(css, "body.theme-transitioning .main")
	if ttIdx == -1 {
		return
	}
	endIdx := strings.Index(css[ttIdx:], "}")
	if endIdx != -1 {
		block := css[ttIdx : ttIdx+endIdx+1]
		if !strings.Contains(block, "opacity") {
			t.Error("style.css: body.theme-transitioning .main must set opacity (to 0) for the fade-out effect")
		}
	}
}

// TestStaticJS_ApplyThemeUsesMainElement verifies that applyTheme() attaches
// the transitionend listener to the .main element (not document.body), so the
// theme variables are swapped after only the main content has faded out and
// header/footer remain fully visible throughout.
func TestStaticJS_ApplyThemeUsesMainElement(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("app.js: applyTheme function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, ".main") && !strings.Contains(fnBody, "querySelector('.main')") {
		t.Error("app.js: applyTheme must use .main (querySelector('.main')) as the fade target, not body")
	}
	if !strings.Contains(fnBody, "addEventListener('transitionend'") && !strings.Contains(fnBody, `addEventListener("transitionend"`) {
		t.Error("app.js: applyTheme must attach a transitionend listener to the fade target")
	}
}

// TestStaticJS_CopyrightStartYearConst verifies that app.js declares a
// COPYRIGHT_START_YEAR constant so the copyright year range is driven from a
// single, readable source-of-truth rather than scattered literal values.
func TestStaticJS_CopyrightStartYearConst(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "COPYRIGHT_START_YEAR") {
		t.Error("app.js: COPYRIGHT_START_YEAR constant not found — copyright year logic requires a single source-of-truth")
	}
	// The constant must be assigned a four-digit year value.
	if !strings.Contains(body, "COPYRIGHT_START_YEAR = 2026") {
		t.Error("app.js: COPYRIGHT_START_YEAR must be initialised to 2026")
	}
}

// TestStaticJS_UpdateCopyrightYearFunction verifies that app.js defines an
// updateCopyrightYear() function that references the footer-copyright i18n key
// and builds an en-dash year range from COPYRIGHT_START_YEAR to the current year.
func TestStaticJS_UpdateCopyrightYearFunction(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function updateCopyrightYear(")
	if fnStart == -1 {
		t.Fatal("app.js: updateCopyrightYear function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "footer-copyright") {
		t.Error("app.js: updateCopyrightYear must target [data-i18n='footer-copyright'] elements")
	}
	if !strings.Contains(fnBody, "COPYRIGHT_START_YEAR") {
		t.Error("app.js: updateCopyrightYear must use COPYRIGHT_START_YEAR constant")
	}
	// En-dash (U+2013) separates the start and end years in the range string.
	if !strings.Contains(fnBody, `\u2013`) && !strings.Contains(fnBody, "–") {
		t.Error("app.js: updateCopyrightYear must use an en-dash (U+2013) to separate the year range")
	}
}

// TestStaticJS_ApplyLocaleCallsCopyrightYear verifies that applyLocale() calls
// updateCopyrightYear() so the copyright year is refreshed every time the
// locale is applied (including on page load and when the user switches language).
func TestStaticJS_ApplyLocaleCallsCopyrightYear(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("app.js: applyLocale function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1000
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "updateCopyrightYear") {
		t.Error("app.js: applyLocale must call updateCopyrightYear() to keep the copyright year range current")
	}
}

// Phase 7 (Round 2) tests — spellcheck suppression / map tile bg-color flash fix
// ---------------------------------------------------------------------------

// TestStaticJS_SpellcheckDisabledInDOMContentLoaded verifies that app.js
// centrally disables browser spell-check, autocorrect and autocapitalize on
// all input[type="text"] elements.  Doing this in the initialisation block
// (rather than per-element HTML attributes) ensures every current and future
// text field is covered without per-field opt-out.
func TestStaticJS_SpellcheckDisabledInDOMContentLoaded(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	if !strings.Contains(appJS, "spellcheck") {
		t.Error("app.js: must disable spellcheck on text inputs")
	}
	if !strings.Contains(appJS, "spellcheck = false") {
		t.Error("app.js: spellcheck must be set to false (el.spellcheck = false)")
	}
	if !strings.Contains(appJS, "autocorrect") {
		t.Error("app.js: must set autocorrect='off' on text inputs")
	}
	if !strings.Contains(appJS, "autocapitalize") {
		t.Error("app.js: must set autocapitalize='none' on text inputs")
	}
	// Must target input[type="text"] specifically.
	if !strings.Contains(appJS, `input[type="text"]`) {
		t.Error(`app.js: spellcheck suppression must target input[type="text"] elements`)
	}
}

// TestStaticJS_TileLayerConfigsBgColor verifies that every entry in
// TILE_LAYER_CONFIGS declares a bgColor property.  bgColor is the single
// source of truth for the map container background colour; without it the
// white-flash artefact cannot be fixed without hardcoding values elsewhere.
func TestStaticJS_TileLayerConfigsBgColor(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	cfgStart := strings.Index(appJS, "const TILE_LAYER_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("app.js: TILE_LAYER_CONFIGS not found")
	}
	// Extract to the closing brace of the object.
	endIdx := strings.Index(appJS[cfgStart:], "\n};")
	var cfgBlock string
	if endIdx != -1 {
		cfgBlock = appJS[cfgStart : cfgStart+endIdx+3]
	} else {
		cfgBlock = appJS[cfgStart : cfgStart+1500]
	}

	if !strings.Contains(cfgBlock, "bgColor") {
		t.Error("app.js: TILE_LAYER_CONFIGS must include a bgColor property on each entry")
	}
	// All three variants must carry the property.
	for _, variant := range []string{"light", "osm", "dark"} {
		vStart := strings.Index(cfgBlock, variant+":")
		if vStart == -1 {
			t.Errorf("app.js: TILE_LAYER_CONFIGS.%s entry not found", variant)
			continue
		}
		vEnd := strings.Index(cfgBlock[vStart:], "},")
		if vEnd == -1 {
			vEnd = len(cfgBlock) - vStart
		}
		vBlock := cfgBlock[vStart : vStart+vEnd]
		if !strings.Contains(vBlock, "bgColor") {
			t.Errorf("app.js: TILE_LAYER_CONFIGS.%s must have a bgColor property", variant)
		}
	}
}

// TestStaticJS_ApplyMapBgColorFunction verifies that app.js defines an
// applyMapBgColor() function that reads bgColor from TILE_LAYER_CONFIGS and
// applies it to the map container, acting as the single point responsible for
// the background-colour update.
func TestStaticJS_ApplyMapBgColorFunction(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function applyMapBgColor(")
	if fnStart == -1 {
		t.Fatal("app.js: applyMapBgColor function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 400
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "bgColor") {
		t.Error("app.js: applyMapBgColor must read bgColor from TILE_LAYER_CONFIGS")
	}
	if !strings.Contains(fnBody, "background") {
		t.Error("app.js: applyMapBgColor must set container.style.background")
	}
}

// TestStaticJS_RefreshMapTilesCallsApplyMapBgColor verifies that the animated
// tile swap path in refreshMapTiles() calls applyMapBgColor() after the new
// tile layer is added, so the container background is correct before the map
// fades back in.
func TestStaticJS_RefreshMapTilesCallsApplyMapBgColor(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function refreshMapTiles(")
	if fnStart == -1 {
		t.Fatal("app.js: refreshMapTiles function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1000
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "applyMapBgColor") {
		t.Error("app.js: refreshMapTiles must call applyMapBgColor() after swapping tiles to prevent white-flash on dark tile load")
	}
}
