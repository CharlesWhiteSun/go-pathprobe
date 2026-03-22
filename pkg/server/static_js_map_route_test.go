package server_test

import (
	"strings"
	"testing"
)

// ── Traceroute multi-point route map feature ───────────────────────────────────
//
// These tests verify that the route-layer pipeline is fully wired:
//
//  map-connector.js exports buildRouteLayer() which builds a layerGroup of
//    gradient polyline segments + circle markers for each geo-located hop.
//  map.js stores _lastRoute state, extends renderMap() to accept a third
//    'route' parameter, and defers to buildRouteLayer() via buildConnectorLayer()
//    when route hops with geo data are available.
//  api-client.js forwards payload.Route to renderMap() so the map reflects the
//    actual traced network path rather than a straight origin→target arc.
//  style.css provides .geo-popup__role--hop for intermediate hop popups.

// TestStaticJS_MapConnector_ExportsRouteLayer verifies that map-connector.js
// exports the buildRouteLayer function alongside the existing buildConnectorLayer.
func TestStaticJS_MapConnector_ExportsRouteLayer(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	// Declaration must be present.
	if !strings.Contains(body, "function buildRouteLayer(") {
		t.Error("map-connector.js: buildRouteLayer function must be defined")
	}
	// Must be exported via PathProbe.MapConnector.
	exportIdx := strings.Index(body, "_ns.MapConnector = {")
	if exportIdx == -1 {
		t.Fatal("map-connector.js: PathProbe.MapConnector export not found")
	}
	end := exportIdx + 300
	if end > len(body) {
		end = len(body)
	}
	exportBlock := body[exportIdx:end]
	if !strings.Contains(exportBlock, "buildRouteLayer") {
		t.Error("map-connector.js: buildRouteLayer must appear in the PathProbe.MapConnector export object")
	}
}

// TestStaticJS_MapConnector_RouteLayerFiltersGeoHops verifies that
// buildRouteLayer() explicitly filters hops by the HasGeo field, so hops
// without geo (timed-out, private IPs, unresolved) are silently skipped and
// never cause errors.
func TestStaticJS_MapConnector_RouteLayerFiltersGeoHops(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildRouteLayer not found")
	}
	// Find end of function by locating the next sibling function.
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
	if !strings.Contains(fnBody, "HasGeo") {
		t.Error("map-connector.js: buildRouteLayer must filter hops by the HasGeo field")
	}
	// Must short-circuit when no geo hops remain.
	if !strings.Contains(fnBody, "geoHops.length === 0") {
		t.Error("map-connector.js: buildRouteLayer must return an empty layerGroup when no geo hops are present")
	}
}

// TestStaticJS_MapConnector_RouteLayerUsesPolyline verifies that
// buildRouteLayer() uses L.polyline to connect consecutive geo hops, giving
// the user a line that follows the actual network route order.
func TestStaticJS_MapConnector_RouteLayerUsesPolyline(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildRouteLayer not found")
	}
	end := fnStart + 3000
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	if !strings.Contains(fnBody, "L.polyline(") {
		t.Error("map-connector.js: buildRouteLayer must use L.polyline() to connect consecutive hops")
	}
}

// TestStaticJS_MapConnector_RouteLayerUsesHopMarker verifies that
// buildRouteLayer() places a diamond-shaped L.marker at each geo cluster,
// using _buildHopMarkerIcon() to match the visual style of the origin/target
// markers rather than plain circle markers.
func TestStaticJS_MapConnector_RouteLayerUsesHopMarker(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildRouteLayer not found")
	}
	end := fnStart + 4000
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	// Must use the hop diamond icon builder (not a plain circle marker).
	if !strings.Contains(fnBody, "_buildHopMarkerIcon(") {
		t.Error("map-connector.js: buildRouteLayer must use _buildHopMarkerIcon() for intermediate hop markers (diamond style consistent with origin/target)")
	}
	// The icon builder itself must reference the hop CSS classes.
	if !strings.Contains(body, "geo-marker__hop-core") {
		t.Error("map-connector.js: _buildHopMarkerIcon must apply the 'geo-marker__hop-core' CSS class for the diamond shape")
	}
}

