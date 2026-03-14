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
// before being hidden, and includes an animationend guard for rapid switching.
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
	// Rapid-switch guard: only hide if the panel is still in a leaving state.
	if !strings.Contains(body, "panel-leaving") || !strings.Contains(body, "classList.contains") {
		t.Error("app.js: animationend handler must guard with classList.contains('panel-leaving') to handle rapid target switching")
	}
	// Toggle functions must be absent — vivid mode is now the HTML-level default.
	for _, sym := range []string{"cycleAnim", "initAnim", "applyAnim", "ANIM_MODES"} {
		if strings.Contains(body, sym) {
			t.Errorf("app.js: %s must be removed; animation mode is now a static HTML attribute, not a runtime toggle", sym)
		}
	}
}
