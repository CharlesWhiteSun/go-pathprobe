package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


// TestStaticJS_ConnectorLineFunctions verifies that map-connector.js defines
// the arc-rendering helpers and that app.js retains refreshConnectorLayer.
func TestStaticJS_ConnectorLineFunctions(t *testing.T) {
	h := newStaticHandler(t)
	// map-connector.js must expose the extracted arc-rendering helpers.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	for _, fn := range []string{
		"function lerpHex(",
		"function buildArcLatLngs(",
		"function buildArrowConnectorLayer(",
		"function buildConnectorLayer(",
	} {
		if !strings.Contains(mcJS, fn) {
			t.Errorf("map-connector.js: function %q not found — required by the connector arc feature", fn)
		}
	}

	// map.js contains refreshConnectorLayer (private, injecting _map).
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "function refreshConnectorLayer(") {
		t.Error("map.js: function refreshConnectorLayer not found — required by the connector arc system")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 10) tests — 10 line-pattern styles + temporary picker
// ---------------------------------------------------------------------------

// TestStaticJS_BuildArrowConnectorLayerFunction verifies that map-connector.js defines
// buildArrowConnectorLayer() and that it renders directional symbols using
// pixel-distance-based placement (consistent density at every zoom level).
func TestStaticJS_BuildArrowConnectorLayerFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArrowConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArrowConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 6000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	// Delegates to the SVG helper.
	if !strings.Contains(fnBody, "buildArrowSVG(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must call buildArrowSVG() to render arrow icons")
	}
	// Shape is read from config, not a hardcoded Unicode glyph.
	if !strings.Contains(fnBody, "arrowShape") {
		t.Error("map-connector.js: buildArrowConnectorLayer must read styleCfg.arrowShape to select the SVG shape")
	}
	// Pixel-distance-based placement: cumulative distance table + arrowSpacing.
	if !strings.Contains(fnBody, "cum") {
		t.Error("map-connector.js: buildArrowConnectorLayer must build a cumulative pixel-distance table ('cum') for even spacing")
	}
	if !strings.Contains(fnBody, "arrowSpacing") {
		t.Error("map-connector.js: buildArrowConnectorLayer must read styleCfg.arrowSpacing to control symbol density")
	}
	// Rotation from screen-space tangent.
	if !strings.Contains(fnBody, "latLngToLayerPoint(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must call latLngToLayerPoint() to compute the arc tangent angle")
	}
	if !strings.Contains(fnBody, "atan2(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must use Math.atan2() to derive arrow rotation from arc direction")
	}
}

// TestStaticJS_BuildArcLatLngsMercatorSpace verifies that buildArcLatLngs()
// computes the Bézier arc in Web-Mercator (EPSG:3857) space so the rendered
// curve is geometrically smooth on the Leaflet Mercator map.
func TestStaticJS_BuildArcLatLngsMercatorSpace(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArcLatLngs(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArcLatLngs function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	// Must contain Mercator forward projection (toMerc) and inverse (fromMerc).
	if !strings.Contains(fnBody, "toMerc") {
		t.Error("map-connector.js: buildArcLatLngs must define a toMerc helper for forward Web-Mercator projection")
	}
	if !strings.Contains(fnBody, "fromMerc") {
		t.Error("map-connector.js: buildArcLatLngs must define a fromMerc helper for inverse Web-Mercator projection")
	}
	// Earth radius constant must be present for EPSG:3857 math.
	if !strings.Contains(fnBody, "6378137") {
		t.Error("map-connector.js: buildArcLatLngs must use the WGS-84 Earth radius (6378137) for Mercator conversion")
	}
}

// TestStaticJS_BuildConnectorLayerDispatchesByType verifies that
// buildConnectorLayer() delegates to buildArrowConnectorLayer() when the
// style config declares type === 'arrows', following the Open/Closed principle.
func TestStaticJS_BuildConnectorLayerDispatchesByType(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "buildArrowConnectorLayer(") {
		t.Error("map-connector.js: buildConnectorLayer must call buildArrowConnectorLayer() for 'arrows' type styles")
	}
	if !strings.Contains(fnBody, "'arrows'") {
		t.Error("map-connector.js: buildConnectorLayer must check for type === 'arrows' to dispatch correctly")
	}
}

