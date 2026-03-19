package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


// ── Web mode radio-button tests ───────────────────────────────────────────

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

// ── animation & error-message tests ──────────────────────────────────────

// TestStaticJS_DotsRunAnimation verifies that app.js always injects the
// dots animation markup into #run-btn and that the picker system has been
// removed in favour of the fixed dots choice.
func TestStaticJS_DotsRunAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

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
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Both the catch block and the SSE error handler must clear innerHTML.
	// We verify by counting occurrences of the clear pattern.
	clearPattern := "progressEl.innerHTML = ''"
	count := strings.Count(body, clearPattern)
	if count < 2 {
		t.Errorf("app.js: progressEl.innerHTML='' must appear in both the catch block and the SSE error handler; found %d occurrence(s)", count)
	}
}

// TestStaticJS_LocalizeError verifies that app.js defines the localizeError
// function to map raw server error strings to user-friendly i18n messages,
// replacing opaque Go internal strings like "context deadline exceeded".
func TestStaticJS_LocalizeError(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

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

// TestStaticJS_SSEResultRevealOrder verifies that in the SSE 'result' event
// handler, resultEl.hidden = false is set BEFORE renderMap() is called.
// Leaflet initialises by reading the container's layout dimensions; if the
// parent #results section is still hidden (display:none) at that point, the
// map gets a 0×0 size and tiles are blank.
func TestStaticJS_SSEResultRevealOrder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

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
	body := fetchBody(t, newStaticHandler(t), "/app.js")

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

// TestStaticJS_AppendProgressNoInnerHTML verifies that appendProgress in
// app.js builds its DOM nodes with textContent (not innerHTML) so that
// untrusted progress-event strings cannot inject HTML/JS (XSS protection).
func TestStaticJS_AppendProgressNoInnerHTML(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Locate the function body.
	fnStart := strings.Index(body, "function appendProgress(")
	if fnStart == -1 {
		t.Fatal("app.js: appendProgress function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 600
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if strings.Contains(fnBody, "innerHTML") {
		t.Error("app.js: appendProgress must not use innerHTML — use textContent for XSS safety")
	}
	if !strings.Contains(fnBody, "textContent") {
		t.Error("app.js: appendProgress must use textContent to set stage/message text")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 8) tests — locale-aware history timestamps
// ---------------------------------------------------------------------------

// TestStaticJS_LastHistoryItemsStateVar verifies that app.js declares a
// module-level _lastHistoryItems variable used to cache the fetched history
// items for re-rendering when the user switches locale.
func TestStaticJS_LastHistoryItemsStateVar(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "let _lastHistoryItems = null") {
		t.Error("app.js: module-level variable '_lastHistoryItems' not found — required to cache history list for locale-switch re-render")
	}
}

// TestStaticJS_FormatHistoryTimeFunction verifies that app.js defines a
// formatHistoryTime() function and that it reads the active locale from
// PathProbe.Locale.getLocale() so timestamps reflect the active language.
func TestStaticJS_FormatHistoryTimeFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function formatHistoryTime(")
	if fnStart == -1 {
		t.Fatal("app.js: formatHistoryTime function not found")
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

	// Must delegate locale lookup to PathProbe.Locale.getLocale() since _locale
	// is now encapsulated inside locale.js.
	if !strings.Contains(fnBody, "getLocale()") {
		t.Error("app.js: formatHistoryTime must obtain the active locale via PathProbe.Locale.getLocale() — _locale is private to locale.js")
	}
	if !strings.Contains(fnBody, "toLocaleString(") {
		t.Error("app.js: formatHistoryTime must call toLocaleString() to format timestamps using the active locale")
	}
}

// TestStaticJS_RenderHistoryListCachesItems verifies that renderHistoryList()
// assigns items to _lastHistoryItems so applyLocale() can re-render the list
// when the user switches language.
func TestStaticJS_RenderHistoryListCachesItems(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function renderHistoryList(")
	if fnStart == -1 {
		t.Fatal("app.js: renderHistoryList function not found")
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

	if !strings.Contains(fnBody, "_lastHistoryItems = items") {
		t.Error("app.js: renderHistoryList must assign '_lastHistoryItems = items' so the list can be replayed on locale change")
	}
	if !strings.Contains(fnBody, "formatHistoryTime(") {
		t.Error("app.js: renderHistoryList must call formatHistoryTime() to produce locale-aware timestamps")
	}
}

// TestStaticJS_AppConfigDefensiveAccess verifies that app.js accesses
// PathProbe.Config through the explicit window.PathProbe property rather than
// a bare PathProbe identifier.
//
// Background: bare identifier lookup in a classic browser script throws
// ReferenceError when window.PathProbe was never set (e.g. when the browser
// cache serves an old index.html that lacks the config.js <script> tag).
// window.PathProbe property access safely returns undefined instead of
// throwing, preventing a catastrophic script failure that would leave
// setTheme() and setLocale() uncallable from inline onclick attributes.
func TestStaticJS_AppConfigDefensiveAccess(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Must access config through window.PathProbe, not via a bare PathProbe
	// identifier that throws ReferenceError when config.js did not run.
	if !strings.Contains(body, "window.PathProbe") {
		t.Error("app.js: must access config through window.PathProbe to avoid " +
			"ReferenceError when config.js fails to execute")
	}

	// Must use a defensive fallback (|| {} or ?? {}) so the destructuring
	// never throws even when window.PathProbe.Config is unavailable.
	if !strings.Contains(body, "|| {}") && !strings.Contains(body, "?? {}") {
		t.Error("app.js: config alias block must use a defensive fallback (|| {} or ?? {}) " +
			"to prevent crashing when config.js is unavailable")
	}

	// THEMES must have an explicit fallback default inside the destructuring
	// so that applyTheme() / initTheme() can safely call THEMES.includes()
	// even when config.js failed to load.
	if !strings.Contains(body, "THEMES") || !strings.Contains(body, "'default'") {
		t.Error("app.js: THEMES must carry a fallback default value in the config alias block")
	}

	// setTheme and setLocale must be defined in app.js as top-level function
	// declarations so they are accessible from inline onclick attributes.
	for _, fn := range []string{"function setTheme(", "function setLocale("} {
		if !strings.Contains(body, fn) {
			t.Errorf("app.js: %q must be a top-level function declaration", fn)
		}
	}
}
