'use strict';
// ── map.js — Leaflet map core module ────────────────────────────────────────
// Extracted from app.js (sub-task 3.8).
// Exposes: PathProbe.Map = { renderMap, syncMapTileVariantToTheme, setMapTileVariant, refreshMapMarkers }
// All internal state, helpers, and sub-functions are private to this IIFE.
(() => {

  // ── Leaflet default-icon path override (embedded assets, no CDN) ──────────
  // leaflet.js is loaded before map.js (both with defer), so L is available.
  document.addEventListener('DOMContentLoaded', () => {
    if (typeof L !== 'undefined') {
      // Remove the auto-detection hook so Leaflet uses our explicit paths.
      delete L.Icon.Default.prototype._getIconUrl;
      L.Icon.Default.mergeOptions({
        iconUrl:       '/images/marker-icon.png',
        iconRetinaUrl: '/images/marker-icon-2x.png',
        shadowUrl:     '/images/marker-shadow.png',
      });
    }
  });

  // ── Private state ──────────────────────────────────────────────────────────
  // Active Leaflet map instance (one at a time).
  let _map = null;
  // Active tile layer attached to _map; kept separate so it can be swapped on
  // theme change without tearing down the whole map instance.
  let _tileLayer = null;
  // Currently selected map tile variant; null means "not yet set by user".
  // Initialised by syncMapTileVariantToTheme() when the app theme is applied.
  let _mapTileVariant = null;
  // Active marker style identifier.
  let _markerStyleId = 'diamond-pulse';
  // Active marker colour scheme ID.
  let _markerColorSchemeId = 'ocean';
  // Last rendered geo data; retained so refreshMapMarkers() can redraw markers
  // after a style change without a full map rebuild.
  let _lastPub   = null;
  let _lastTgt   = null;
  // Last rendered traceroute hop array; retained alongside _lastPub/_lastTgt so
  // that refreshConnectorLayer() can rebuild the route layer after colour-scheme
  // or style changes without requiring a full renderMap() call.
  let _lastRoute = null;
  // The live Leaflet legend control; stored so refreshMapMarkers() can remove
  // the old legend and add a fresh one that reflects the current colour scheme.
  let _legendControl = null;
  // Active connector arc style identifier.
  let _connectorStyleId = 'tick-xs';
  // Live Leaflet layer group for the connector arc between origin and target;
  // stored so refreshConnectorLayer() can remove the old layer before rebuilding.
  let _connectorLayer  = null;
  // Live Leaflet layer group for the multi-hop traceroute route path;
  // contains gradient polyline segments and circle markers for each geo hop.
  // Mutually exclusive with _connectorLayer: when route hops have geo data the
  // route layer is preferred and the simple arc is not drawn.
  let _routeLayer = null;
  // Last computed great-circle distance in km between origin and target;
  // retained so rerenderLabels() can refresh the distance badge text when the
  // locale changes without recomputing haversineKm().  Null when no two-point
  // map is currently displayed.
  let _lastDistanceKm = null;
  // Maps marker type ('origin' | 'target') to the live L.Marker instance so
  // refreshMapMarkers() can restore any open popup after rebuilding markers.
  let _markerByType = {};

  // ── Config aliases (resolved at parse time from PathProbe.Config) ──────────
  // config.js is loaded before map.js (see index.html).  The defensive guard
  // means this destructuring never throws even if config.js fails to load.
  const {
    MAP_POINT_CONFIGS             = {},
    MARKER_COLOR_SCHEME_CONFIGS   = {},
    MARKER_STYLE_CONFIGS          = {},
    CONNECTOR_LINE_CONFIGS        = {},
    MAP_TILE_VARIANTS             = [],
    MAP_THEME_TO_TILE_VARIANT     = {},
    TILE_LAYER_CONFIGS            = {},
    GEO_SAME_REGION_THRESHOLD_KM  = 200,
    DEFAULT_THEME = 'default',
  } = (window.PathProbe && window.PathProbe.Config) || {};

  // ── Private locale helper ──────────────────────────────────────────────────
  function t(key) {
    return window.PathProbe && window.PathProbe.Locale
      ? window.PathProbe.Locale.t(key)
      : key;
  }

  // ── Private HTML-escape helper ─────────────────────────────────────────────
  function esc(s) {
    return String(s)
      .replace(/&/g,  '&amp;')
      .replace(/</g,  '&lt;')
      .replace(/>/g,  '&gt;')
      .replace(/"/g,  '&quot;')
      .replace(/'/g,  '&#39;');
  }

  // ── Map tile helpers ───────────────────────────────────────────────────────

  /** Return the current map tile variant identifier.
   *  Falls back to the theme-derived default if _mapTileVariant is not set.
   */
  function getMapTileVariant() {
    if (_mapTileVariant && TILE_LAYER_CONFIGS[_mapTileVariant]) return _mapTileVariant;
    const theme = document.documentElement.dataset.theme || DEFAULT_THEME;
    return MAP_THEME_TO_TILE_VARIANT[theme] || 'light';
  }

  /**
   * Set the background colour of the #geo-map container to the representative
   * base colour of the target tile variant.  This ensures that un-loaded tile
   * gaps always show the expected colour — not white — immediately after a swap,
   * eliminating the white-flash artefact when switching to a dark tile set.
   * The colour values live exclusively in TILE_LAYER_CONFIGS (single source of
   * truth); this helper is the only consumer, keeping the logic cohesive.
   */
  function applyMapBgColor(container, variant) {
    if (!container) return;
    const cfg = TILE_LAYER_CONFIGS[variant];
    if (cfg && cfg.bgColor) container.style.background = cfg.bgColor;
  }

  /** Sync _mapTileVariant to the theme-derived default and update the map bar UI.
   *  Always performs a SILENT tile swap (no CSS fade animation) because this
   *  function is called either at page-load (map not yet created) or inside
   *  applyTheme() while the body is already opacity:0.  Animated tile changes
   *  are driven exclusively by setMapTileVariant() (user clicks a map-bar button).
   */
  function syncMapTileVariantToTheme(themeId) {
    const variant = MAP_THEME_TO_TILE_VARIANT[themeId] || 'light';
    _mapTileVariant = variant;
    updateMapBarButtons();
    if (!_map) return;
    const cfg = TILE_LAYER_CONFIGS[variant];
    if (!cfg) return;
    if (_tileLayer) { _tileLayer.remove(); _tileLayer = null; }
    _tileLayer = L.tileLayer(cfg.url, { attribution: cfg.attribution, maxZoom: 18 });
    _tileLayer.addTo(_map);
    // Apply the representative tile background colour so gaps during initial
    // tile-load show the correct base colour immediately.
    applyMapBgColor(document.getElementById('geo-map'), variant);
  }

  /** Update the active state of the three map-bar variant buttons. */
  function updateMapBarButtons() {
    document.querySelectorAll('.map-tile-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tileVariant === _mapTileVariant);
    });
  }

  /** Replace the tile layer on the live _map with a fade-out/fade-in transition.
   *  Called only from setMapTileVariant() (user-driven tile-variant change).
   *  No-op when no map instance exists yet.
   */
  function refreshMapTiles() {
    if (!_map) return;
    const container = document.getElementById('geo-map');
    const variant = getMapTileVariant();
    const cfg = TILE_LAYER_CONFIGS[variant];
    if (!cfg) return;

    if (container) container.classList.add('geo-map--fading');
    const doSwap = (e) => {
      // Guard: only act on this container's own opacity transition end.
      // Ignores events from child elements that bubble up, and ignores any
      // other CSS properties (e.g. border-color) that may also transition.
      if (e && (e.target !== container || e.propertyName !== 'opacity')) return;
      if (container) container.removeEventListener('transitionend', doSwap);
      if (_tileLayer) { _tileLayer.remove(); _tileLayer = null; }
      _tileLayer = L.tileLayer(cfg.url, { attribution: cfg.attribution, maxZoom: 18 });
      _tileLayer.addTo(_map);
      // Apply the target variant's base background colour BEFORE fading back in.
      // This ensures that any un-loaded tile gaps show the correct dark/light
      // colour rather than white during the fade-in, preventing the white-flash
      // artefact that appears when switching to the dark tile set.
      applyMapBgColor(container, variant);
      // Remove fading class on the next frame — CSS transition handles the
      // fade-back-in automatically; no second event listener is required.
      requestAnimationFrame(() => {
        if (container) container.classList.remove('geo-map--fading');
      });
    };
    if (container) {
      container.addEventListener('transitionend', doSwap);
    } else {
      doSwap(null);
    }
  }

  /** Set the active map tile variant explicitly (called by map-bar buttons).
   *  Updates button state and refreshes tiles with the fade animation.
   */
  function setMapTileVariant(variant) {
    if (!TILE_LAYER_CONFIGS[variant]) return;
    _mapTileVariant = variant;
    updateMapBarButtons();
    refreshMapTiles();
  }

  // ── Marker helpers ─────────────────────────────────────────────────────────

  /** Redraw only the Leaflet Marker layers using the current _markerStyleId.
   *  Preserves the tile layer; also removes and re-adds the legend so its icon
   *  reflects the new style, and rebuilds the connector arc layer.
   *  No-op when the map has not been initialised or no geo data is available.
   */
  function refreshMapMarkers() {
    if (!_map || (!_lastPub && !_lastTgt)) return;
    // Preserve open popup state before removing markers so that locale switches
    // (which call this function) do not dismiss any popup the user has open.
    const openPopupTypes = new Set();
    for (const [type, marker] of Object.entries(_markerByType)) {
      if (marker && marker.isPopupOpen()) openPopupTypes.add(type);
    }
    _markerByType = {};
    _map.eachLayer(layer => {
      if (layer instanceof L.Marker) _map.removeLayer(layer);
    });
    // Remove stale legend before rebuilding.
    if (_legendControl) { _legendControl.remove(); _legendControl = null; }
    const points = [];
    if (_lastPub && _lastPub.HasLocation) points.push({ geo: _lastPub, type: 'origin' });
    if (_lastTgt && _lastTgt.HasLocation) points.push({ geo: _lastTgt, type: 'target' });
    for (const p of points) {
      const m = L.marker([p.geo.Lat, p.geo.Lon], { icon: buildMarkerIcon(p.type) })
        .addTo(_map)
        .bindPopup(buildPopupHtml(p.geo, p.type));
      _markerByType[p.type] = m;
    }
    // Restore any popup that was open before the rebuild.
    for (const type of openPopupTypes) {
      if (_markerByType[type]) _markerByType[type].openPopup();
    }
    if (points.length > 1) {
      _legendControl = buildMapLegend(points.map(p => p.type));
      _legendControl.addTo(_map);
    }
    // Rebuild the gradient arc connector so it stays in sync with any
    // colour scheme or style changes that triggered this refresh.
    refreshConnectorLayer();
  }

  /** Apply the active colour scheme by writing --mc-origin / --mc-target onto
   *  the document root.  All diamond-marker CSS rules reference these two tokens
   *  so no DOM rebuild is needed when the scheme changes.
   */
  function applyMarkerColorScheme() {
    const scheme = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                 || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
    document.documentElement.style.setProperty('--mc-origin', scheme.originColor);
    document.documentElement.style.setProperty('--mc-target', scheme.targetColor);
  }

  /** Render the three map tile variant dot-buttons inside #geo-map-bar.
   *  Buttons are styled as coloured circles matching the header .theme-btn style
   *  — no visible text, accessible via aria-label and native title tooltip.
   *  Called once when renderMap() creates the map for the first time.
   */
  function renderMapBar() {
    const bar = document.getElementById('geo-map-bar');
    if (!bar) return;
    bar.innerHTML = MAP_TILE_VARIANTS.map(v => {
      const cfg = TILE_LAYER_CONFIGS[v];
      if (!cfg) return '';
      const isActive = (v === (_mapTileVariant || getMapTileVariant()));
      const label = esc(t(cfg.i18nKey));
      return '<button class="map-tile-btn' + (isActive ? ' active' : '') + '"' +
        ' data-tile-variant="' + esc(v) + '"' +
        ' onclick="setMapTileVariant(\'' + esc(v) + '\')"' +
        ' aria-label="' + label + '"' +
        ' title="' + label + '">' +
        '</button>';
    }).join('');
  }

  /** Calculate the great-circle distance in km between two lat/lon pairs
   *  using the Haversine formula.
   */
  function haversineKm(lat1, lon1, lat2, lon2) {
    const R = 6371;
    const toRad = deg => deg * Math.PI / 180;
    const dLat = toRad(lat2 - lat1);
    const dLon = toRad(lon2 - lon1);
    const a = Math.sin(dLat / 2) ** 2 +
      Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2;
    return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  }

  /** Create a custom Leaflet divIcon for the given point type ('origin'|'target').
   *  Visual config is read from MAP_POINT_CONFIGS, so no change is needed here
   *  when new roles are added.
   */
  function buildMarkerIcon(type) {
    const roleCfg  = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
    const styleCfg = MARKER_STYLE_CONFIGS[_markerStyleId] || MARKER_STYLE_CONFIGS.dot;
    return L.divIcon({
      className:   'geo-marker ' + roleCfg.cssClass,
      html:        styleCfg.buildHtml(roleCfg),
      iconSize:    styleCfg.iconSize,
      iconAnchor:  styleCfg.iconAnchor,
      popupAnchor: styleCfg.popupAnchor,
    });
  }

  /** Build an HTML string for a Leaflet popup from a GeoAnnotation object.
   *  All dynamic values are escaped via esc() to prevent XSS.
   */
  function buildPopupHtml(geo, type) {
    const cfg = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
    const lines = [
      '<div class="geo-popup">',
      '<span class="geo-popup__role geo-popup__role--' + type + '">' + esc(t(cfg.i18nKey)) + '</span>',
    ];
    if (geo.IP)          lines.push('<div class="geo-popup__ip">'  + esc(geo.IP)          + '</div>');
    if (geo.City)        lines.push('<div class="geo-popup__row">' + esc(geo.City)        + '</div>');
    if (geo.CountryName) lines.push('<div class="geo-popup__row">' + esc(geo.CountryName) + ' (' + esc(geo.CountryCode) + ')' + '</div>');
    if (geo.OrgName)     lines.push('<div class="geo-popup__asn">' + 'AS' + esc(String(geo.ASN)) + ' ' + esc(geo.OrgName) + '</div>');
    if (geo.LocationPrecision) {
      const precKey = 'geo-precision-' + geo.LocationPrecision;
      lines.push('<div class="geo-popup__precision">' + esc(t(precKey)) + '</div>');
    }
    lines.push('</div>');
    return lines.join('');
  }

  /** Build a Leaflet Control legend for the given set of point types.
   *  Accepts an array of role strings so only visible marker types are shown.
   *  The legend icon mirrors the active marker style via buildHtml() so it
   *  stays in sync when the picker changes the shape or colour scheme.
   */
  function buildMapLegend(pointTypes) {
    const legend = L.control({ position: 'bottomright' });
    legend.onAdd = function () {
      const div = L.DomUtil.create('div', 'geo-legend');
      const styleCfg = MARKER_STYLE_CONFIGS[_markerStyleId] || MARKER_STYLE_CONFIGS['diamond-pulse'];
      div.innerHTML = pointTypes.map(type => {
        const cfg = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
        return '<div class="geo-legend__item">' +
          '<span class="geo-marker ' + esc(cfg.cssClass) + ' geo-legend__marker">' +
          styleCfg.buildHtml(cfg) +
          '</span>' +
          '<span data-i18n="' + esc(cfg.i18nKey) + '">' + esc(t(cfg.i18nKey)) + '</span>' +
          '</div>';
      }).join('');
      return div;
    };
    return legend;
  }

  // ── Connector arc bridge — delegates to PathProbe.MapConnector ─────────────

  /** Returns true when the Leaflet map has been both created and initialised. */
  function isMapLoaded() {
    return Boolean(_map && _map._loaded);
  }

  /** Private wrapper — injects the module-level _map so call-sites inside this
   *  IIFE remain unchanged while the actual rendering lives in map-connector.js.
   */
  function buildConnectorLayer(pub, tgt, styleCfg, originColor, targetColor) {
    return (window.PathProbe && window.PathProbe.MapConnector)
      ? window.PathProbe.MapConnector.buildConnectorLayer(
          pub, tgt, styleCfg, originColor, targetColor, _map)
      : L.layerGroup();
  }

  /** Private wrapper — delegates multi-hop route rendering to map-connector.js.
   *  Passes the Leaflet map instance and the active connector style config
   *  explicitly so map-connector.js can render hop-to-hop segments using the
   *  same arc / arrow / glow style as the Public IP Detection map.
   */
  function buildRouteLayer(hops, originColor, targetColor) {
    const styleCfg = CONNECTOR_LINE_CONFIGS[_connectorStyleId]
                   || CONNECTOR_LINE_CONFIGS['dot-bead'];
    return (window.PathProbe && window.PathProbe.MapConnector)
      ? window.PathProbe.MapConnector.buildRouteLayer(
          hops, { originColor, targetColor }, styleCfg, _map)
      : L.layerGroup();
  }

  /** Show or update the #geo-route-info info card that explains the geo
   *  coverage of the current traceroute result.  Displayed when:
   *    • Some hops lack geo data (private IPs, timeouts, no GeoIP record), OR
   *    • Multiple hops cluster at the same geographic location (e.g. all hops
   *      within one country resolve to the same country centroid).
   *  Uses the same .route-stats-card / .route-stats-grid CSS as the route
   *  summary card so the visual language is consistent.
   *  Pass null/undefined to hide the card (non-traceroute modes).
   */
  function _renderRouteInfoCard(route) {
    const card = document.getElementById('geo-route-info');
    if (!card) return;
    if (!route || !route.length) { card.hidden = true; return; }

    const geoHops = route.filter(function(h) { return h.HasGeo && (h.Lat || h.Lon); });
    // Count distinct geographic locations using the same coordinate resolution
    // as _clusterHops() in map-connector.js (0.05° threshold ≈ 5.5 km).
    const locKey   = function(h) { return h.Lat.toFixed(2) + ',' + h.Lon.toFixed(2); };
    const clusters = (new Set(geoHops.map(locKey))).size;

    // Hide when every hop has a unique geo location — nothing extra to explain.
    if (geoHops.length === route.length && clusters === geoHops.length) {
      card.hidden = true;
      return;
    }

    // Update the card title using the current locale.
    const titleEl = card.querySelector('.route-stats-title');
    if (titleEl) titleEl.textContent = t('route-info-title');

    // Populate the stat grid with the same .route-stat-item pattern used by
    // renderRouteStats() in renderer.js so the two cards look identical.
    const grid = card.querySelector('.route-stats-grid');
    if (!grid) { card.hidden = true; return; }

    const items = [
      { label: t('route-info-hops'),       value: String(route.length)  },
      { label: t('route-info-geolocated'), value: String(geoHops.length) },
      { label: t('route-info-locations'),  value: String(clusters)       },
    ];
    grid.innerHTML = items.map(function(item) {
      return '<div class="route-stat-item">' +
        '<span class="route-stat-label">' + esc(item.label) + '</span>' +
        '<span class="route-stat-value">' + esc(item.value) + '</span>' +
        '</div>';
    }).join('');
    card.hidden = false;
  }

  /** Remove any existing connector or route layer and draw a fresh one.
   *
   *  Priority:
   *    1. Multi-hop route layer  — when _lastRoute contains hops with geo data.
   *       The route layer renders the actual traced path as gradient polyline
   *       segments + circle markers; the simple origin→target arc is NOT drawn
   *       because it would be misleading (the arc is a straight great-circle
   *       approximation, not the real network path).
   *    2. Simple arc fallback    — when no route geo hops are available but
   *       both origin and target have location data (existing behaviour).
   *
   *  No-op when the map has not been initialised.
   */
  function refreshConnectorLayer() {
    if (!isMapLoaded()) return;
    if (_connectorLayer) { _connectorLayer.remove(); _connectorLayer = null; }
    if (_routeLayer)     { _routeLayer.remove();     _routeLayer     = null; }

    const scheme = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                 || MARKER_COLOR_SCHEME_CONFIGS['ocean'];

    // ── Priority 1: multi-hop route layer ──────────────────────────────────
    const geoHops = (_lastRoute || []).filter(function(h) { return h.HasGeo; });
    if (geoHops.length > 0) {
      _routeLayer = buildRouteLayer(_lastRoute, scheme.originColor, scheme.targetColor);
      _routeLayer.addTo(_map);
      _renderRouteInfoCard(_lastRoute);  // explain geo coverage when hops cluster
      return;  // route shown — skip simple arc to avoid misleading overlay
    }

    // ── Priority 2: simple origin→target arc (existing behaviour) ──────────
    _renderRouteInfoCard(null);  // hide route info card in non-traceroute mode
    if (!_lastPub || !_lastTgt || !_lastPub.HasLocation || !_lastTgt.HasLocation) return;
    const styleCfg = CONNECTOR_LINE_CONFIGS[_connectorStyleId]
                   || CONNECTOR_LINE_CONFIGS['dot-bead'];
    _connectorLayer = buildConnectorLayer(_lastPub, _lastTgt, styleCfg,
                                          scheme.originColor, scheme.targetColor);
    _connectorLayer.addTo(_map);
  }
  // ── Distance badge ─────────────────────────────────────────────────────────

  /** Refresh #geo-distance text using the stored _lastDistanceKm value.
   *  Called on initial render and from rerenderLabels() so the 'map-distance'
   *  i18n key is always resolved in the current locale.  When no two-point map
   *  is displayed (_lastDistanceKm is null) the badge remains hidden.
   */
  function updateDistanceBadge() {
    const distEl = document.getElementById('geo-distance');
    if (!distEl) return;
    if (_lastDistanceKm === null) { distEl.hidden = true; return; }
    distEl.textContent = t('map-distance') + ': ' + Math.round(_lastDistanceKm).toLocaleString() + ' km';
    distEl.hidden = false;
  }

  // ── Main render function ───────────────────────────────────────────────────

  /** Render (or remove) the Leaflet map based on geo results.
   *
   * @param {object|null}  pub   PublicGeo annotation (client's public IP geo)
   * @param {object|null}  tgt   TargetGeo annotation (diagnostic target geo)
   * @param {Array|null}   route Optional array of traceroute HopEntry objects.
   *                             When supplied and at least one hop has HasGeo,
   *                             the map shows a multi-point route path instead
   *                             of the single origin→target arc.  The viewport
   *                             is fitted over all geo-located points (pub, tgt,
   *                             and route hops).  Passing null/undefined falls
   *                             back to the existing two-point arc behaviour.
   */
  function renderMap(pub, tgt, route) {
    // Retain all geo data so refreshMapMarkers() / refreshConnectorLayer() can
    // redraw after style changes without requiring a full rebuild.
    _lastPub   = pub;
    _lastTgt   = tgt;
    _lastRoute = route || null;

    const container = document.getElementById('geo-map');
    const distEl    = document.getElementById('geo-distance');
    if (!container || typeof L === 'undefined') return;

    // Reset distance badge on every render cycle.
    if (distEl) distEl.hidden = true;

    // Origin / target diamond markers (unchanged behaviour).
    const points = [];
    if (pub && pub.HasLocation) points.push({ geo: pub, type: 'origin' });
    if (tgt && tgt.HasLocation) points.push({ geo: tgt, type: 'target' });

    // Route hops with valid geo coordinates (new).
    const geoHops = (_lastRoute || []).filter(function(h) { return h.HasGeo && (h.Lat || h.Lon); });

    // All geo-located points used to calculate the map viewport.
    const boundsLatLngs = [
      ...points.map(function(p) { return [p.geo.Lat, p.geo.Lon]; }),
      ...geoHops.map(function(h) { return [h.Lat, h.Lon]; }),
    ];

    // Hide the map when no geo data is available anywhere.
    if (boundsLatLngs.length === 0) {
      container.classList.remove('visible');
      const outerEl = document.getElementById('geo-map-outer');
      if (outerEl) outerEl.hidden = true;
      if (_map) { _map.remove(); _map = null; _tileLayer = null; _connectorLayer = null; _routeLayer = null; }
      _lastDistanceKm = null;
      _markerByType = {};
      _renderRouteInfoCard(null);  // clear info card when map is hidden
      return;
    }

    // Reveal the outer wrapper (bar + map) before showing the map itself.
    const outerEl = document.getElementById('geo-map-outer');
    if (outerEl) outerEl.hidden = false;
    container.classList.add('visible');

    // Render tile-variant bar and apply the colour scheme.
    renderMapBar();
    applyMarkerColorScheme();

    // Destroy any existing map instance before creating a new one.
    if (_map) { _map.remove(); _map = null; _tileLayer = null; _connectorLayer = null; }

    _map = L.map('geo-map');
    // Tile layer is driven by the current application theme via TILE_LAYER_CONFIGS.
    const tileCfg = TILE_LAYER_CONFIGS[getMapTileVariant()];
    _tileLayer = L.tileLayer(tileCfg.url, {
      attribution: tileCfg.attribution,
      maxZoom: 18,
    }).addTo(_map);

    // ── Origin / target diamond markers ──────────────────────────────────────
    const markerLatLngs = [];
    for (const p of points) {
      const m = L.marker([p.geo.Lat, p.geo.Lon], { icon: buildMarkerIcon(p.type) })
        .addTo(_map)
        .bindPopup(buildPopupHtml(p.geo, p.type));
      _markerByType[p.type] = m;
      markerLatLngs.push([p.geo.Lat, p.geo.Lon]);
    }

    // ── Viewport setup ────────────────────────────────────────────────────────
    // MUST be done BEFORE building the connector/route layer because Leaflet
    // functions such as latLngToLayerPoint() require the map to be fully
    // initialised (throws "Set map center and zoom first." otherwise).
    if (boundsLatLngs.length === 1) {
      // Single geo point: zoom level depends on coordinate precision.
      const precision = (points.length > 0 ? points[0].geo.LocationPrecision : null)
                      || (geoHops.length  > 0 ? '' : '');
      const zoom = (precision === 'city') ? 8 : 5;
      _map.setView(boundsLatLngs[0], zoom);
    } else {
      // Multiple geo points: fit the viewport over all of them.
      // Distance badge and proximity guard only apply when both pub and tgt
      // have location data (existing two-point behaviour; unchanged).
      if (markerLatLngs.length === 2) {
        const km = haversineKm(
          markerLatLngs[0][0], markerLatLngs[0][1],
          markerLatLngs[1][0], markerLatLngs[1][1]);
        _lastDistanceKm = km;
        const hasCountryPrecision = points.some(p => p.geo.LocationPrecision === 'country');
        // Proximity guard: prevents over-zoom when two country centroids coincide.
        if (hasCountryPrecision && km < GEO_SAME_REGION_THRESHOLD_KM) {
          const midLat = (markerLatLngs[0][0] + markerLatLngs[1][0]) / 2;
          const midLon = (markerLatLngs[0][1] + markerLatLngs[1][1]) / 2;
          _map.setView([midLat, midLon], 5);
        } else {
          _map.fitBounds(boundsLatLngs, { padding: [40, 40] });
        }
        updateDistanceBadge();
      } else {
        // Route hops only (or mixed without both pub+tgt): plain fitBounds.
        _map.fitBounds(boundsLatLngs, { padding: [40, 40] });
      }

      // Draw connector arc OR multi-hop route layer via refreshConnectorLayer().
      // refreshConnectorLayer() selects the route layer when _lastRoute has geo
      // hops; otherwise falls back to the simple origin→target arc.
      refreshConnectorLayer();
    }

    // Add a legend when multiple origin/target marker types are present so users
    // can distinguish 偵測起點 from 偵測目標.
    if (points.length > 1) {
      _legendControl = buildMapLegend(points.map(p => p.type));
      _legendControl.addTo(_map);
    }

    // Leaflet cannot correctly position tiles when the container (or an ancestor)
    // has display:none at initialisation time.  The #results section is hidden
    // before results arrive, so the first renderMap() call sees a 0×0 layout.
    // Scheduling invalidateSize() for the next frame (after the browser has
    // applied the visibility change and reflowed the layout) ensures all tiles
    // are projected to the correct positions and eliminates the blank grey areas.
    requestAnimationFrame(() => {
      if (_map) _map.invalidateSize();
    });
  }

  // ── Locale-driven label refresh ─────────────────────────────────────────────

  /** Re-apply all i18n-dependent map labels using the current locale.
   *  Called by locale.js / applyLocale() after every locale switch so that:
   *    • tile-bar button aria-label / title attributes
   *    • Leaflet marker popup HTML (stored inside Leaflet, not in live DOM)
   *    • map legend text (has data-i18n but is in Leaflet's control layer)
   *    • distance badge composite text (key + numeric value + unit)
   *  are all translated to the new locale without requiring a full map rebuild.
   *  Safe to call when no map is displayed — each private sub-call guards on
   *  state availability (null-checks on _map, _lastPub, _lastTgt, _lastDistanceKm).
   */
  function rerenderLabels() {
    renderMapBar();
    refreshMapMarkers();
    updateDistanceBadge();
    _renderRouteInfoCard(_lastRoute);  // refresh i18n text in route info card
  }

  // ── Global bridge (HTML onclick buttons need window.setMapTileVariant) ──────
  window.setMapTileVariant = setMapTileVariant;

  // ── Export ─────────────────────────────────────────────────────────────────
  const _ns = window.PathProbe || {};
  _ns.Map = { renderMap, syncMapTileVariantToTheme, setMapTileVariant, refreshMapMarkers, rerenderLabels };
  window.PathProbe = _ns;
})();
