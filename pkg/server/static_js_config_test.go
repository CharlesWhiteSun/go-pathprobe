package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


// TestStaticJS_WebTargetPortDefaults verifies that TARGET_PORTS.web includes
// both port 80 (HTTP) and port 443 (HTTPS) as the auto-fill defaults shown
// when the user selects web target + port connectivity mode.
func TestStaticJS_WebTargetPortDefaults(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// TARGET_PORTS.web must include both HTTP (80) and HTTPS (443) defaults.
	if !strings.Contains(body, "web:  [80, 443]") {
		t.Error("config.js: TARGET_PORTS.web must be [80, 443] (HTTP + HTTPS defaults for port-connectivity mode)")
	}
}

// TestStaticJS_WEB_MODES_WITH_PORTS verifies that config.js declares the
// WEB_MODES_WITH_PORTS constant used to drive port-group visibility in a
// data-driven, non-hardcoded manner.
func TestStaticJS_WEB_MODES_WITH_PORTS(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "WEB_MODES_WITH_PORTS") {
		t.Error("config.js: WEB_MODES_WITH_PORTS constant must be declared")
	}
	// Port connectivity mode must be listed as requiring port selection.
	if !strings.Contains(body, "'port'") {
		t.Error("config.js: WEB_MODES_WITH_PORTS must include 'port' mode")
	}
}

// TestStaticJS_MapPointConfigs verifies that app.js declares MAP_POINT_CONFIGS
// with 'origin' and 'target' keys, forming a data-driven foundation for all
// map marker styling.  Callers derive visual behaviour from this object rather
// than hardcoding logic inside renderMap().
func TestStaticJS_MapPointConfigs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_POINT_CONFIGS") {
		t.Error("config.js: MAP_POINT_CONFIGS constant not found — map marker config must be data-driven")
	}
	if !strings.Contains(body, "'origin'") {
		t.Error("config.js: MAP_POINT_CONFIGS must include an 'origin' key for the public-IP marker")
	}
	if !strings.Contains(body, "'target'") {
		t.Error("config.js: MAP_POINT_CONFIGS must include a 'target' key for the destination marker")
	}
}

// TestStaticJS_TileLayerConfigs verifies that app.js declares TILE_LAYER_CONFIGS
// with both 'light' and 'dark' variants pointing to the CARTO basemap service.
// Tile URLs must not use the raw OSM URL so theme-aware switching works correctly.
func TestStaticJS_TileLayerConfigs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "TILE_LAYER_CONFIGS") {
		t.Error("config.js: TILE_LAYER_CONFIGS constant not found — tile URLs must be data-driven")
	}
	if !strings.Contains(body, "'light'") {
		t.Error("config.js: TILE_LAYER_CONFIGS must include a 'light' variant")
	}
	if !strings.Contains(body, "'dark'") {
		t.Error("config.js: TILE_LAYER_CONFIGS must include a 'dark' variant")
	}
	// CARTO attribution must be present to satisfy the tile provider's terms.
	if !strings.Contains(body, "carto.com/attributions") {
		t.Error("config.js: CARTO attribution URL must be present in TILE_LAYER_CONFIGS")
	}
	// OSM is now a supported variant inside TILE_LAYER_CONFIGS; its URL is data-driven
	// and must appear inside that config block, not hardcoded in renderMap.
	if !strings.Contains(body, "tile.openstreetmap.org") {
		t.Error("config.js: tile.openstreetmap.org URL must appear in TILE_LAYER_CONFIGS as the osm variant")
	}
}

// TestStaticJS_MapDarkThemes verifies that config.js declares MAP_DARK_THEMES as
// the authoritative set of theme IDs that map to the dark tile variant.
func TestStaticJS_MapDarkThemes(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_DARK_THEMES") {
		t.Error("config.js: MAP_DARK_THEMES constant not found — dark/light tile selection must be data-driven")
	}
	// The known dark themes must be listed.
	for _, id := range []string{"'dark'", "'deep-blue'", "'forest-green'"} {
		cfg := strings.Index(body, "MAP_DARK_THEMES")
		if cfg == -1 {
			break
		}
		// look for the id somewhere after MAP_DARK_THEMES declaration
		if !strings.Contains(body[cfg:cfg+200], id) {
			t.Errorf("config.js: MAP_DARK_THEMES must include theme id %s", id)
		}
	}
}

