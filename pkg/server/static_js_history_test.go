package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticHandler_ServesHistoryJS verifies that GET /history.js returns
// HTTP 200 with a JavaScript Content-Type and registers PathProbe.History.
func TestStaticHandler_ServesHistoryJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /history.js: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("GET /history.js Content-Type = %q, want javascript", ct)
	}
	if !strings.Contains(rec.Body.String(), "PathProbe.History") {
		t.Error("history.js: must register PathProbe.History namespace")
	}
}

// TestStaticJS_RenderHistoryListCachesItems verifies that renderHistoryList()
// assigns items to _lastHistoryItems so rerenderLast() can replay the list
// when the user switches language.
func TestStaticJS_RenderHistoryListCachesItems(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /history.js: want 200, got %d", rec.Code)
	}
	histJS := rec.Body.String()

	fnStart := strings.Index(histJS, "function renderHistoryList(")
	if fnStart == -1 {
		t.Fatal("history.js: renderHistoryList function not found")
	}
	nextFn := strings.Index(histJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = histJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(histJS) {
			end = len(histJS)
		}
		fnBody = histJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "_lastHistoryItems = items") {
		t.Error("history.js: renderHistoryList must assign '_lastHistoryItems = items' so the list can be replayed on locale change")
	}
	if !strings.Contains(fnBody, "formatHistoryTime(") {
		t.Error("history.js: renderHistoryList must call formatHistoryTime() to produce locale-aware timestamps")
	}
}

// TestStaticJS_FormatHistoryTimeFunction verifies that history.js defines a
// formatHistoryTime() function and that it reads the active locale from
// document.documentElement.lang rather than PathProbe.Locale.getLocale(),
// so history.js has no static dependency on locale.js's internal API.
func TestStaticJS_FormatHistoryTimeFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /history.js: want 200, got %d", rec.Code)
	}
	histJS := rec.Body.String()

	fnStart := strings.Index(histJS, "function formatHistoryTime(")
	if fnStart == -1 {
		t.Fatal("history.js: formatHistoryTime function not found")
	}
	nextFn := strings.Index(histJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = histJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 400
		if end > len(histJS) {
			end = len(histJS)
		}
		fnBody = histJS[fnStart:end]
	}

	// Must read locale from document.documentElement.lang — locale.js sets this
	// attribute on every applyLocale() call. Reading it directly avoids a hard
	// dependency on PathProbe.Locale's internal API.
	if !strings.Contains(fnBody, "document.documentElement.lang") {
		t.Error("history.js: formatHistoryTime must read locale from document.documentElement.lang — avoids dependency on PathProbe.Locale internals")
	}
	// Must NOT call getLocale() — that API is internal to locale.js.
	if strings.Contains(fnBody, "getLocale()") {
		t.Error("history.js: formatHistoryTime must not call getLocale() — use document.documentElement.lang instead")
	}
	if !strings.Contains(fnBody, "toLocaleString(") {
		t.Error("history.js: formatHistoryTime must call toLocaleString() to format timestamps using the active locale")
	}
}

// TestStaticJS_HistoryEntryRevealOrder verifies that in loadHistoryEntry(),
// resultEl.hidden = false is set BEFORE renderMap() so the Leaflet map
// container has non-zero layout dimensions when the library initialises.
func TestStaticJS_HistoryEntryRevealOrder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/history.js")

	fnStart := strings.Index(body, "function loadHistoryEntry(")
	if fnStart == -1 {
		t.Fatal("history.js: loadHistoryEntry function not found")
	}
	// Bound the search to the function body.
	nextFn := strings.Index(body[fnStart+1:], "\n  async function ")
	var fnBody string
	if nextFn != -1 && (fnStart+1+nextFn) <= len(body) {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1200
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	hiddenIdx := strings.Index(fnBody, "resultEl.hidden = false")
	renderMapIdx := strings.Index(fnBody, "renderMap(")
	if hiddenIdx == -1 {
		t.Fatal("history.js: resultEl.hidden = false not found in loadHistoryEntry")
	}
	if renderMapIdx == -1 {
		t.Fatal("history.js: renderMap( not found in loadHistoryEntry")
	}
	if hiddenIdx > renderMapIdx {
		t.Error("history.js: resultEl.hidden = false must appear BEFORE renderMap() in loadHistoryEntry — " +
			"#results must be visible so the Leaflet container has layout dimensions")
	}
}