// TestStaticJS_MapConnector_RouteLayerGradient verifies that buildRouteLayer()
// applies a colour gradient across the hops by calling lerpHex() so the visual
// language matches the existing gradient arc on non-traceroute maps.
func TestStaticJS_MapConnector_RouteLayerGradient(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildRouteLayer not found")
	}
	end := fnStart + 3000
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	if !strings.Contains(fnBody, "lerpHex(") {
		t.Error("map-connector.js: buildRouteLayer must call lerpHex() to gradient-colour hop segments and markers")
	}
}

// TestStaticJS_MapConnector_HopPopupHtml verifies that buildRouteLayer()
// builds a popup for each hop that includes TTL info and reuses the .geo-popup
// CSS classes for consistent visual styling.
func TestStaticJS_MapConnector_HopPopupHtml(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	if !strings.Contains(body, "_buildHopPopupHtml(") {
		t.Error("map-connector.js: _buildHopPopupHtml helper must be defined for hop marker popups")
	}
	// Must include the TTL value in the popup role badge.
	if !strings.Contains(body, "geo-popup__role--hop") {
		t.Error("map-connector.js: _buildHopPopupHtml must apply the 'geo-popup__role--hop' CSS class")
	}
	// Must include TTL field.
	if !strings.Contains(body, "hop.TTL") {
		t.Error("map-connector.js: _buildHopPopupHtml must render the hop's TTL number")
	}
}

// TestStaticJS_Map_HasRouteState verifies that map.js maintains private
// _lastRoute and _routeLayer state variables to support rebuilding the route
// layer when the colour scheme or marker style changes.
func TestStaticJS_Map_HasRouteState(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "_lastRoute") {
		t.Error("map.js: _lastRoute state variable must be declared to retain route data for refreshes")
	}
	if !strings.Contains(body, "_routeLayer") {
		t.Error("map.js: _routeLayer state variable must be declared to manage the route LayerGroup lifecycle")
	}
}

// TestStaticJS_Map_BuildRouteLayerWrapper verifies that map.js defines a
// private buildRouteLayer() wrapper that delegates to
// PathProbe.MapConnector.buildRouteLayer, following the same pattern as the
// existing buildConnectorLayer() wrapper.
func TestStaticJS_Map_BuildRouteLayerWrapper(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function buildRouteLayer(") {
		t.Error("map.js: private buildRouteLayer() wrapper must be defined")
	}
	if !strings.Contains(body, "MapConnector.buildRouteLayer(") {
		t.Error("map.js: buildRouteLayer() wrapper must delegate to PathProbe.MapConnector.buildRouteLayer")
	}
}

// TestStaticJS_Map_RenderMapAcceptsRoute verifies that renderMap() accepts a
// third 'route' parameter and stores it as _lastRoute, enabling the route layer
// to be rebuilt by refreshConnectorLayer() on style/colour changes.
func TestStaticJS_Map_RenderMapAcceptsRoute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	end := fnStart + 150
	if end > len(body) {
		end = len(body)
	}
	signature := body[fnStart:end]
	if !strings.Contains(signature, "route") {
		t.Error("map.js: renderMap() must accept a 'route' parameter as its third argument")
	}
}

// TestStaticJS_Map_RefreshConnectorPrefersRoute verifies that
// refreshConnectorLayer() checks for route hops with geo data and builds a
// route layer (via buildRouteLayer) rather than the simple arc when they are
// present.  This is the core logic that makes the traceroute map display the
// actual network path instead of an origin→target approximation.
func TestStaticJS_Map_RefreshConnectorPrefersRoute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function refreshConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map.js: refreshConnectorLayer function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	// Must filter _lastRoute by HasGeo.
	if !strings.Contains(fnBody, "HasGeo") {
		t.Error("map.js: refreshConnectorLayer must filter _lastRoute hops by HasGeo before deciding route vs arc")
	}
	// When route hops exist, must call buildRouteLayer.
	if !strings.Contains(fnBody, "buildRouteLayer(") {
		t.Error("map.js: refreshConnectorLayer must call buildRouteLayer() when geo hops are available")
	}
	// Must store result in _routeLayer.
	if !strings.Contains(fnBody, "_routeLayer =") {
		t.Error("map.js: refreshConnectorLayer must store the route LayerGroup in _routeLayer")
	}
}

