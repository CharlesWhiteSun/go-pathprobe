package server_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)


// TestStaticJS_UpdateCopyrightYearFunction verifies that locale.js defines an
// updateCopyrightYear() function that references the footer-copyright i18n key
// and builds an en-dash year range from COPYRIGHT_START_YEAR to the current year.
func TestStaticJS_UpdateCopyrightYearFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function updateCopyrightYear(")
	if fnStart == -1 {
		t.Fatal("locale.js: updateCopyrightYear function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "footer-copyright") {
		t.Error("locale.js: updateCopyrightYear must target [data-i18n='footer-copyright'] elements")
	}
	if !strings.Contains(fnBody, "COPYRIGHT_START_YEAR") {
		t.Error("locale.js: updateCopyrightYear must use COPYRIGHT_START_YEAR constant")
	}
	// En-dash (U+2013) separates the start and end years in the range string.
	if !strings.Contains(fnBody, `\u2013`) && !strings.Contains(fnBody, "–") {
		t.Error("locale.js: updateCopyrightYear must use an en-dash to separate the year range")
	}
}

// TestStaticJS_ApplyLocaleCallsCopyrightYear verifies that applyLocale() calls
// updateCopyrightYear() so the copyright year is refreshed every time the
// locale is applied (including on page load and when the user switches language).
func TestStaticJS_ApplyLocaleCallsCopyrightYear(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("locale.js: applyLocale function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "updateCopyrightYear") {
		t.Error("locale.js: applyLocale must call updateCopyrightYear() to keep the copyright year range current")
	}
}

// TestStaticJS_ApplyLocaleReRendersReport verifies that locale.js / applyLocale()
// triggers results-section re-render via the runtime-resolved
// PathProbe.Renderer.rerenderLast() callback when a cached report is present.
// This keeps all dynamically generated label text in sync with the active
// locale without requiring data-i18n attributes in the generated HTML.
func TestStaticJS_ApplyLocaleReRendersReport(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("locale.js: applyLocale function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	// Must call through the runtime-resolved callback, not directly.
	if !strings.Contains(fnBody, "PathProbe.Renderer") {
		t.Error("locale.js: applyLocale must trigger report re-render via PathProbe.Renderer.rerenderLast()")
	}
	if !strings.Contains(fnBody, "rerenderLast") {
		t.Error("locale.js: applyLocale must call rerenderLast() to re-render the results section on locale change")
	}
}

// TestStaticJS_ApplyLocaleReRendersHistory verifies that locale.js / applyLocale()
// triggers history-list re-render via the runtime-resolved
// PathProbe.History.rerenderLast() callback so locale-aware timestamps are
// updated immediately when the user switches language.
func TestStaticJS_ApplyLocaleReRendersHistory(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("locale.js: applyLocale function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	// Must call through the runtime-resolved callback, not directly.
	if !strings.Contains(fnBody, "PathProbe.History") {
		t.Error("locale.js: applyLocale must trigger history re-render via PathProbe.History.rerenderLast()")
	}
	if !strings.Contains(fnBody, "rerenderLast") {
		t.Error("locale.js: applyLocale must call rerenderLast() to re-render the history list on locale change")
	}
}

// ── Sub-task 3.2: locale.js tests ─────────────────────────────────────────

// TestStaticJS_SetLocaleGlobal verifies that locale.js exposes setLocale as a
// global (window.setLocale = setLocale) so that inline onclick attributes in
// index.html (e.g. onclick="setLocale('en')") can call it without requiring
// app.js to re-declare the function.
func TestStaticJS_SetLocaleGlobal(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/locale.js")

	// The explicit global assignment must be present so browsers can call
	// setLocale() from inline onclick attributes on language buttons.
	if !strings.Contains(body, "window.setLocale = setLocale") {
		t.Error("locale.js: must contain 'window.setLocale = setLocale' to expose " +
			"setLocale as a global callable from inline onclick attributes")
	}
}

// TestStaticJS_LocaleUsesConfigCopyrightYear verifies that locale.js reads the
// copyright start year from PathProbe.Config.COPYRIGHT_START_YEAR at runtime
// rather than hard-coding a numeric year literal.  Hard-coding the year would
// violate the single-source-of-truth principle: the year is already declared
// in config.js and must not be duplicated.
func TestStaticJS_LocaleUsesConfigCopyrightYear(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/locale.js")

	// locale.js must read the year from the config namespace, not define it.
	if !strings.Contains(body, "COPYRIGHT_START_YEAR") {
		t.Error("locale.js: must read COPYRIGHT_START_YEAR from PathProbe.Config, " +
			"not hard-code a numeric year value")
	}

	// There must be no standalone four-digit year literal in the file.
	// The regex matches a bare year number that is not part of a larger
	// identifier (e.g. "2026" as a standalone token).
	import_re := `\b20\d{2}\b`
	matched, _ := regexp.MatchString(import_re, body)
	if matched {
		t.Error("locale.js: must not contain a hard-coded year literal — " +
			"read COPYRIGHT_START_YEAR from PathProbe.Config instead")
	}
}

// TestStaticJS_LocaleRuntimeResolvedCrossModuleCalls verifies that locale.js
// triggers re-render of the results section and history list through
// runtime-resolved cross-module calls (PathProbe.Renderer.rerenderLast and
// PathProbe.History.rerenderLast) rather than calling renderReport() or
// renderHistoryList() directly.
//
// Direct calls would create a hard load-order dependency on app.js (or future
// renderer.js / history.js modules), making locale.js impossible to test in
// isolation and breaking the low-coupling principle.  Guard expressions
// (PathProbe.Renderer && …) ensure the calls degrade gracefully when the
// target module is not yet registered.
func TestStaticJS_LocaleRuntimeResolvedCrossModuleCalls(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/locale.js")

	// Guard for the Renderer module must be present.
	if !strings.Contains(body, "PathProbe.Renderer &&") {
		t.Error("locale.js: applyLocale() must guard PathProbe.Renderer with " +
			"'PathProbe.Renderer &&' before calling rerenderLast() so the call " +
			"degrades gracefully when renderer.js has not yet loaded")
	}

	// Guard for the History module must be present.
	if !strings.Contains(body, "PathProbe.History &&") {
		t.Error("locale.js: applyLocale() must guard PathProbe.History with " +
			"'PathProbe.History &&' before calling rerenderLast() so the call " +
			"degrades gracefully when history.js has not yet loaded")
	}

	// The re-render must go through rerenderLast(), not call renderReport()
	// or renderHistoryList() directly (which would create a hard dependency).
	if strings.Contains(body, "renderReport(") || strings.Contains(body, "renderHistoryList(") {
		t.Error("locale.js: must NOT call renderReport() or renderHistoryList() directly — " +
			"use PathProbe.Renderer.rerenderLast() and PathProbe.History.rerenderLast() " +
			"for runtime-resolved cross-module calls")
	}
}
