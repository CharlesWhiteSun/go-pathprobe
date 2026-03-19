package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


// ── Phase 5: traceroute result rendering assertions ───────────────────────

// TestStaticJS_RenderRouteSection verifies that app.js defines a
// renderRouteSection function and wires it into renderReport so route hops
// are shown in the results pane when a traceroute diagnostic is returned.
func TestStaticJS_RenderRouteSection(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	// The render function must be defined.
	if !strings.Contains(body, "renderRouteSection") {
		t.Error("renderer.js: renderRouteSection function must be defined")
	}
	// It must be invoked from renderReport with the Route field.
	if !strings.Contains(body, "renderRouteSection(r.Route)") {
		t.Error("renderer.js: renderReport must call renderRouteSection(r.Route)")
	}
	// The route section heading i18n key must be referenced.
	if !strings.Contains(body, "'section-route'") {
		t.Error("renderer.js: renderRouteSection must reference i18n key 'section-route'")
	}
	// Timed-out hop indicator must be present.
	if !strings.Contains(body, "hop-timedout") {
		t.Error("renderer.js: renderRouteSection must apply 'hop-timedout' class to timed-out hops")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 6) tests — results section i18n re-render on locale switch
// ---------------------------------------------------------------------------

// TestStaticJS_LastReportStateVar verifies that app.js declares a module-level
// _lastReport variable used to cache the most recently rendered diagnostic
// report for re-rendering when the user switches locale.
func TestStaticJS_LastReportStateVar(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/renderer.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /renderer.js: want 200, got %d", rec.Code)
	}
	rendererJS := rec.Body.String()

	if !strings.Contains(rendererJS, "let _lastReport = null") {
		t.Error("renderer.js: module-level variable '_lastReport' not found — required to cache the report for locale-switch re-render")
	}
}

// TestStaticJS_RenderReportStoresLastReport verifies that renderReport() saves
// the report object into _lastReport so applyLocale() can re-render it later.
func TestStaticJS_RenderReportStoresLastReport(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/renderer.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /renderer.js: want 200, got %d", rec.Code)
	}
	rendererJS := rec.Body.String()

	fnStart := strings.Index(rendererJS, "function renderReport(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderReport function not found")
	}
	nextFn := strings.Index(rendererJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = rendererJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 500
		if end > len(rendererJS) {
			end = len(rendererJS)
		}
		fnBody = rendererJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "_lastReport = r") {
		t.Error("renderer.js: renderReport must assign '_lastReport = r' so the report can be replayed when the locale changes")
	}
}

// TestStaticJS_RenderRouteSectionColumns verifies that renderer.js references
// all six i18n column-header keys used in the route-trace hop table.
func TestStaticJS_RenderRouteSectionColumns(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	keys := []string{"th-ttl", "th-ip-host", "th-asn", "th-country", "th-loss", "th-avg-rtt"}
	for _, k := range keys {
		if !strings.Contains(body, "'"+k+"'") {
			t.Errorf("renderer.js: renderRouteSection must reference i18n key %q", k)
		}
	}
}