// TestStaticJS_ApiClient_PassesRouteTorenderMap verifies that api-client.js
// forwards payload.Route as the third argument to renderMap() so that
// traceroute hop geo data reaches the map rendering pipeline.
func TestStaticJS_ApiClient_PassesRouteTorenderMap(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	if !strings.Contains(body, "renderMap(payload.PublicGeo, payload.TargetGeo, payload.Route)") {
		t.Error("api-client.js: renderMap must be called with payload.Route as the third argument to enable traceroute route layer rendering")
	}
}

// TestStaticCSS_HopMarkerRole verifies that style.css defines the
// .geo-popup__role--hop CSS rule used by intermediate hop popups.
// Without this rule the hop badge falls back to unstyled (no background colour).
func TestStaticCSS_HopMarkerRole(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, ".geo-popup__role--hop") {
		t.Error("style.css: .geo-popup__role--hop rule must be defined for intermediate hop popup role badges")
	}
}

// ── New tests for route-map style consistency, clustering and info card ────────

// TestStaticJS_MapConnector_RouteClusters verifies that buildRouteLayer()
// groups geographically co-located hops into clusters so that ISP backbone
// IPs which all resolve to the same country centroid are shown as a single
// diamond marker instead of many overlapping invisible markers.
func TestStaticJS_MapConnector_RouteClusters(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	if !strings.Contains(body, "function _clusterHops(") {
		t.Error("map-connector.js: _clusterHops helper must be defined to group co-located hops")
	}
	if !strings.Contains(body, "GEO_HOP_CLUSTER_THRESHOLD_DEG") {
		t.Error("map-connector.js: GEO_HOP_CLUSTER_THRESHOLD_DEG constant must be defined for the clustering radius")
	}
}

// TestStaticJS_MapConnector_RouteClusterPopup verifies that a dedicated popup
// builder exists for multi-hop clusters and uses the shared .geo-popup CSS class
// for visual consistency.
func TestStaticJS_MapConnector_RouteClusterPopup(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	if !strings.Contains(body, "function _buildClusterPopupHtml(") {
		t.Error("map-connector.js: _buildClusterPopupHtml must be defined for cluster marker popups")
	}
	if !strings.Contains(body, "geo-popup__cluster-hop") {
		t.Error("map-connector.js: _buildClusterPopupHtml must use 'geo-popup__cluster-hop' CSS class for each co-located hop row")
	}
}

// TestStaticJS_MapConnector_RouteUsesConnectorLayer verifies that
// buildRouteLayer() delegates each hop-cluster-to-cluster segment to
// buildConnectorLayer() when a styleCfg is provided, so the arc style,
// arrow symbols, and glow animation match the "Public IP Detection" map.
func TestStaticJS_MapConnector_RouteUsesConnectorLayer(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildRouteLayer not found")
	}
	end := fnStart + 4000
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	if !strings.Contains(fnBody, "buildConnectorLayer(") {
		t.Error("map-connector.js: buildRouteLayer must call buildConnectorLayer() when styleCfg is provided to match the Public IP Detection visual style")
	}
}

// TestStaticJS_MapConnector_RouteStyleCfgParam verifies that buildRouteLayer()
// accepts a styleCfg parameter (3rd positional argument before mapInstance).
func TestStaticJS_MapConnector_RouteStyleCfgParam(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map-connector.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildRouteLayer not found")
	}
	end := fnStart + 80
	if end > len(body) {
		end = len(body)
	}
	sig := body[fnStart:end]
	if !strings.Contains(sig, "styleCfg") {
		t.Error("map-connector.js: buildRouteLayer must accept a 'styleCfg' parameter for connector style config")
	}
}