// ---------------------------------------------------------------------------
// Phase 6 — theme-fade / map-tile-bar tests
// ---------------------------------------------------------------------------

// TestStaticJS_MapThemeToTileVariant verifies that config.js declares
// MAP_THEME_TO_TILE_VARIANT mapping all five supported theme IDs to either
// 'light' or 'dark', providing the default tile variant for each theme.
func TestStaticJS_MapThemeToTileVariant(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_THEME_TO_TILE_VARIANT") {
		t.Fatal("config.js: MAP_THEME_TO_TILE_VARIANT constant not found")
	}
	for _, themeID := range []string{"'default'", "'light-green'", "'deep-blue'", "'forest-green'", "'dark'"} {
		if !strings.Contains(body, themeID) {
			t.Errorf("config.js: MAP_THEME_TO_TILE_VARIANT must include theme %s", themeID)
		}
	}
}

// TestStaticJS_MapTileVariants verifies that MAP_TILE_VARIANTS is declared in
// config.js as an ordered array containing all three supported tile variants.
func TestStaticJS_MapTileVariants(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_TILE_VARIANTS") {
		t.Fatal("config.js: MAP_TILE_VARIANTS constant not found")
	}
	// All three variants must be listed.
	for _, v := range []string{"'light'", "'osm'", "'dark'"} {
		if !strings.Contains(body, v) {
			t.Errorf("config.js: MAP_TILE_VARIANTS must contain variant %s", v)
		}
	}
}

// TestStaticJS_OsmTileInConfigs verifies that the osm tile variant entry in
// TILE_LAYER_CONFIGS points to tile.openstreetmap.org.
func TestStaticJS_OsmTileInConfigs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// 'osm' key must exist as a variant key.
	if !strings.Contains(body, "osm:") && !strings.Contains(body, "'osm'") {
		t.Error("config.js: TILE_LAYER_CONFIGS must declare an osm variant")
	}
	if !strings.Contains(body, "tile.openstreetmap.org") {
		t.Error("config.js: osm variant must use tile.openstreetmap.org URL")
	}
}

// TestStaticJS_CopyrightStartYearConst verifies that app.js declares a
// COPYRIGHT_START_YEAR constant so the copyright year range is driven from a
// single, readable source-of-truth rather than scattered literal values.
func TestStaticJS_CopyrightStartYearConst(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "COPYRIGHT_START_YEAR") {
		t.Error("config.js: COPYRIGHT_START_YEAR constant not found — copyright year logic requires a single source-of-truth")
	}
	// The constant must be assigned a four-digit year value.
	if !strings.Contains(body, "COPYRIGHT_START_YEAR = 2026") {
		t.Error("config.js: COPYRIGHT_START_YEAR must be initialised to 2026")
	}
}

