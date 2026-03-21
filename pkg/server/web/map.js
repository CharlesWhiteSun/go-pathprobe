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
  let _lastPub = null;
  let _lastTgt = null;
  // The live Leaflet legend control; stored so refreshMapMarkers() can remove
  // the old legend and add a fresh one that reflects the current colour scheme.
  let _legendControl = null;
  // Active connector arc style identifier.
  let _connectorStyleId = 'tick-xs';
  // Live Leaflet layer group for the connector arc between origin and target;
  // stored so refreshConnectorLayer() can remove the old layer before rebuilding.
  let _connectorLayer  = null;
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

  /** Remove any existing connector arc layer and draw a fresh one using the
   *  current _connectorStyleId and colour scheme.  No-op when the map has
   *  not been initialised or geo data for both endpoints is unavailable.
   */
  function refreshConnectorLayer() {
    if (!isMapLoaded() || !_lastPub || !_lastTgt) return;
    if (_connectorLayer) { _connectorLayer.remove(); _connectorLayer = null; }
    if (!_lastPub.HasLocation || !_lastTgt.HasLocation) return;
    const scheme   = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                   || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
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
   *  pub / tgt are GeoAnnotation objects that may be null/undefined.
   */
  function renderMap(pub, tgt) {
    // Retain geo data so refreshMapMarkers() can redraw without a full rebuild.
    _lastPub = pub;
    _lastTgt = tgt;

    const container = document.getElementById('geo-map');
    const distEl    = document.getElementById('geo-distance');
    if (!container || typeof L === 'undefined') return;

    // Reset distance badge on every render cycle.
    if (distEl) distEl.hidden = true;

    const points = [];
    if (pub && pub.HasLocation) {
      points.push({ geo: pub, type: 'origin' });
    }
    if (tgt && tgt.HasLocation) {
      points.push({ geo: tgt, type: 'target' });
    }

    // Hide the map when there are no geo-located points.
    if (points.length === 0) {
      container.classList.remove('visible');
      const outerEl = document.getElementById('geo-map-outer');
      if (outerEl) outerEl.hidden = true;
      if (_map) { _map.remove(); _map = null; _tileLayer = null; _connectorLayer = null; }
      _lastDistanceKm = null;
      _markerByType = {};
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

    const latLngs = [];
    for (const p of points) {
      const m = L.marker([p.geo.Lat, p.geo.Lon], { icon: buildMarkerIcon(p.type) })
        .addTo(_map)
        .bindPopup(buildPopupHtml(p.geo, p.type));
      _markerByType[p.type] = m;
      latLngs.push([p.geo.Lat, p.geo.Lon]);
    }

    // Set the map viewport BEFORE building the connector because functions such
    // as latLngToLayerPoint() and the dash-offset calculation require the map to
    // be fully initialised (Leaflet throws "Set map center and zoom first." if
    // called before setView / fitBounds).
    if (latLngs.length === 1) {
      // Use a lower zoom for country-level coordinates (country centroid) so
      // the surrounding context is visible; city-level coordinates can zoom in further.
      const precision = points[0].geo.LocationPrecision;
      const zoom = (precision === 'city') ? 8 : 5;
      _map.setView(latLngs[0], zoom);
    } else {
      // Pre-compute the great-circle distance so it drives both the viewport
      // proximity guard and the distance badge without a second haversineKm() call.
      const km = haversineKm(latLngs[0][0], latLngs[0][1], latLngs[1][0], latLngs[1][1]);
      _lastDistanceKm = km;
      const hasCountryPrecision = points.some(p => p.geo.LocationPrecision === 'country');
      // Proximity guard: fitBounds() over-zooms when country centroids coincide
      // or are very close (e.g. both points map to the same national centroid).
      // A fixed zoom on the midpoint keeps geographic context visible.
      if (hasCountryPrecision && km < GEO_SAME_REGION_THRESHOLD_KM) {
        const midLat = (latLngs[0][0] + latLngs[1][0]) / 2;
        const midLon = (latLngs[0][1] + latLngs[1][1]) / 2;
        _map.setView([midLat, midLon], 5);
      } else {
        _map.fitBounds(latLngs, { padding: [40, 40] });
      }

      // Draw the gradient arc connector and show the distance badge.
      // Connector style and colour scheme come from module-level state so
      // switching pickers refreshes the arc without a full map rebuild.
      const arcScheme   = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                        || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
      const arcStyleCfg = CONNECTOR_LINE_CONFIGS[_connectorStyleId]
                        || CONNECTOR_LINE_CONFIGS['dot-bead'];
      _connectorLayer   = buildConnectorLayer(
        points[0].geo, points[1].geo,
        arcStyleCfg, arcScheme.originColor, arcScheme.targetColor
      );
      _connectorLayer.addTo(_map);
      updateDistanceBadge();
    }

    // Add a legend when multiple marker types are present so users can
    // distinguish origin (偵測起點) from target (偵測目標).
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
  }

  // ── Global bridge (HTML onclick buttons need window.setMapTileVariant) ──────
  window.setMapTileVariant = setMapTileVariant;

  // ── Export ─────────────────────────────────────────────────────────────────
  const _ns = window.PathProbe || {};
  _ns.Map = { renderMap, syncMapTileVariantToTheme, setMapTileVariant, refreshMapMarkers, rerenderLabels };
  window.PathProbe = _ns;
})();
