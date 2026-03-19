package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticJS_DefaultThemeConstant verifies that theme.js reads the
// server-declared default theme from the HTML data-default-theme attribute
// and validates it against the THEMES list before applying.
func TestStaticJS_DefaultThemeConstant(t *testing.T) {
	h := newStaticHandler(t)

	// theme.js owns initTheme(); verify it reads the HTML attribute for the
	// server-declared default and validates against THEMES.
	themeRec := httptest.NewRecorder()
	h.ServeHTTP(themeRec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if themeRec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", themeRec.Code)
	}
	themeBody := themeRec.Body.String()

	if !strings.Contains(themeBody, "dataset.defaultTheme") {
		t.Error("theme.js: initTheme() must read document.documentElement.dataset.defaultTheme")
	}
	if !strings.Contains(themeBody, "themes.includes(htmlDefault)") {
		t.Error("theme.js: initTheme() must validate htmlDefault against the themes list before use")
	}
}

// TestStaticJS_ApplyThemeCallsRefreshMapTiles verifies that applyTheme() ensures
// map tiles are refreshed when the colour theme changes.  The function may do
// this directly (refreshMapTiles()) or via syncMapTileVariantToTheme(), which
// itself calls refreshMapTiles() internally.
func TestStaticJS_ApplyThemeCallsRefreshMapTiles(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")

	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	// Find the closing brace of applyTheme by scanning for the next top-level function.
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// applyTheme must trigger a tile refresh either directly or via syncMapTileVariantToTheme.
	if !strings.Contains(fnBody, "refreshMapTiles()") && !strings.Contains(fnBody, "syncMapTileVariantToTheme(") {
		t.Error("theme.js: applyTheme must call refreshMapTiles() or syncMapTileVariantToTheme() so tile layer updates on theme change")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 fix tests — theme fade / input colour / map-bar visibility / tile swap
// ---------------------------------------------------------------------------

// TestStaticJS_ApplyThemeFiltersOpacityEvent verifies that applyTheme() uses
// e.propertyName to guard the transitionend handler so only the body's own
// opacity transition — not background/color transitions or bubbling child
// events — triggers the theme swap.
func TestStaticJS_ApplyThemeFiltersOpacityEvent(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")

	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
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
		t.Error("theme.js: applyTheme transitionend handler must check e.propertyName to filter the correct transition event")
	}
	if !strings.Contains(fnBody, "'opacity'") {
		t.Error("theme.js: applyTheme must guard transitionend with e.propertyName === 'opacity'")
	}
}

// TestStaticJS_ThemeTransitioning verifies that applyTheme() adds the
// 'theme-transitioning' class to body to drive the fade-out/in animation.
func TestStaticJS_ThemeTransitioning(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")

	if !strings.Contains(body, "theme-transitioning") {
		t.Error("theme.js: 'theme-transitioning' class not found — theme fade animation requires it")
	}
	// The class must be both added and removed within applyTheme.
	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	if !strings.Contains(fnBody, "theme-transitioning") {
		t.Error("theme.js: applyTheme must reference 'theme-transitioning' class")
	}
}

// Phase 7 fix tests — map z-index isolation / header+footer fade / copyright year
// ---------------------------------------------------------------------------

// TestStaticJS_ApplyThemeUsesMainElement verifies that applyTheme() attaches
// the transitionend listener to the .main element (not document.body), so the
// theme variables are swapped after only the main content has faded out and
// header/footer remain fully visible throughout.
func TestStaticJS_ApplyThemeUsesMainElement(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", rec.Code)
	}
	themeJS := rec.Body.String()

	fnStart := strings.Index(themeJS, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	nextFn := strings.Index(themeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = themeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(themeJS) {
			end = len(themeJS)
		}
		fnBody = themeJS[fnStart:end]
	}

	if !strings.Contains(fnBody, ".main") && !strings.Contains(fnBody, "querySelector('.main')") {
		t.Error("theme.js: applyTheme must use .main (querySelector('.main')) as the fade target, not body")
	}
	if !strings.Contains(fnBody, "addEventListener('transitionend'") && !strings.Contains(fnBody, `addEventListener("transitionend"`) {
		t.Error("theme.js: applyTheme must attach a transitionend listener to the fade target")
	}
}

// ---------------------------------------------------------------------------
// Subtask 3.3 — theme.js module registration tests
// ---------------------------------------------------------------------------

// TestStaticJS_SetThemeGlobal verifies that theme.js exposes setTheme as a
// window-level global so that HTML onclick="setTheme(...)" attributes work
// without requiring callers to reference the PathProbe namespace directly.
func TestStaticJS_SetThemeGlobal(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "window.setTheme = setTheme") {
		t.Error("theme.js: must assign window.setTheme = setTheme for HTML onclick compatibility")
	}
}

// TestStaticJS_ThemeJSRuntimeResolvedMapSync verifies that theme.js guards
// the syncMapTileVariantToTheme call with a runtime check for PathProbe.Map
// so theme.js has no hard load-order dependency on the map module.
func TestStaticJS_ThemeJSRuntimeResolvedMapSync(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")
	if !strings.Contains(body, "PathProbe.Map") {
		t.Error("theme.js: syncMapTileVariantToTheme call must be guarded by a PathProbe.Map runtime check")
	}
	if !strings.Contains(body, "syncMapTileVariantToTheme") {
		t.Error("theme.js: must call syncMapTileVariantToTheme to keep map tiles in sync with the active theme")
	}
}

// TestStaticJS_InitThemeReadsDataDefaultTheme verifies that initTheme() reads
// the server-declared fallback from dataset.defaultTheme rather than repeating
// a hard-coded string, so a server-side theme preference takes effect without
// modifying any JavaScript source.
func TestStaticJS_InitThemeReadsDataDefaultTheme(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")
	fnStart := strings.Index(body, "function initTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: initTheme function not found")
	}
	end := fnStart + 600
	if end > len(body) {
		end = len(body)
	}
	if !strings.Contains(body[fnStart:end], "dataset.defaultTheme") {
		t.Error("theme.js: initTheme() must read document.documentElement.dataset.defaultTheme as the server-declared default")
	}
}