// TestStaticJS_TileLayerConfigsBgColor verifies that every entry in
// TILE_LAYER_CONFIGS declares a bgColor property.  bgColor is the single
// source of truth for the map container background colour; without it the
// white-flash artefact cannot be fixed without hardcoding values elsewhere.
func TestStaticJS_TileLayerConfigsBgColor(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	cfgJS := rec.Body.String()

	cfgStart := strings.Index(cfgJS, "const TILE_LAYER_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("config.js: TILE_LAYER_CONFIGS not found")
	}
	// Extract to the closing brace of the object.
	endIdx := strings.Index(cfgJS[cfgStart:], "\n};")
	var cfgBlock string
	if endIdx != -1 {
		cfgBlock = cfgJS[cfgStart : cfgStart+endIdx+3]
	} else {
		cfgBlock = cfgJS[cfgStart : cfgStart+1500]
	}

	if !strings.Contains(cfgBlock, "bgColor") {
		t.Error("config.js: TILE_LAYER_CONFIGS must include a bgColor property on each entry")
	}
	// All three variants must carry the property.
	for _, variant := range []string{"light", "osm", "dark"} {
		vStart := strings.Index(cfgBlock, variant+":")
		if vStart == -1 {
			t.Errorf("config.js: TILE_LAYER_CONFIGS.%s entry not found", variant)
			continue
		}
		vEnd := strings.Index(cfgBlock[vStart:], "},")
		if vEnd == -1 {
			vEnd = len(cfgBlock) - vStart
		}
		vBlock := cfgBlock[vStart : vStart+vEnd]
		if !strings.Contains(vBlock, "bgColor") {
			t.Errorf("config.js: TILE_LAYER_CONFIGS.%s must have a bgColor property", variant)
		}
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 3) — marker icon redesign & temporary style picker
// ---------------------------------------------------------------------------

// TestStaticJS_MarkerStyleConfigsDefined verifies that config.js declares
// MARKER_STYLE_CONFIGS with the diamond-pulse style entry.
func TestStaticJS_MarkerStyleConfigsDefined(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MARKER_STYLE_CONFIGS") {
		t.Fatal("config.js: MARKER_STYLE_CONFIGS constant must be declared")
	}
	if !strings.Contains(body, "'marker-style-diamond-pulse'") {
		t.Error("config.js: MARKER_STYLE_CONFIGS must include i18nKey 'marker-style-diamond-pulse'")
	}
	// Entry must declare a buildHtml property.
	if !strings.Contains(body, "buildHtml") {
		t.Error("config.js: MARKER_STYLE_CONFIGS entries must declare a buildHtml function")
	}
}

// TestStaticJS_MapPointConfigsShortLabel verifies that MAP_POINT_CONFIGS
// carries a shortLabel property on both 'origin' and 'target' entries.
// shortLabel is passed to buildHtml() so the labeled style can render the
// marker letter without hardcoding it inside the style config.
func TestStaticJS_MapPointConfigsShortLabel(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	cfgStart := strings.Index(body, "const MAP_POINT_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("config.js: MAP_POINT_CONFIGS not found")
	}
	endIdx := strings.Index(body[cfgStart:], "};")
	if endIdx == -1 {
		endIdx = 400
	}
	cfgBlock := body[cfgStart : cfgStart+endIdx+2]
	if !strings.Contains(cfgBlock, "shortLabel") {
		t.Error("config.js: MAP_POINT_CONFIGS must include a shortLabel property for the labeled marker style")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 3) — diamond marker redesign & theme-adaptive tokens
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Phase 7 (Round 4) — marker colour scheme picker + legend sync
// ---------------------------------------------------------------------------

// TestStaticJS_MarkerColorSchemeConfigsDefined verifies that config.js declares
// MARKER_COLOR_SCHEME_CONFIGS with the ocean colour scheme entry.
func TestStaticJS_MarkerColorSchemeConfigsDefined(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MARKER_COLOR_SCHEME_CONFIGS") {
		t.Fatal("config.js: MARKER_COLOR_SCHEME_CONFIGS constant must be declared")
	}
	if !strings.Contains(body, "'marker-color-ocean'") {
		t.Error("config.js: MARKER_COLOR_SCHEME_CONFIGS must include i18nKey 'marker-color-ocean'")
	}
	if !strings.Contains(body, "originColor") || !strings.Contains(body, "targetColor") {
		t.Error("config.js: MARKER_COLOR_SCHEME_CONFIGS entries must declare originColor and targetColor fields")
	}
}