// TestStaticJS_BuildArrowSVGHelper verifies that map-connector.js defines a buildArrowSVG()
// helper that renders all shape variants as inline SVG using a normalised viewBox.
// SVG-based arrows avoid Unicode glyph size/font variance and ensure
// pixel-accurate arrowheads at every zoom level.
func TestStaticJS_BuildArrowSVGHelper(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArrowSVG(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArrowSVG helper function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	// Must produce inline SVG output.
	if !strings.Contains(fnBody, "viewBox") {
		t.Error("map-connector.js: buildArrowSVG must use an SVG viewBox for normalised coordinate rendering")
	}
	// Must handle all defined shape variants via switch.
	for _, shape := range []string{"chevron", "double", "open", "pointer", "fat"} {
		if !strings.Contains(fnBody, "'"+shape+"'") {
			t.Errorf("map-connector.js: buildArrowSVG is missing a case for shape %q", shape)
		}
	}
	// Rotation must be applied via SVG transform (not CSS) for anchor consistency.
	if !strings.Contains(fnBody, "rotate(") {
		t.Error("map-connector.js: buildArrowSVG must apply rotation via SVG transform rotate()")
	}
}

// TestStaticJS_ConnectorArcLayerSinglePassRendering verifies that
// ConnectorArcLayer._redraw() draws the full arc in a single canvas drawing
// pass using setLineDash (for seamless dot/dash patterns) and
// createLinearGradient (for smooth colour gradient).  This replaces the old
// N-polyline + dashOffset approach which produced doubled end-caps and
// float-precision seams at every segment boundary.
func TestStaticJS_ConnectorArcLayerSinglePassRendering(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	// ConnectorArcLayer must be defined as a L.Layer extension.
	if !strings.Contains(mcJS, "ConnectorArcLayer") {
		t.Fatal("map-connector.js: ConnectorArcLayer not found")
	}
	// Find the _redraw method body.
	redrawStart := strings.Index(mcJS, "_redraw: function()")
	if redrawStart == -1 {
		t.Fatal("map-connector.js: ConnectorArcLayer._redraw method not found")
	}
	nextFn := strings.Index(mcJS[redrawStart+1:], "\n    },")
	var redrawBody string
	if nextFn != -1 {
		redrawBody = mcJS[redrawStart : redrawStart+1+nextFn]
	} else {
		end := redrawStart + 3000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		redrawBody = mcJS[redrawStart:end]
	}
	if !strings.Contains(redrawBody, "setLineDash(") {
		t.Error("map-connector.js: ConnectorArcLayer._redraw must call setLineDash() for seamless dot/dash patterns")
	}
	if !strings.Contains(redrawBody, "createLinearGradient(") {
		t.Error("map-connector.js: ConnectorArcLayer._redraw must call createLinearGradient() for smooth colour gradient")
	}
	if !strings.Contains(redrawBody, "ctx.stroke()") {
		t.Error("map-connector.js: ConnectorArcLayer._redraw must call ctx.stroke() to render the arc")
	}
}

// TestStaticJS_BuildConnectorLayerUsesArcLayer verifies that
// buildConnectorLayer() delegates dot/dash arc rendering to ConnectorArcLayer
// (a single-canvas-path Leaflet layer) instead of creating N gradient
// sub-polylines.  The single-pass architecture is the correct fix for
// the visual discontinuities of the old polyline-per-segment approach.
func TestStaticJS_BuildConnectorLayerUsesArcLayer(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "ConnectorArcLayer") {
		t.Error("map-connector.js: buildConnectorLayer must instantiate ConnectorArcLayer for polyline-type styles")
	}
	if !strings.Contains(fnBody, "group.addLayer(") {
		t.Error("map-connector.js: buildConnectorLayer must add ConnectorArcLayer to the LayerGroup via addLayer()")
	}
}

// TestStaticJS_BuildArrowConnectorLayerSpine verifies that
// buildArrowConnectorLayer() supports an optional spine drawn beneath the
// arrow icons when styleCfg.spineWeight > 0.  The spine must use
// ConnectorArcLayer so it benefits from the same single-canvas-pass
// seamless gradient rendering as the dot-family connector styles.
func TestStaticJS_BuildArrowConnectorLayerSpine(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArrowConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArrowConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 6000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "spineWeight") {
		t.Error("map-connector.js: buildArrowConnectorLayer must read styleCfg.spineWeight to conditionally draw a spine")
	}
	if !strings.Contains(fnBody, "ConnectorArcLayer") {
		t.Error("map-connector.js: buildArrowConnectorLayer spine must use ConnectorArcLayer for seamless single-pass rendering")
	}
}

