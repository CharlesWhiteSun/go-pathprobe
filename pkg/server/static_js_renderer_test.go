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
func TestStaticJS_RenderDNSSectionFiveStateBadge(t *testing.T) {
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
	noneFoundIdx := strings.Index(body, "entry.NoneFound")
	allEmptyIdx := strings.Index(body, "entry.AllEmpty")
	if allFailedIdx == -1 {
		t.Fatal("renderer.js: renderDNSSection must check entry.AllFailed for the all-failed badge")
	}
	if hasDivIdx == -1 {
		t.Fatal("renderer.js: renderDNSSection must check entry.HasDivergence")
	}
	if noneFoundIdx == -1 {
		t.Fatal("renderer.js: renderDNSSection must check entry.NoneFound for the no-records badge (fifth state)")
	}
	if allEmptyIdx == -1 {
		t.Error("renderer.js: renderDNSSection must check entry.AllEmpty (AllEmpty ⊆ NoneFound, kept for safety)")
	}
	// Priority order: AllFailed → HasDivergence → NoneFound → AllEmpty → consistent.
	if allFailedIdx > hasDivIdx {
		t.Error("renderer.js: entry.AllFailed must be checked BEFORE entry.HasDivergence")
	}
	if hasDivIdx > noneFoundIdx {
		t.Error("renderer.js: entry.HasDivergence must be checked BEFORE entry.NoneFound")
	}
	if noneFoundIdx > allEmptyIdx {
		t.Error("renderer.js: entry.NoneFound must be checked BEFORE entry.AllEmpty")
	}
	// NoneFound and AllEmpty → badge-warn.
	if !strings.Contains(body, "badge-warn") {
		t.Error("renderer.js: renderDNSSection must use badge-warn class for NoneFound/AllEmpty entries")
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

// TestStaticJS_DNSAnswerCategoryDisplay verifies that renderer.js uses the
// Go-computed ans.ErrorCategory field to choose a user-facing category label,
// keeping raw Go error strings out of the visible UI and in the tooltip only.
// Classification logic lives in Go (ClassifyDNSLookupError); the renderer
// performs a pure lookup-table mapping from category string to i18n key.
func TestStaticJS_DNSAnswerCategoryDisplay(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	// The old JS-side pattern-matching helper must no longer exist.
	if strings.Contains(body, "function friendlyDNSError(") {
		t.Error("renderer.js: friendlyDNSError() must be removed — classification moved to Go side")
	}
	// renderDNSSection must reference ans.ErrorCategory (Go-computed field).
	if !strings.Contains(body, "ans.ErrorCategory") {
		t.Error("renderer.js: renderDNSSection must use ans.ErrorCategory to choose the error label")
	}
	// Must map all four actionable categories to i18n keys.
	for _, key := range []string{
		"'dns-cat-input'", "'dns-cat-nxdomain'", "'dns-cat-network'",
		"'dns-cat-resolver'", "'dns-cat-unknown'",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("renderer.js: must reference i18n key %s for DNS error category display", key)
		}
	}
	// Raw LookupError must appear in a title= tooltip, not as visible text.
	fnStart := strings.Index(body, "function renderDNSSection(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderDNSSection not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 4000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	if !strings.Contains(fnBody, "title=") {
		t.Error("renderer.js: raw LookupError must be placed in a title= attribute (tooltip), not rendered as visible text")
	}
	// Hint row must be rendered using entry.HintKey.
	if !strings.Contains(fnBody, "entry.HintKey") {
		t.Error("renderer.js: renderDNSSection must render a hint banner using entry.HintKey")
	}
	// Hint is now a <div class="dns-hint"> inside the card, independent of the
	// resolver table's column structure — not a table row with colspan.
	if !strings.Contains(fnBody, "dns-hint") {
		t.Error(`renderer.js: hint banner must use class "dns-hint" (div-based, not a table row)`)
	}
	if strings.Contains(fnBody, "dns-hint-row") {
		t.Error("renderer.js: old tr.dns-hint-row must be removed — hint is now a <div class=\"dns-hint\">")
	}

	// Error category badge (dns-err-label) must be attached to the Resolver
	// column cell (resolverCell), NOT to the Records cell (recordsCell).
	// The Records column is for actual DNS record values only.
	errLabelIdx := strings.Index(fnBody, "dns-err-label")
	resolverCellIdx := strings.Index(fnBody, "resolverCell")
	recordsCellIdx := strings.Index(fnBody, "recordsCell")
	if errLabelIdx == -1 {
		t.Fatal("renderer.js: dns-err-label badge class must be used (error badge in Resolver column)")
	}
	if resolverCellIdx == -1 || recordsCellIdx == -1 {
		t.Fatal("renderer.js: expected both resolverCell and recordsCell variables in renderDNSSection")
	}
	// dns-err-label must appear in the resolverCell block (before recordsCell).
	if errLabelIdx > recordsCellIdx {
		t.Error("renderer.js: dns-err-label must be set on resolverCell (before recordsCell), not in the Records column")
	}
	// The recordsCell block must not check ans.LookupError — Records column
	// shows actual records (ans.Values) or a dash; error status belongs in Resolver column.
	rttCellIdx := strings.Index(fnBody, "const rttCell")
	if rttCellIdx > recordsCellIdx {
		recordsCellBlock := fnBody[recordsCellIdx:rttCellIdx]
		if strings.Contains(recordsCellBlock, "dns-err-label") {
			t.Error("renderer.js: dns-err-label must not appear in the recordsCell block")
		}
	}
}

// TestStaticJS_DNSSectionCardLayout verifies that renderDNSSection uses the
// card-per-entry layout that cleanly separates group-level identity (domain,
// type, status) from resolver-level detail (resolver name, records, RTT).
//
// Key structural requirements:
//   - Outer container is dns-groups (div), not a flat table
//   - Each entry is a dns-group card with a dns-group-header
//   - Card header contains dns-group-domain and dns-group-type spans
//   - Inner resolver table uses class dns-answer-table (3 columns only)
//   - th-dns-domain and th-dns-type must NOT appear (solved column-repetition bug)
func TestStaticJS_DNSSectionCardLayout(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function renderDNSSection(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderDNSSection not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 6000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// Outer container must be a div.dns-groups, not a flat table.
	if !strings.Contains(fnBody, "dns-groups") {
		t.Error("renderer.js: renderDNSSection must use a dns-groups container div")
	}
	// Each entry must be a dns-group card.
	if !strings.Contains(fnBody, "dns-group") {
		t.Error("renderer.js: renderDNSSection must wrap each entry in a dns-group card")
	}
	// Card header must contain both dns-group-domain and dns-group-type.
	for _, cls := range []string{"dns-group-header", "dns-group-domain", "dns-group-type"} {
		if !strings.Contains(fnBody, cls) {
			t.Errorf("renderer.js: card header must use class %q", cls)
		}
	}
	// Domain and Type are shown directly in the card header — the old i18n
	// column-header keys must no longer appear in the renderer.
	for _, obsolete := range []string{"th-dns-domain", "th-dns-type"} {
		if strings.Contains(fnBody, "'"+obsolete+"'") {
			t.Errorf("renderer.js: obsolete column-header i18n key %q must be removed", obsolete)
		}
	}
	// Inner resolver table must use dns-answer-table (not result-table).
	if !strings.Contains(fnBody, "dns-answer-table") {
		t.Error("renderer.js: inner resolver table must use class dns-answer-table")
	}
	// The three column-header i18n keys that remain must still be present.
	for _, key := range []string{"'th-dns-resolver'", "'th-dns-records'", "'th-dns-rtt'"} {
		if !strings.Contains(fnBody, key) {
			t.Errorf("renderer.js: inner table must reference i18n key %s", key)
		}
	}
	// Old flat-table artifacts must be gone.
	for _, old := range []string{"dns-entry-row", "dns-table", "colspan=\"5\""} {
		if strings.Contains(fnBody, old) {
			t.Errorf("renderer.js: old flat-table artifact %q must be removed", old)
		}
	}
}