// TestStaticJS_ConnectorLineConfigsDefined verifies that CONNECTOR_LINE_CONFIGS
// contains the single connector style ('tick-xs').
func TestStaticJS_ConnectorLineConfigsDefined(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "CONNECTOR_LINE_CONFIGS") {
		t.Fatal("config.js: CONNECTOR_LINE_CONFIGS not found")
	}
	for _, id := range []string{
		"'tick-xs'",
	} {
		if !strings.Contains(body, id) {
			t.Errorf("config.js: CONNECTOR_LINE_CONFIGS is missing entry %s", id)
		}
	}
}

// TestStaticJS_ConnectorTickXsGlowEnabled verifies that the 'tick-xs' connector
// style config opts into the meteor animation via glowEnabled: true and
// references the 'default' glow preset via glowConfig.
func TestStaticJS_ConnectorTickXsGlowEnabled(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	cfgJS := rec.Body.String()

	if !strings.Contains(cfgJS, "glowEnabled: true") {
		t.Error("config.js: CONNECTOR_LINE_CONFIGS 'tick-xs' must set glowEnabled: true to enable the meteor animation")
	}
	if !strings.Contains(cfgJS, "glowConfig: 'default'") {
		t.Error("config.js: CONNECTOR_LINE_CONFIGS 'tick-xs' must set glowConfig: 'default' to reference the glow preset")
	}
}

// TestStaticJS_ConnectorGlowConfigsFadeMs verifies that CONNECTOR_GLOW_CONFIGS
// declares a fadeMs parameter in the 'default' preset.  fadeMs defines the
// duration of the extinguish phase after the head reaches the destination and
// is the structural requirement for the three-phase animation cycle.
func TestStaticJS_ConnectorGlowConfigsFadeMs(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	cfgJS := rec.Body.String()

	if !strings.Contains(cfgJS, "fadeMs") {
		t.Fatal("config.js: CONNECTOR_GLOW_CONFIGS must declare a fadeMs parameter for the extinguish phase")
	}
	// fadeMs must appear inside the CONNECTOR_GLOW_CONFIGS block.
	cfgStart := strings.Index(cfgJS, "const CONNECTOR_GLOW_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("config.js: CONNECTOR_GLOW_CONFIGS not found")
	}
	cfgEnd := strings.Index(cfgJS[cfgStart:], "};")
	if cfgEnd == -1 {
		cfgEnd = 300
	}
	cfgBlock := cfgJS[cfgStart : cfgStart+cfgEnd+2]
	if !strings.Contains(cfgBlock, "fadeMs") {
		t.Error("config.js: fadeMs must be declared inside CONNECTOR_GLOW_CONFIGS (not elsewhere)")
	}
}

// ── config.js tests ────────────────────────────────────────────────────────────────────

// TestStaticJS_ConfigNamespace verifies that config.js exports PathProbe.Config
// and exposes all expected public constant keys.
func TestStaticJS_ConfigNamespace(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// Namespace assignment must be present.
	if !strings.Contains(body, "PathProbe.Config") {
		t.Error("config.js: must export PathProbe.Config namespace")
	}

	// All public constant keys must appear in the file.
	for _, key := range []string{
		"MAP_POINT_CONFIGS",
		"MARKER_COLOR_SCHEME_CONFIGS",
		"MARKER_STYLE_CONFIGS",
		"CONNECTOR_LINE_CONFIGS",
		"CONNECTOR_GLOW_CONFIGS",
		"MAP_TILE_VARIANTS",
		"MAP_THEME_TO_TILE_VARIANT",
		"MAP_DARK_THEMES",
		"TILE_LAYER_CONFIGS",
		"TARGET_PORTS",
		"TARGET_MODE_PANELS",
		"WEB_MODES_WITH_PORTS",
		"TARGET_PLACEHOLDER_KEYS",
		"COPYRIGHT_START_YEAR",
		"THEMES",
		"DEFAULT_THEME",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("config.js: missing constant key %q", key)
		}
	}

	// The constant declarations must be present.
	for _, decl := range []string{
		"const DEFAULT_THEME",
		"const THEMES",
		"const COPYRIGHT_START_YEAR",
	} {
		if !strings.Contains(body, decl) {
			t.Errorf("config.js: expected constant declaration %q", decl)
		}
	}
}