// TestStaticJS_Map_BuildRoutePassesStyleCfg verifies that the map.js
// buildRouteLayer() wrapper reads CONNECTOR_LINE_CONFIGS and passes it as
// styleCfg to MapConnector.buildRouteLayer so hop-segment connectors use the
// same style the user selected for the Public IP Detection arc.
func TestStaticJS_Map_BuildRoutePassesStyleCfg(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildRouteLayer(")
	if fnStart == -1 {
		t.Fatal("map.js: private buildRouteLayer wrapper not found")
	}
	end := fnStart + 400
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	if !strings.Contains(fnBody, "CONNECTOR_LINE_CONFIGS") {
		t.Error("map.js: buildRouteLayer wrapper must read CONNECTOR_LINE_CONFIGS to resolve the active style config")
	}
	if !strings.Contains(fnBody, "styleCfg") {
		t.Error("map.js: buildRouteLayer wrapper must pass styleCfg to MapConnector.buildRouteLayer")
	}
}

// TestStaticJS_Map_RenderRouteInfoCard verifies that map.js defines
// _renderRouteInfoCard which populates #geo-route-info using the
// route-stats-card / route-stats-grid visual pattern.
func TestStaticJS_Map_RenderRouteInfoCard(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function _renderRouteInfoCard(") {
		t.Error("map.js: _renderRouteInfoCard must be defined to show/hide the route info card")
	}
	if !strings.Contains(body, "'geo-route-info'") {
		t.Error("map.js: _renderRouteInfoCard must reference the '#geo-route-info' element by id")
	}
	if !strings.Contains(body, "card.hidden = true") {
		t.Error("map.js: _renderRouteInfoCard must hide the card when no route data is available")
	}
	// Must use the shared stat-item pattern for visual consistency.
	if !strings.Contains(body, "route-stat-item") {
		t.Error("map.js: _renderRouteInfoCard must generate .route-stat-item elements to match .route-stats-card visual style")
	}
	if !strings.Contains(body, "route-stats-title") {
		t.Error("map.js: _renderRouteInfoCard must update the .route-stats-title element with the i18n card title")
	}
}

// TestStaticHTML_RouteInfoCard verifies that index.html contains the
// #geo-route-info element structured as a .route-stats-card so that its
// visual style matches the Route Summary card above it.
func TestStaticHTML_RouteInfoCard(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, "geo-route-info") {
		t.Error("index.html: #geo-route-info element must be present for the route info card")
	}
	// Must use the same route-stats-card / route-stats-grid CSS as the route
	// summary card for visual consistency.
	if !strings.Contains(body, "route-stats-grid") {
		t.Error("index.html: #geo-route-info must contain a .route-stats-grid div for the stat items")
	}
	if !strings.Contains(body, "route-stats-title") {
		t.Error("index.html: #geo-route-info must contain a .route-stats-title element for the card heading")
	}
}

// TestStaticCSS_HopMarkerCore verifies that style.css defines
// .geo-marker__hop-core (diamond shape for intermediate hop markers) and that
// the legacy .route-info-card rule is absent — #geo-route-info now uses
// .route-stats-card so the two cards share the same visual language.
func TestStaticCSS_HopMarkerCore(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, ".geo-marker__hop-core") {
		t.Error("style.css: .geo-marker__hop-core rule must be defined for the diamond shape of intermediate hop markers")
	}
	// The #geo-route-info element now reuses .route-stats-card; the old
	// dedicated .route-info-card rule should no longer exist.
	if strings.Contains(body, ".route-info-card") {
		t.Error("style.css: legacy .route-info-card rule must be removed; #geo-route-info now uses .route-stats-card for style consistency")
	}
}

// TestStaticI18n_RouteInfoKeys verifies that i18n.js defines all four keys
// used by _renderRouteInfoCard() in both EN and ZH locales.
func TestStaticI18n_RouteInfoKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	for _, key := range []string{"route-info-title", "route-info-hops", "route-info-geolocated", "route-info-locations"} {
		if !strings.Contains(body, "'"+key+"'") {
			t.Errorf("i18n.js: key '%s' must be defined in both EN and ZH locales for the route info card", key)
		}
	}
}
