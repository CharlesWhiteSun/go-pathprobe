package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


// TestStaticJS_RenderMapInvalidateSize verifies that renderMap() defers a call
// to _map.invalidateSize() via requestAnimationFrame so Leaflet re-projects all
// tiles after the #results section transitions from display:none to display:block.
// Without this, Leaflet sees a 0×0 container at init time and leaves large
// blank grey areas on the OpenStreetMap canvas.
func TestStaticJS_RenderMapInvalidateSize(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	// renderMap must call invalidateSize to correct blank-tile regression.
	if !strings.Contains(body, "invalidateSize") {
		t.Error("map.js: renderMap must call _map.invalidateSize() to fix blank tile regression when container was hidden")
	}
	// The call must be deferred via requestAnimationFrame so it runs after the
	// browser has re-laid-out the newly visible container.
	if !strings.Contains(body, "requestAnimationFrame") {
		t.Error("map.js: invalidateSize must be deferred via requestAnimationFrame so layout is complete before tiles repaint")
	}
}

// TestStaticJS_HaversineKm verifies that app.js defines a haversineKm()
// helper for computing the great-circle distance.  This powers the distance
// badge displayed below the map between origin and target markers.
func TestStaticJS_HaversineKm(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function haversineKm(") {
		t.Error("map.js: haversineKm function not found — distance calculation must be a named helper")
	}
	// Earth radius constant must appear to confirm correct formula.
	if !strings.Contains(body, "6371") {
		t.Error("map.js: haversineKm must use Earth radius constant 6371 km")
	}
}

// TestStaticJS_BuildMarkerIcon verifies that app.js defines buildMarkerIcon()
// which creates L.divIcon instances driven by MAP_POINT_CONFIGS, replacing the
// default Leaflet marker pin with a role-coloured dot.
func TestStaticJS_BuildMarkerIcon(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function buildMarkerIcon(") {
		t.Error("map.js: buildMarkerIcon function not found — marker icon creation must be a named helper")
	}
	if !strings.Contains(body, "L.divIcon(") {
		t.Error("map.js: buildMarkerIcon must use L.divIcon for custom marker styling")
	}
}

// TestStaticJS_BuildPopupHtml verifies that app.js defines buildPopupHtml()
// which constructs a rich HTML popup from a GeoAnnotation, using the
// geo-popup__role badge to clearly identify origin vs target.
func TestStaticJS_BuildPopupHtml(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function buildPopupHtml(") {
		t.Error("map.js: buildPopupHtml function not found")
	}
	if !strings.Contains(body, "geo-popup__role") {
		t.Error("map.js: buildPopupHtml must emit .geo-popup__role element for visual role identification")
	}
	if !strings.Contains(body, "geo-popup__ip") {
		t.Error("map.js: buildPopupHtml must emit .geo-popup__ip element for the IP address")
	}
}

// TestStaticJS_RenderMapPolyline verifies that renderMap() draws a connector
// between origin and target to give users a clear visual probe direction.
// The connector is now rendered by ConnectorArcLayer (HTML5 canvas) rather
// than a raw L.polyline, so the test checks for buildConnectorLayer() and
// that dot/dash rhythms are configured via dashArray in CONNECTOR_LINE_CONFIGS.
func TestStaticJS_RenderMapPolyline(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "buildConnectorLayer(") {
		t.Error("map.js: renderMap must call buildConnectorLayer() to connect origin and target markers")
	}
	// dashArray configuration lives in config.js (CONNECTOR_LINE_CONFIGS presets).
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "dashArray") {
		t.Error("config.js: CONNECTOR_LINE_CONFIGS must include dashArray entries for dot/dash rhythm styles")
	}
}

// TestStaticJS_GetMapTileVariant verifies that app.js exposes a named
// getMapTileVariant() function which is the single decision point for
// mapping the active application theme to a tile-layer variant string.
func TestStaticJS_GetMapTileVariant(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function getMapTileVariant(") {
		t.Error("map.js: getMapTileVariant function not found")
	}
}

// TestStaticJS_RefreshMapTiles verifies that map.js exposes a named
// refreshMapTiles() function that swaps the tile layer on the live map
// with a fade-out/fade-in animation.  It is called only from
// setMapTileVariant() (user-driven tile changes).  Theme-triggered tile swaps
// are handled silently by syncMapTileVariantToTheme().
func TestStaticJS_RefreshMapTiles(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function refreshMapTiles(") {
		t.Error("map.js: refreshMapTiles function not found")
	}
}