// TestStaticJS_ConfigNoFunctions verifies that config.js is a pure data layer
// and contains no function declarations (only arrow-function values in data
// properties such as MARKER_STYLE_CONFIGS.buildHtml are permitted).
func TestStaticJS_ConfigNoFunctions(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// The keyword 'function ' (with trailing space) identifies named function
	// declarations or function expressions.  Arrow functions (=>) do not match
	// and are permitted as inline data-property values (e.g. buildHtml).
	if strings.Contains(body, "function ") {
		t.Error("config.js: must not contain function declarations — pure data layer only")
	}
}

// TestStaticJS_ConfigScopeIsolation verifies that config.js wraps all its
// const declarations inside an arrow IIFE (Immediately-Invoked Function
// Expression) to prevent the "redeclaration of const" SyntaxError that
// browsers report when multiple classic <script> elements declare const
// variables with the same name in the shared global script scope.
//
// Background: in a browser, all classic (non-module) <script> tags share one
// "script scope" for const/let.  config.js declares e.g.
//
//	const CONNECTOR_LINE_CONFIGS = {...}
//
// and app.js destructures:
//
//	const { CONNECTOR_LINE_CONFIGS, ... } = window.PathProbe.Config
//
// Both declarations use the same identifier and therefore collide.  The
// SyntaxError is a parse-time failure: the entire app.js script is rejected
// before a single function is defined, which is why setTheme / setLocale /
// runDiag are all unreachable from inline onclick attributes.
//
// The arrow-IIFE wrapper `(() => { ... })()` moves config.js constants into a
// function scope, preventing them from appearing in the shared script scope.
func TestStaticJS_ConfigScopeIsolation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// config.js must open an arrow IIFE so const declarations are not in the
	// shared script scope.  Both compact and spaced variants are accepted.
	hasArrowIIFE := strings.Contains(body, "(() => {") || strings.Contains(body, "(()=>{")
	if !hasArrowIIFE {
		t.Error("config.js: all constants must be wrapped in an arrow IIFE (() => { ... })() " +
			"to prevent 'redeclaration of const' SyntaxErrors when the browser " +
			"loads both config.js and app.js in the same script scope")
	}

	// The IIFE must be properly closed so the code executes immediately.
	if !strings.Contains(body, "})()") {
		t.Error("config.js: the arrow IIFE must be closed with })() or })(); " +
			"without the closing invocation the constants are never assigned to " +
			"PathProbe.Config and window.PathProbe")
	}
}
// TestStaticJS_GeoSameRegionThresholdKM 驗證 config.js 導出了
// GEO_SAME_REGION_THRESHOLD_KM 常數，以導驅 map.js 的返回區域返回區域接近避免下的 zoom 决策。
// 該常數展水自 config.js ，避免在 map.js 中硬編碼數字。
func TestStaticJS_GeoSameRegionThresholdKM(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// 常數必須存在。
	if !strings.Contains(body, "GEO_SAME_REGION_THRESHOLD_KM") {
		t.Error("config.js: GEO_SAME_REGION_THRESHOLD_KM constant must be declared")
	}
	// 必須導出到 PathProbe.Config。
	cfgStart := strings.Index(body, "PathProbe.Config = {")
	if cfgStart == -1 {
		t.Fatal("config.js: PathProbe.Config export block not found")
	}
	cfgEnd := strings.Index(body[cfgStart:], "};")
	if cfgEnd == -1 {
		t.Fatal("config.js: closing }; of PathProbe.Config not found")
	}
	cfgBlock := body[cfgStart : cfgStart+cfgEnd]
	if !strings.Contains(cfgBlock, "GEO_SAME_REGION_THRESHOLD_KM") {
		t.Error("config.js: GEO_SAME_REGION_THRESHOLD_KM must be exported inside PathProbe.Config")
	}
}