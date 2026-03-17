'use strict';
// ── map.js — Leaflet map core: init, markers, tiles, legend (PathProbe.Map)
// Depends on: config.js, locale.js, map-connector.js
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

// ── Module state ──────────────────────────────────────────────────────────
let _map              = null;   // active Leaflet map instance
let _tileLayer        = null;   // active tile layer
let _mapTileVariant   = null;   // 'light' | 'osm' | 'dark' | null
let _markerStyleId    = 'diamond-pulse';
let _markerColorSchemeId = 'ocean';
let _lastPub          = null;   // last rendered PublicGeo
let _lastTgt          = null;   // last rendered TargetGeo
let _legendControl    = null;
let _connectorLayer   = null;
let _connectorStyleId = 'tick-xs';

function t(key) { return PathProbe.Locale.t(key); }

function esc(s) {
  return String(s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

// ── Tile helpers ──────────────────────────────────────────────────────────

function getMapTileVariant() {
  const { TILE_LAYER_CONFIGS, MAP_THEME_TO_TILE_VARIANT, DEFAULT_THEME } = PathProbe.Config;
  if (_mapTileVariant && TILE_LAYER_CONFIGS[_mapTileVariant]) return _mapTileVariant;
  const theme = document.documentElement.dataset.theme || DEFAULT_THEME;
  return MAP_THEME_TO_TILE_VARIANT[theme] || 'light';
}

function applyMapBgColor(container, variant) {
  if (!container) return;
  const cfg = PathProbe.Config.TILE_LAYER_CONFIGS[variant];
  if (cfg && cfg.bgColor) container.style.background = cfg.bgColor;
}

function updateMapBarButtons() {
  document.querySelectorAll('.map-tile-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.tileVariant === _mapTileVariant);
  });
}

function syncMapTileVariantToTheme(themeId) {
  const { MAP_THEME_TO_TILE_VARIANT, TILE_LAYER_CONFIGS } = PathProbe.Config;
  const variant = MAP_THEME_TO_TILE_VARIANT[themeId] || 'light';
  _mapTileVariant = variant;
  updateMapBarButtons();
  if (!_map) return;
  const cfg = TILE_LAYER_CONFIGS[variant];
  if (!cfg) return;
  if (_tileLayer) { _tileLayer.remove(); _tileLayer = null; }
  _tileLayer = L.tileLayer(cfg.url, { attribution: cfg.attribution, maxZoom: 18 });
  _tileLayer.addTo(_map);
  applyMapBgColor(document.getElementById('geo-map'), variant);
}

function refreshMapTiles() {
  if (!_map) return;
  const { TILE_LAYER_CONFIGS } = PathProbe.Config;
  const container = document.getElementById('geo-map');
  const variant   = getMapTileVariant();
  const cfg       = TILE_LAYER_CONFIGS[variant];
  if (!cfg) return;
  if (container) container.classList.add('geo-map--fading');
  const doSwap = (e) => {
    if (e && (e.target !== container || e.propertyName !== 'opacity')) return;
    if (container) container.removeEventListener('transitionend', doSwap);
    if (_tileLayer) { _tileLayer.remove(); _tileLayer = null; }
    _tileLayer = L.tileLayer(cfg.url, { attribution: cfg.attribution, maxZoom: 18 });
    _tileLayer.addTo(_map);
    applyMapBgColor(container, variant);
    requestAnimationFrame(() => {
      if (container) container.classList.remove('geo-map--fading');
    });
  };
  if (container) { container.addEventListener('transitionend', doSwap); }
  else           { doSwap(null); }
}

function setMapTileVariant(variant) {
  if (!PathProbe.Config.TILE_LAYER_CONFIGS[variant]) return;
  _mapTileVariant = variant;
  updateMapBarButtons();
  refreshMapTiles();
}

function renderMapBar() {
  const { MAP_TILE_VARIANTS, TILE_LAYER_CONFIGS } = PathProbe.Config;
  const bar = document.getElementById('geo-map-bar');
  if (!bar) return;
  bar.innerHTML = MAP_TILE_VARIANTS.map(v => {
    const cfg = TILE_LAYER_CONFIGS[v];
    if (!cfg) return '';
    const isActive = (v === (_mapTileVariant || getMapTileVariant()));
    const label    = esc(t(cfg.i18nKey));
    return '<button class="map-tile-btn' + (isActive ? ' active' : '') + '"' +
      ' data-tile-variant="' + esc(v) + '"' +
      ' onclick="setMapTileVariant(\'' + esc(v) + '\')"' +
      ' aria-label="' + label + '" title="' + label + '">' +
      '</button>';
  }).join('');
}

// ── Map state guards ──────────────────────────────────────────────────────

function isMapLoaded() {
  return Boolean(_map && _map._loaded);
}

// ── Marker helpers ────────────────────────────────────────────────────────

function applyMarkerColorScheme() {
  const { MARKER_COLOR_SCHEME_CONFIGS } = PathProbe.Config;
  const scheme = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
               || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
  document.documentElement.style.setProperty('--mc-origin', scheme.originColor);
  document.documentElement.style.setProperty('--mc-target', scheme.targetColor);
}

function buildMarkerIcon(type) {
  const { MAP_POINT_CONFIGS, MARKER_STYLE_CONFIGS } = PathProbe.Config;
  const roleCfg  = MAP_POINT_CONFIGS[type]  || MAP_POINT_CONFIGS.target;
  const styleCfg = MARKER_STYLE_CONFIGS[_markerStyleId] || MARKER_STYLE_CONFIGS['diamond-pulse'];
  return L.divIcon({
    className:   'geo-marker ' + roleCfg.cssClass,
    html:        styleCfg.buildHtml(roleCfg),
    iconSize:    styleCfg.iconSize,
    iconAnchor:  styleCfg.iconAnchor,
    popupAnchor: styleCfg.popupAnchor,
  });
}

function buildPopupHtml(geo, type) {
  const { MAP_POINT_CONFIGS } = PathProbe.Config;
  const cfg   = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
  const lines = [
    '<div class="geo-popup">',
    '<span class="geo-popup__role geo-popup__role--' + type + '">' + esc(t(cfg.i18nKey)) + '</span>',
  ];
  if (geo.IP)          lines.push('<div class="geo-popup__ip">'  + esc(geo.IP)          + '</div>');
  if (geo.City)        lines.push('<div class="geo-popup__row">' + esc(geo.City)        + '</div>');
  if (geo.CountryName) lines.push('<div class="geo-popup__row">' + esc(geo.CountryName) + ' (' + esc(geo.CountryCode) + ')</div>');
  if (geo.OrgName)     lines.push('<div class="geo-popup__asn">' + 'AS' + esc(String(geo.ASN)) + ' ' + esc(geo.OrgName) + '</div>');
  lines.push('</div>');
  return lines.join('');
}

function buildMapLegend(pointTypes) {
  const { MAP_POINT_CONFIGS, MARKER_STYLE_CONFIGS } = PathProbe.Config;
  const legend = L.control({ position: 'bottomright' });
  legend.onAdd = function() {
    const div      = L.DomUtil.create('div', 'geo-legend');
    const styleCfg = MARKER_STYLE_CONFIGS[_markerStyleId] || MARKER_STYLE_CONFIGS['diamond-pulse'];
    div.innerHTML = pointTypes.map(type => {
      const cfg = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
      return '<div class="geo-legend__item">' +
        '<span class="geo-marker ' + esc(cfg.cssClass) + ' geo-legend__marker">' +
        styleCfg.buildHtml(cfg) + '</span>' +
        '<span data-i18n="' + esc(cfg.i18nKey) + '">' + esc(t(cfg.i18nKey)) + '</span>' +
        '</div>';
    }).join('');
    return div;
  };
  return legend;
}

// ── Connector management ──────────────────────────────────────────────────

function refreshConnectorLayer() {
  if (!isMapLoaded() || !_lastPub || !_lastTgt) return;
  if (_connectorLayer) { _connectorLayer.remove(); _connectorLayer = null; }
  if (!_lastPub.HasLocation || !_lastTgt.HasLocation) return;
  const { MARKER_COLOR_SCHEME_CONFIGS, CONNECTOR_LINE_CONFIGS } = PathProbe.Config;
  const scheme   = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                 || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
  const styleCfg = CONNECTOR_LINE_CONFIGS[_connectorStyleId || 'tick-xs']
                 || CONNECTOR_LINE_CONFIGS['tick-xs'];
  _connectorLayer = PathProbe.MapConnector.buildConnectorLayer(
    _lastPub, _lastTgt, styleCfg, scheme.originColor, scheme.targetColor, _map
  );
  _connectorLayer.addTo(_map);
}

// ── Marker redraw ─────────────────────────────────────────────────────────

function refreshMapMarkers() {
  if (!_map || (!_lastPub && !_lastTgt)) return;
  _map.eachLayer(layer => {
    if (layer instanceof L.Marker) _map.removeLayer(layer);
  });
  if (_legendControl) { _legendControl.remove(); _legendControl = null; }
  const points = [];
  if (_lastPub && _lastPub.HasLocation) points.push({ geo: _lastPub, type: 'origin' });
  if (_lastTgt && _lastTgt.HasLocation) points.push({ geo: _lastTgt, type: 'target' });
  for (const p of points) {
    L.marker([p.geo.Lat, p.geo.Lon], { icon: buildMarkerIcon(p.type) })
      .addTo(_map)
      .bindPopup(buildPopupHtml(p.geo, p.type));
  }
  if (points.length > 1) {
    _legendControl = buildMapLegend(points.map(p => p.type));
    _legendControl.addTo(_map);
  }
  refreshConnectorLayer();
}

// ── Haversine distance ───────────────────────────────────────────────────

function haversineKm(lat1, lon1, lat2, lon2) {
  const R      = 6371;
  const toRad  = deg => deg * Math.PI / 180;
  const dLat   = toRad(lat2 - lat1);
  const dLon   = toRad(lon2 - lon1);
  const a      = Math.sin(dLat/2)**2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon/2)**2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

// ── renderMap ─────────────────────────────────────────────────────────────

function renderMap(pub, tgt) {
  _lastPub = pub;
  _lastTgt = tgt;
  const { TILE_LAYER_CONFIGS, MARKER_COLOR_SCHEME_CONFIGS, CONNECTOR_LINE_CONFIGS } = PathProbe.Config;
  const container = document.getElementById('geo-map');
  const distEl    = document.getElementById('geo-distance');
  if (!container || typeof L === 'undefined') return;
  if (distEl) distEl.hidden = true;

  const points = [];
  if (pub && pub.HasLocation) points.push({ geo: pub, type: 'origin' });
  if (tgt && tgt.HasLocation) points.push({ geo: tgt, type: 'target' });

  if (points.length === 0) {
    container.classList.remove('visible');
    const outerEl = document.getElementById('geo-map-outer');
    if (outerEl) outerEl.hidden = true;
    if (_map) { _map.remove(); _map = null; _tileLayer = null; _connectorLayer = null; }
    return;
  }

  const outerEl = document.getElementById('geo-map-outer');
  if (outerEl) outerEl.hidden = false;
  container.classList.add('visible');
  renderMapBar();
  applyMarkerColorScheme();

  if (_map) { _map.remove(); _map = null; _tileLayer = null; _connectorLayer = null; }
  _map = L.map('geo-map');

  const tileCfg = TILE_LAYER_CONFIGS[getMapTileVariant()];
  _tileLayer    = L.tileLayer(tileCfg.url, { attribution: tileCfg.attribution, maxZoom: 18 })
                   .addTo(_map);

  const latLngs = [];
  for (const p of points) {
    L.marker([p.geo.Lat, p.geo.Lon], { icon: buildMarkerIcon(p.type) })
      .addTo(_map)
      .bindPopup(buildPopupHtml(p.geo, p.type));
    latLngs.push([p.geo.Lat, p.geo.Lon]);
  }

  if (latLngs.length === 1) { _map.setView(latLngs[0], 8); }
  else                       { _map.fitBounds(latLngs, { padding: [40, 40] }); }

  if (latLngs.length === 2) {
    const arcScheme   = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                      || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
    const arcStyleCfg = CONNECTOR_LINE_CONFIGS[_connectorStyleId || 'tick-xs']
                      || CONNECTOR_LINE_CONFIGS['tick-xs'];
    _connectorLayer = PathProbe.MapConnector.buildConnectorLayer(
      points[0].geo, points[1].geo, arcStyleCfg,
      arcScheme.originColor, arcScheme.targetColor, _map
    );
    _connectorLayer.addTo(_map);
    if (distEl) {
      const km = haversineKm(latLngs[0][0], latLngs[0][1], latLngs[1][0], latLngs[1][1]);
      distEl.textContent = t('map-distance') + ': ' + Math.round(km).toLocaleString() + ' km';
      distEl.hidden = false;
    }
  }

  if (points.length > 1) {
    _legendControl = buildMapLegend(points.map(p => p.type));
    _legendControl.addTo(_map);
  }

  requestAnimationFrame(() => { if (_map) _map.invalidateSize(); });
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.Map = {
  renderMap,
  setMapTileVariant,
  syncMapTileVariantToTheme,
  refreshMapMarkers,
  isMapLoaded,
  getMapTileVariant,
};

// Expose globally for HTML event handlers in renderMapBar().
window.setMapTileVariant = setMapTileVariant;
