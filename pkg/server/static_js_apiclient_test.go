package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticHandler_ServesApiClientJS verifies that GET /api-client.js returns
// HTTP 200 with a JavaScript Content-Type and registers PathProbe.ApiClient.
func TestStaticHandler_ServesApiClientJS(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api-client.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api-client.js: want 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("GET /api-client.js Content-Type = %q, want javascript", ct)
	}
	if !strings.Contains(rec.Body.String(), "PathProbe.ApiClient") {
		t.Error("api-client.js: must register PathProbe.ApiClient namespace")
	}
}

// TestStaticJS_RunDiagGlobal verifies that api-client.js exposes runDiag as a
// window-level global so that the #run-btn onclick="runDiag()" attribute works
// without requiring callers to reference the PathProbe namespace directly.
func TestStaticJS_RunDiagGlobal(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	if !strings.Contains(body, "window.runDiag = runDiag") {
		t.Error("api-client.js: must assign window.runDiag = runDiag for HTML onclick compatibility")
	}
}

// TestStaticJS_LocalizeError verifies that api-client.js defines the
// localizeError function with a three-way branching strategy:
//   - timeout / deadline exceeded → err-timeout i18n key
//   - no runner registered        → err-no-runner i18n key
//   - fallback                    → strip "diagnostic error: " prefix
func TestStaticJS_LocalizeError(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	if !strings.Contains(body, "localizeError") {
		t.Error("api-client.js: localizeError function must be defined")
	}
	// Branch 1: timeout / deadline exceeded.
	if !strings.Contains(body, "deadline exceeded") {
		t.Error("api-client.js: localizeError must handle 'deadline exceeded' error pattern")
	}
	// Branch 1 i18n key.
	if !strings.Contains(body, "err-timeout") {
		t.Error("api-client.js: localizeError must reference 'err-timeout' i18n key for timeout errors")
	}
	// Branch 2: no runner.
	if !strings.Contains(body, "no runner registered") {
		t.Error("api-client.js: localizeError must handle 'no runner registered' error pattern")
	}
	// Branch 2 i18n key.
	if !strings.Contains(body, "err-no-runner") {
		t.Error("api-client.js: localizeError must reference 'err-no-runner' i18n key")
	}
	// Branch 3: strip prefix.
	if !strings.Contains(body, "diagnostic error:") {
		t.Error("api-client.js: localizeError fallback must strip the 'diagnostic error:' prefix")
	}
}

// TestStaticJS_SSEResultRevealOrder verifies that in the SSE 'result' event
// handler inside handleSSEMessage, resultEl.hidden = false is set BEFORE
// renderMap() is called.  Leaflet initialises by reading the container's
// layout dimensions; if #results is still hidden (display:none) at that
// point the map gets a 0×0 size and tiles are blank.
func TestStaticJS_SSEResultRevealOrder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	// Locate handleSSEMessage and the evtName==='result' branch within it.
	fnStart := strings.Index(body, "function handleSSEMessage(")
	if fnStart == -1 {
		t.Fatal("api-client.js: handleSSEMessage function not found")
	}
	resultBranchIdx := strings.Index(body[fnStart:], "evtName === 'result'")
	if resultBranchIdx == -1 {
		t.Fatal("api-client.js: evtName === 'result' branch not found in handleSSEMessage")
	}
	// Inspect a window large enough to cover the result branch body.
	windowStart := fnStart + resultBranchIdx
	end := windowStart + 600
	if end > len(body) {
		end = len(body)
	}
	window := body[windowStart:end]

	hiddenIdx := strings.Index(window, "resultEl.hidden = false")
	renderMapIdx := strings.Index(window, "renderMap(")
	if hiddenIdx == -1 {
		t.Fatal("api-client.js: resultEl.hidden = false not found in SSE result branch")
	}
	if renderMapIdx == -1 {
		t.Fatal("api-client.js: renderMap( not found in SSE result branch")
	}
	if hiddenIdx > renderMapIdx {
		t.Error("api-client.js: resultEl.hidden = false must appear BEFORE renderMap() in the SSE result handler — " +
			"#results must be visible so the Leaflet container has layout dimensions")
	}
}

// TestStaticJS_ErrorClearsProgressLog verifies that api-client.js clears and
// hides the progress log both on SSE error events and on network-level
// failures, so partial traceroute output does not remain visible below the
// error banner.
func TestStaticJS_ErrorClearsProgressLog(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	// Both the catch block and the SSE error handler must clear innerHTML.
	clearPattern := "progressEl.innerHTML = ''"
	count := strings.Count(body, clearPattern)
	if count < 2 {
		t.Errorf("api-client.js: progressEl.innerHTML='' must appear in both the catch block "+
			"and the SSE error handler; found %d occurrence(s)", count)
	}
}

// TestStaticJS_AppendProgressNoInnerHTML verifies that appendProgress in
// api-client.js builds its DOM nodes with textContent (not innerHTML) so that
// untrusted progress-event strings cannot inject HTML/JS (XSS protection).
func TestStaticJS_AppendProgressNoInnerHTML(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	fnStart := strings.Index(body, "function appendProgress(")
	if fnStart == -1 {
		t.Fatal("api-client.js: appendProgress function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
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
		t.Error("api-client.js: appendProgress must not use innerHTML — use textContent for XSS safety")
	}
	if !strings.Contains(fnBody, "textContent") {
		t.Error("api-client.js: appendProgress must use textContent to set stage/message text")
	}
}
