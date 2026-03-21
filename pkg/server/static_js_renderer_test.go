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

// TestStaticJS_RenderDNSSectionFourStateBadge verifies that renderDNSSection
// in renderer.js implements the four-state badge logic:
//
//	AllFailed       → badge-fail  + dns-all-failed  (every resolver errored — highest priority)
//	HasDivergence   → badge-fail  + dns-divergent
//	AllEmpty        → badge-warn  + dns-no-records
//	consistent+data → badge-ok    + dns-consistent
//
// AllFailed must be checked before HasDivergence so that the case where all
// resolvers fail (Values all nil → HasDivergence=false) is correctly labelled
// rather than falling through as "Consistent".
func TestStaticJS_RenderDNSSectionFourStateBadge(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	// The function must exist.
	if !strings.Contains(body, "renderDNSSection") {
		t.Fatal("renderer.js: renderDNSSection function must be defined")
	}
	// All four i18n keys must be referenced.
	for _, key := range []string{"dns-all-failed", "dns-divergent", "dns-no-records", "dns-consistent"} {
		if !strings.Contains(body, "'"+key+"'") {
			t.Errorf("renderer.js: renderDNSSection must reference i18n key %q", key)
		}
	}
	// AllFailed must be checked first (entry.AllFailed before entry.HasDivergence).
	allFailedIdx := strings.Index(body, "entry.AllFailed")
	hasDivIdx := strings.Index(body, "entry.HasDivergence")
	if allFailedIdx == -1 {
		t.Fatal("renderer.js: renderDNSSection must check entry.AllFailed for the all-failed badge")
	}
	if hasDivIdx == -1 {
		t.Fatal("renderer.js: renderDNSSection must check entry.HasDivergence")
	}
	if allFailedIdx > hasDivIdx {
		t.Error("renderer.js: entry.AllFailed must be checked BEFORE entry.HasDivergence (priority order)")
	}
	// AllEmpty → badge-warn.
	if !strings.Contains(body, "entry.AllEmpty") {
		t.Error("renderer.js: renderDNSSection must check entry.AllEmpty for the no-records badge")
	}
	if !strings.Contains(body, "badge-warn") {
		t.Error("renderer.js: renderDNSSection must use badge-warn class for AllEmpty entries")
	}
	// Failure states → badge-fail.
	if !strings.Contains(body, "badge-fail") {
		t.Error("renderer.js: renderDNSSection must use badge-fail class for AllFailed/Divergent entries")
	}
	// Consistent → badge-ok.
	if !strings.Contains(body, "badge-ok") {
		t.Error("renderer.js: renderDNSSection must use badge-ok class for consistent entries")
	}
	// renderDNSSection must be wired into renderReport.
	if !strings.Contains(body, "renderDNSSection(r.DNS)") {
		t.Error("renderer.js: renderReport must call renderDNSSection(r.DNS)")
	}
}

// TestStaticJS_FriendlyDNSError verifies that renderer.js defines a
// friendlyDNSError() helper that converts raw Go resolver errors into
// user-readable labels, keeping raw error strings out of the UI.
func TestStaticJS_FriendlyDNSError(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	if !strings.Contains(body, "function friendlyDNSError(") {
		t.Fatal("renderer.js: friendlyDNSError function must be defined")
	}
	// Must reference the friendly-error i18n keys.
	for _, key := range []string{"dns-err-no-host", "dns-err-invalid-domain",
		"dns-err-resolver-failed", "dns-err-timeout", "dns-err-generic"} {
		if !strings.Contains(body, "'"+key+"'") {
			t.Errorf("renderer.js: friendlyDNSError must reference i18n key %q", key)
		}
	}
	// Must detect common raw Go error patterns.
	for _, pattern := range []string{"no such host", "invalid character",
		"resolver returned error", "deadline exceeded", "timed out"} {
		if !strings.Contains(body, "'"+pattern+"'") && !strings.Contains(body, "\""+pattern+"\"") {
			t.Errorf("renderer.js: friendlyDNSError must detect the pattern %q", pattern)
		}
	}
	// The error cell must use friendlyDNSError() when rendering LookupError.
	fnStart := strings.Index(body, "function renderDNSSection(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderDNSSection not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 3000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	if !strings.Contains(fnBody, "friendlyDNSError(") {
		t.Error("renderer.js: renderDNSSection must call friendlyDNSError() when rendering ans.LookupError")
	}
	// Raw LookupError must be in tooltip (title=), not as visible text.
	if !strings.Contains(fnBody, "title=") {
		t.Error("renderer.js: raw LookupError must be placed in a title= attribute (tooltip), not rendered as visible text")
	}
}