// TestStaticJS_HexToRgbaHelper verifies that map-connector.js defines a hexToRgba()
// helper that converts a '#rrggbb' hex colour and an alpha value [0,1] to the
// rgba() CSS format required by ConnectorArcLayer for canvas strokeStyle.
func TestStaticJS_HexToRgbaHelper(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "function hexToRgba(") {
		t.Fatal("map-connector.js: hexToRgba() helper function not found")
	}
	fnStart := strings.Index(mcJS, "function hexToRgba(")
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}
	if !strings.Contains(fnBody, "rgba(") {
		t.Error("map-connector.js: hexToRgba() must produce an rgba() CSS string")
	}
	if !strings.Contains(fnBody, "parseInt(") {
		t.Error("map-connector.js: hexToRgba() must parse hex channel values with parseInt()")
	}
}

// TestStaticJS_ConnectorArcLayerDefined verifies that map-connector.js defines
// ConnectorArcLayer as a L.Layer extension with all required lifecycle methods,
// map event bindings, and canvas placement inside the map container.
func TestStaticJS_ConnectorArcLayerDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "ConnectorArcLayer = L.Layer.extend(") {
		t.Fatal("map-connector.js: ConnectorArcLayer must be defined as a L.Layer extension")
	}
	for _, method := range []string{"initialize: function(", "onAdd: function(", "onRemove: function(", "_redraw: function("} {
		if !strings.Contains(mcJS, method) {
			t.Errorf("map-connector.js: ConnectorArcLayer must define the %q method", method)
		}
	}
	if !strings.Contains(mcJS, "map.on('move zoom zoomend resize'") {
		t.Error("map-connector.js: ConnectorArcLayer.onAdd must bind 'move zoom zoomend resize' map events")
	}
	if !strings.Contains(mcJS, "map.off('move zoom zoomend resize'") {
		t.Error("map-connector.js: ConnectorArcLayer.onRemove must unbind 'move zoom zoomend resize' map events")
	}
	if !strings.Contains(mcJS, "map.getContainer().appendChild(") {
		t.Error("map-connector.js: ConnectorArcLayer.onAdd must append the canvas to map.getContainer()")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 11) tests — meteor glow animation on connector arc
// ---------------------------------------------------------------------------

// TestStaticJS_ConnectorGlowConfigsDefined verifies that map-connector.js references
// CONNECTOR_GLOW_CONFIGS and uses all required timing and visual parameters
// for the meteor animation.
func TestStaticJS_ConnectorGlowConfigsDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "CONNECTOR_GLOW_CONFIGS") {
		t.Fatal("map-connector.js: CONNECTOR_GLOW_CONFIGS must be referenced")
	}
	// 'default' preset must define all required animation parameters.
	for _, param := range []string{"travelMs", "pauseMs", "glowRadius", "glowOpacity", "tailLength"} {
		if !strings.Contains(mcJS, param) {
			t.Errorf("map-connector.js: CONNECTOR_GLOW_CONFIGS must include parameter %q", param)
		}
	}
}

// TestStaticJS_ConnectorGlowLayerDefined verifies that map-connector.js defines
// ConnectorGlowLayer as a L.Layer extension with the required lifecycle
// methods and animation helpers for the meteor light-pulse effect.
func TestStaticJS_ConnectorGlowLayerDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "ConnectorGlowLayer = L.Layer.extend(") {
		t.Fatal("map-connector.js: ConnectorGlowLayer must be defined as a L.Layer extension")
	}
	for _, method := range []string{
		"initialize: function(",
		"onAdd: function(",
		"onRemove: function(",
		"_tick: function(",
		"_drawGlow: function(",
		"_posAtPx: function(",
		"_getScreenPts: function(",
	} {
		if !strings.Contains(mcJS, method) {
			t.Errorf("map-connector.js: ConnectorGlowLayer must define the %q method", method)
		}
	}
}