// TestStaticJS_MapBarHiddenToggled verifies that renderMap() removes the hidden
// attribute from #geo-map-outer when the map is shown and sets it when hidden,
// so the tile-variant selector bar (inside the outer wrapper) is visible exactly
// when the map is visible.
func TestStaticJS_MapBarHiddenToggled(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
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

	// Both show and hide paths must reference the outer wrapper element and toggle hidden.
	if !strings.Contains(fnBody, "geo-map-outer") {
		t.Error("map.js: renderMap must reference geo-map-outer to toggle its visibility")
	}
	if !strings.Contains(fnBody, "hidden = false") && !strings.Contains(fnBody, "removeAttribute('hidden')") {
		t.Error("map.js: renderMap must reveal #geo-map-outer (hidden = false) when map is shown")
	}
	if !strings.Contains(fnBody, "hidden = true") && !strings.Contains(fnBody, "setAttribute('hidden'") {
		t.Error("map.js: renderMap must hide #geo-map-outer (hidden = true) when map is hidden")
	}
}

// TestStaticJS_RefreshMapTilesRequestAnimationFrame verifies that the updated
// refreshMapTiles() uses requestAnimationFrame to remove the fading class after
// the tile swap, rather than registering a second transitionend listener that
// would never fire (since removing the class triggers the transition, not ends it).
func TestStaticJS_RefreshMapTilesRequestAnimationFrame(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function refreshMapTiles(")
	if fnStart == -1 {
		t.Fatal("map.js: refreshMapTiles function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "requestAnimationFrame") {
		t.Error("map.js: refreshMapTiles must use requestAnimationFrame to remove geo-map--fading after tile swap")
	}
	if !strings.Contains(fnBody, "propertyName") {
		t.Error("map.js: refreshMapTiles transitionend handler must filter by e.propertyName to avoid acting on bubbling child events")
	}
}

// TestStaticJS_SyncMapTileVariantNoFadeAnimation verifies that
// syncMapTileVariantToTheme() does NOT call refreshMapTiles(), ensuring the
// theme-driven tile swap is always silent (no map fade animation).  The fade
// would be redundant because the body is already invisible during a theme
// transition, and the second transitionend listener in the old refreshMapTiles
// would leave geo-map--fading stuck permanently.
func TestStaticJS_SyncMapTileVariantNoFadeAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function syncMapTileVariantToTheme(")
	if fnStart == -1 {
		t.Fatal("map.js: syncMapTileVariantToTheme function not found")
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

	// Must NOT delegate to animated refreshMapTiles — silent swap only.
	if strings.Contains(fnBody, "refreshMapTiles()") {
		t.Error("map.js: syncMapTileVariantToTheme must NOT call refreshMapTiles() — tile swap must be silent during theme transitions")
	}
}

// TestStaticJS_SetMapTileVariant verifies that app.js exposes a named
// setMapTileVariant() function which is called from the map bar buttons.
func TestStaticJS_SetMapTileVariant(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function setMapTileVariant(") {
		t.Error("map.js: setMapTileVariant function not found")
	}
}

// TestStaticJS_RenderMapBar verifies that app.js exposes a named renderMapBar()
// function that builds the three tile-variant buttons above the map.
func TestStaticJS_RenderMapBar(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function renderMapBar(") {
		t.Error("map.js: renderMapBar function not found")
	}
}

// TestStaticJS_SyncMapTileVariantToTheme verifies that map.js exposes a named
// syncMapTileVariantToTheme() function which is called via PathProbe.Map by
// theme.js to align the tile variant with the active colour theme.
func TestStaticJS_SyncMapTileVariantToTheme(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function syncMapTileVariantToTheme(") {
		t.Error("map.js: syncMapTileVariantToTheme function not found")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 fix-2 tests — color-scheme / dot buttons / overlay wrapper
// ---------------------------------------------------------------------------

// TestStaticJS_RenderMapUsesOuterWrapper verifies that renderMap() references
// geo-map-outer to toggle the entire map area (wrapper + bar + map) as one unit.
func TestStaticJS_RenderMapUsesOuterWrapper(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
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

	if !strings.Contains(fnBody, "geo-map-outer") {
		t.Error("map.js: renderMap must reference geo-map-outer to toggle map area visibility")
	}
}

// TestStaticJS_RenderMapBarNoTextContent verifies that renderMapBar() produces
// buttons without text content — dot-only style, accessible via aria-label.
func TestStaticJS_RenderMapBarNoTextContent(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMapBar(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMapBar function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// Must have aria-label for accessibility.
	if !strings.Contains(fnBody, "aria-label") {
		t.Error("map.js: renderMapBar buttons must include aria-label for accessibility")
	}
	// Must have title for native tooltip.
	if !strings.Contains(fnBody, "title=") {
		t.Error("map.js: renderMapBar buttons should include title attribute for tooltip")
	}
	// The button closing tag must immediately follow the opening tag (no text node).
	// Check that the inner text is NOT rendered (no i18nKey value as text content).
	if strings.Contains(fnBody, ">'"+"\n") || strings.Contains(fnBody, "> +\n      esc(t(") {
		t.Error("map.js: renderMapBar must not render i18n text inside the button element")
	}
}

// TestStaticJS_ApplyMapBgColorFunction verifies that app.js defines an
// applyMapBgColor() function that reads bgColor from TILE_LAYER_CONFIGS and
// applies it to the map container, acting as the single point responsible for
// the background-colour update.
func TestStaticJS_ApplyMapBgColorFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function applyMapBgColor(")
	if fnStart == -1 {
		t.Fatal("map.js: applyMapBgColor function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 400
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "bgColor") {
		t.Error("map.js: applyMapBgColor must read bgColor from TILE_LAYER_CONFIGS")
	}
	if !strings.Contains(fnBody, "background") {
		t.Error("map.js: applyMapBgColor must set container.style.background")
	}
}

// TestStaticJS_RefreshMapTilesCallsApplyMapBgColor verifies that the animated
// tile swap path in refreshMapTiles() calls applyMapBgColor() after the new
// tile layer is added, so the container background is correct before the map
// fades back in.
func TestStaticJS_RefreshMapTilesCallsApplyMapBgColor(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function refreshMapTiles(")
	if fnStart == -1 {
		t.Fatal("map.js: refreshMapTiles function not found")
	}
	// Functions inside the IIFE are indented, so look for the next indented function.
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "applyMapBgColor") {
		t.Error("map.js: refreshMapTiles must call applyMapBgColor() after swapping tiles to prevent white-flash on dark tile load")
	}
}

// TestStaticJS_BuildMarkerIconUsesStyleConfig verifies that buildMarkerIcon()
// reads both MAP_POINT_CONFIGS (for role colour / class) and
// MARKER_STYLE_CONFIGS (for shape / size), combining them into a single
// L.divIcon — clean separation of role vs. visual style.
func TestStaticJS_BuildMarkerIconUsesStyleConfig(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildMarkerIcon(")
	if fnStart == -1 {
		t.Fatal("map.js: buildMarkerIcon function not found")
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

	if !strings.Contains(fnBody, "MARKER_STYLE_CONFIGS") {
		t.Error("map.js: buildMarkerIcon must read MARKER_STYLE_CONFIGS for the shape/size data")
	}
	if !strings.Contains(fnBody, "buildHtml") {
		t.Error("map.js: buildMarkerIcon must call styleCfg.buildHtml to produce the inner HTML")
	}
	if !strings.Contains(fnBody, "_markerStyleId") {
		t.Error("map.js: buildMarkerIcon must use _markerStyleId to select the active style config")
	}
}

// TestStaticJS_RefreshMapMarkersFunction verifies that app.js defines
// refreshMapMarkers() which replaces only the Marker layers on the live map
// without destroying the tile layer, polyline, or legend.
func TestStaticJS_RefreshMapMarkersFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function refreshMapMarkers(") {
		t.Fatal("map.js: refreshMapMarkers function not found")
	}
	fnStart := strings.Index(body, "function refreshMapMarkers(")
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	// Must iterate layers and remove Marker instances only.
	if !strings.Contains(fnBody, "eachLayer") {
		t.Error("map.js: refreshMapMarkers must use _map.eachLayer to locate existing markers")
	}
	if !strings.Contains(fnBody, "L.Marker") {
		t.Error("map.js: refreshMapMarkers must guard removal with 'instanceof L.Marker'")
	}
	if !strings.Contains(fnBody, "_lastPub") && !strings.Contains(fnBody, "_lastTgt") {
		t.Error("map.js: refreshMapMarkers must use _lastPub/_lastTgt to recreate markers")
	}
}

// TestStaticJS_RenderMapStoresLastGeo verifies that renderMap() stores the
// pub and tgt arguments into _lastPub and _lastTgt so that refreshMapMarkers()
// can recreate markers without requiring a full map rebuild.
func TestStaticJS_RenderMapStoresLastGeo(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
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

	if !strings.Contains(fnBody, "_lastPub") {
		t.Error("map.js: renderMap must store the pub argument into _lastPub")
	}
	if !strings.Contains(fnBody, "_lastTgt") {
		t.Error("map.js: renderMap must store the tgt argument into _lastTgt")
	}
	if !strings.Contains(fnBody, "applyMarkerColorScheme()") {
		t.Error("map.js: renderMap must call applyMarkerColorScheme() to apply the initial colour scheme")
	}
}

// TestStaticJS_MarkerColorSchemeStateVars verifies that app.js declares the
// _markerColorSchemeId and _legendControl module-level state variables.
func TestStaticJS_MarkerColorSchemeStateVars(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "_markerColorSchemeId") {
		t.Error("map.js: _markerColorSchemeId state variable not declared")
	}
	if !strings.Contains(body, "_legendControl") {
		t.Error("map.js: _legendControl state variable not declared")
	}
}

// TestStaticJS_ColorSchemeFunctions verifies that app.js defines
// applyMarkerColorScheme() which applies the active colour scheme.
func TestStaticJS_ColorSchemeFunctions(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function applyMarkerColorScheme") {
		t.Error("map.js: function applyMarkerColorScheme not found")
	}

	// applyMarkerColorScheme must set --mc-origin and --mc-target on the root element.
	fnStart := strings.Index(body, "function applyMarkerColorScheme")
	if fnStart != -1 {
		nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
		var fnBody string
		if nextFn != -1 {
			fnBody = body[fnStart : fnStart+1+nextFn]
		} else {
			end := fnStart + 800
			if end > len(body) {
				end = len(body)
			}
			fnBody = body[fnStart:end]
		}
		if !strings.Contains(fnBody, "--mc-origin") {
			t.Error("map.js: applyMarkerColorScheme must set the --mc-origin CSS custom property")
		}
		if !strings.Contains(fnBody, "--mc-target") {
			t.Error("map.js: applyMarkerColorScheme must set the --mc-target CSS custom property")
		}
	}
}

// TestStaticJS_BuildMapLegendUsesBuildHtml verifies that buildMapLegend()
// uses MARKER_STYLE_CONFIGS[_markerStyleId].buildHtml(roleCfg) to produce
// the legend icon so it mirrors the active marker style (legend sync fix).
func TestStaticJS_BuildMapLegendUsesBuildHtml(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildMapLegend(")
	if fnStart == -1 {
		t.Fatal("map.js: buildMapLegend function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "MARKER_STYLE_CONFIGS") {
		t.Error("map.js: buildMapLegend must use MARKER_STYLE_CONFIGS to look up the active style")
	}
	if !strings.Contains(fnBody, "buildHtml") {
		t.Error("map.js: buildMapLegend must call buildHtml() to produce the legend icon (legend sync)")
	}
	if !strings.Contains(fnBody, "geo-legend__marker") {
		t.Error("map.js: buildMapLegend must apply the .geo-legend__marker CSS class to the icon wrapper")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 5) — fixed marker appearance + colour scheme + legend i18n
// ---------------------------------------------------------------------------

// TestStaticJS_BuildMapLegendDataI18nAttribute verifies that buildMapLegend()
// adds a data-i18n attribute to the label span in each legend item so that
// applyLocale() can update the text reactively when the user switches language.
// Without this attribute the legend text would be frozen at creation time and
// never reflect a locale change.
func TestStaticJS_BuildMapLegendDataI18nAttribute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildMapLegend(")
	if fnStart == -1 {
		t.Fatal("map.js: buildMapLegend function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "data-i18n") {
		t.Error("map.js: buildMapLegend must add a data-i18n attribute to the legend label span so applyLocale() can update it on language change")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 9) tests — gradient arc connector between origin and target
// ---------------------------------------------------------------------------

// TestStaticJS_ConnectorLineStateVars verifies that app.js declares the
// module-level state variables used by the connector arc system.
func TestStaticJS_ConnectorLineStateVars(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	for _, decl := range []string{"let _connectorStyleId", "let _connectorLayer"} {
		if !strings.Contains(body, decl) {
			t.Errorf("map.js: module-level declaration %q not found — required by the connector arc system", decl)
		}
	}
}

// TestStaticJS_RenderMapUsesConnectorLayer verifies that renderMap() calls
// buildConnectorLayer() to draw the gradient arc instead of a plain polyline.
func TestStaticJS_RenderMapUsesConnectorLayer(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
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

	if !strings.Contains(fnBody, "buildConnectorLayer(") {
		t.Error("map.js: renderMap must call buildConnectorLayer() to draw the gradient arc connector")
	}
	if strings.Contains(fnBody, "color: '#5b8dee'") {
		t.Error("map.js: renderMap must not use the hardcoded '#5b8dee' polyline — use buildConnectorLayer() instead")
	}
}

// TestStaticJS_ConnectorDefaultIsTickXs verifies that the initial connector
// style identifier is set to 'tick-xs'.
func TestStaticJS_ConnectorDefaultIsTickXs(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "_connectorStyleId = 'tick-xs'") {
		t.Error("map.js: _connectorStyleId must default to 'tick-xs'")
	}
}

// TestStaticJS_IsMapLoadedHelper verifies that app.js defines an isMapLoaded()
// shim and that map-connector.js guards buildArrowConnectorLayer with isMapLoaded().
func TestStaticJS_IsMapLoadedHelper(t *testing.T) {
	h := newStaticHandler(t)

	// ── map.js contains the private isMapLoaded() helper ────────────────────
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec.Code)
	}
	mapJS := rec.Body.String()

	if !strings.Contains(mapJS, "function isMapLoaded()") {
		t.Fatal("map.js: isMapLoaded() helper not found")
	}
	fnStart := strings.Index(mapJS, "function isMapLoaded()")
	fnEnd := strings.Index(mapJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if fnEnd != -1 {
		fnBody = mapJS[fnStart : fnStart+1+fnEnd]
	} else {
		end := fnStart + 200
		if end > len(mapJS) {
			end = len(mapJS)
		}
		fnBody = mapJS[fnStart:end]
	}
	if !strings.Contains(fnBody, "_map._loaded") {
		t.Error("map.js: isMapLoaded() must check _map._loaded")
	}

	// ── map-connector.js: buildArrowConnectorLayer must guard with isMapLoaded() ──
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec2.Code)
	}
	mcJS := rec2.Body.String()

	arrowStart := strings.Index(mcJS, "function buildArrowConnectorLayer(")
	if arrowStart == -1 {
		t.Fatal("map-connector.js: buildArrowConnectorLayer not found")
	}
	arrowEnd := strings.Index(mcJS[arrowStart+1:], "\n  function ")
	var arrowBody string
	if arrowEnd != -1 {
		arrowBody = mcJS[arrowStart : arrowStart+1+arrowEnd]
	} else {
		end := arrowStart + 6000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		arrowBody = mcJS[arrowStart:end]
	}
	if !strings.Contains(arrowBody, "isMapLoaded(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must guard with isMapLoaded() before calling latLngToLayerPoint()")
	}

	// ── map.js: refreshConnectorLayer must guard with isMapLoaded() ──────────
	if !strings.Contains(mapJS, "function refreshConnectorLayer(") {
		t.Fatal("map.js: refreshConnectorLayer not found")
	}
	rlStart := strings.Index(mapJS, "function refreshConnectorLayer(")
	rlEnd := strings.Index(mapJS[rlStart+1:], "\nfunction ")
	var rlBody string
	if rlEnd != -1 {
		rlBody = mapJS[rlStart : rlStart+1+rlEnd]
	} else {
		end := rlStart + 400
		if end > len(mapJS) {
			end = len(mapJS)
		}
		rlBody = mapJS[rlStart:end]
	}
	if !strings.Contains(rlBody, "isMapLoaded()") {
		t.Error("map.js: refreshConnectorLayer must guard with isMapLoaded() to prevent premature map operations")
	}
}

// TestStaticJS_RenderMapSetsViewBeforeConnector verifies that renderMap()
// calls setView / fitBounds before buildConnectorLayer() so the Leaflet map
// is fully initialised when latLngToLayerPoint() is first invoked.
func TestStaticJS_RenderMapSetsViewBeforeConnector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
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

	// Find relative positions: setView/fitBounds must appear before buildConnectorLayer.
	setViewIdx := strings.Index(fnBody, ".setView(")
	fitBoundsIdx := strings.Index(fnBody, ".fitBounds(")
	connectorIdx := strings.Index(fnBody, "buildConnectorLayer(")

	if setViewIdx == -1 && fitBoundsIdx == -1 {
		t.Fatal("map.js: renderMap must call setView() or fitBounds() to initialise the map")
	}
	if connectorIdx == -1 {
		t.Fatal("map.js: renderMap must call buildConnectorLayer()")
	}

	// At least one viewport-setting call must precede buildConnectorLayer.
	viewportIdx := setViewIdx
	if fitBoundsIdx != -1 && (viewportIdx == -1 || fitBoundsIdx < viewportIdx) {
		viewportIdx = fitBoundsIdx
	}
	if viewportIdx >= connectorIdx {
		t.Error("map.js: renderMap must call setView/fitBounds BEFORE buildConnectorLayer() " +
			"so the Leaflet map is loaded before latLngToLayerPoint() is invoked")
	}
}