// TestStaticJS_ConnectorGlowLayerAnimation verifies that ConnectorGlowLayer
// uses requestAnimationFrame for the animation loop, cancels it in onRemove,
// and binds map move/zoom events to invalidate the cached screen projection.
func TestStaticJS_ConnectorGlowLayerAnimation(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	// Find the body of ConnectorGlowLayer.
	start := strings.Index(mcJS, "ConnectorGlowLayer = L.Layer.extend(")
	if start == -1 {
		t.Fatal("map-connector.js: ConnectorGlowLayer not found")
	}
	// Locate the end of the const declaration (next IIFE-level function).
	rest := mcJS[start:]
	endIdx := strings.Index(rest, "\n  function ")
	var layerBody string
	if endIdx != -1 {
		layerBody = rest[:endIdx]
	} else {
		end := start + 15000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		layerBody = mcJS[start:end]
	}

	if !strings.Contains(layerBody, "requestAnimationFrame(") {
		t.Error("map-connector.js: ConnectorGlowLayer must use requestAnimationFrame for the animation loop")
	}
	if !strings.Contains(layerBody, "cancelAnimationFrame(") {
		t.Error("map-connector.js: ConnectorGlowLayer.onRemove must call cancelAnimationFrame to stop the loop")
	}
	if !strings.Contains(layerBody, "clearRect(") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must call clearRect to erase the previous frame")
	}
	if !strings.Contains(layerBody, "createRadialGradient(") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must use createRadialGradient for the glow halo")
	}
	if !strings.Contains(layerBody, "lerpHex(") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must call lerpHex to interpolate head colour along the arc")
	}
}

// TestStaticJS_BuildConnectorLayerAddsGlowLayer verifies that
// buildConnectorLayer() instantiates ConnectorGlowLayer when the style config
// carries glowEnabled === true, adding the meteor animation on top of the base
// arc without coupling the two layers.
func TestStaticJS_BuildConnectorLayerAddsGlowLayer(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "glowEnabled") {
		t.Error("map-connector.js: buildConnectorLayer must check styleCfg.glowEnabled to decide whether to add the glow layer")
	}
	if !strings.Contains(fnBody, "ConnectorGlowLayer") {
		t.Error("map-connector.js: buildConnectorLayer must instantiate ConnectorGlowLayer when glowEnabled is true")
	}
	if !strings.Contains(fnBody, "CONNECTOR_GLOW_CONFIGS") {
		t.Error("map-connector.js: buildConnectorLayer must look up the glow config from CONNECTOR_GLOW_CONFIGS")
	}
	if !strings.Contains(fnBody, "group.addLayer(") {
		t.Error("map-connector.js: buildConnectorLayer must add ConnectorGlowLayer to the LayerGroup via addLayer()")
	}
}

// TestStaticJS_ConnectorGlowLayerExtinguish verifies that ConnectorGlowLayer
// implements the three-phase extinguish animation:
//   - Phase 1 (travel): masterAlpha = 1, progress ramps 0→1.
//   - Phase 2 (fade-out): progress fixed at 1, masterAlpha ramps 1→0, tail
//     converges back into the head (tailPx ∝ masterAlpha).
//   - Phase 3 (dark): canvas cleared, no drawing until next cycle.
//
// These invariants are verified by checking for the structural keywords that
// the three-phase _tick() and masterAlpha-aware _drawGlow() must contain.
func TestStaticJS_ConnectorGlowLayerExtinguish(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	layerStart := strings.Index(mcJS, "ConnectorGlowLayer = L.Layer.extend(")
	if layerStart == -1 {
		t.Fatal("map-connector.js: ConnectorGlowLayer not found")
	}
	// Capture enough of the layer body to cover all methods (~8 KB).
	end := layerStart + 15000
	if end > len(mcJS) {
		end = len(mcJS)
	}
	layerBody := mcJS[layerStart:end]

	// _tick must compute a three-phase cycle: travelMs + fadeMs + pauseMs.
	if !strings.Contains(layerBody, "fadeMs") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must read cfg.fadeMs to compute the three-phase cycle duration")
	}
	if !strings.Contains(layerBody, "masterAlpha") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must declare masterAlpha as a phase-dependent brightness multiplier")
	}
	// _drawGlow must accept and apply masterAlpha.
	if !strings.Contains(layerBody, "_drawGlow: function(ctx, progress, masterAlpha)") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must accept masterAlpha as its third parameter")
	}
	// Tail convergence: tailPx must be proportional to masterAlpha.
	if !strings.Contains(layerBody, "tailPx") || !strings.Contains(layerBody, "* masterAlpha") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must multiply tailPx by masterAlpha to converge the tail on fade-out")
	}
	// Phase 3: the dark phase must return early without calling _drawGlow.
	if !strings.Contains(layerBody, "return;") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must return early in phase 3 (dark) without drawing")
	}
}
