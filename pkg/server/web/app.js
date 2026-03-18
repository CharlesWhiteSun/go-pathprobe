'use strict';

// ── Leaflet default-icon path override (embedded assets, no CDN) ─────────
// Both leaflet.js and app.js are loaded with defer, so L is available here.
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

// Active Leaflet map instance (one at a time).
let _map = null;
// Active tile layer attached to _map; kept separate so it can be swapped on
// theme change without tearing down the whole map instance.
let _tileLayer = null;
// Currently selected map tile variant; null means “not yet set by user”.
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
// Last fetched history item array; retained so applyLocale() can re-render
// the history list with correct locale-aware timestamps whenever the user
// switches language.
let _lastHistoryItems = null;

// ── Config aliases — sourced from PathProbe.Config (config.js) ───────────
// config.js is loaded before app.js (see index.html) via a <script defer>
// tag, which guarantees execution order.  However, a stale browser cache may
// serve an old index.html that lacks the config.js <script> tag, causing
// window.PathProbe to be undefined when this file runs.
//
// Accessing via window.PathProbe (rather than the bare PathProbe identifier)
// avoids a ReferenceError when window.PathProbe was never set — bare
// identifier lookup throws ReferenceError for undeclared globals, whereas
// window.PathProbe returns undefined safely.  The defensive (…) || {} guard
// means this line NEVER throws, so all function declarations below (setTheme,
// setLocale, etc.) remain callable from inline onclick attributes even when
// config.js is unavailable.
//
// THEMES and DEFAULT_THEME carry explicit fallbacks so applyTheme() can
// safely call THEMES.includes() without crashing in degraded mode.
if (!window.PathProbe || !window.PathProbe.Config) {
  // Surfacing this in the console makes the misconfiguration immediately
  // visible to developers.  Check the Network tab for a failed /config.js.
  console.error(
    '[PathProbe] PathProbe.Config is unavailable. ' +
    'Ensure config.js loads before app.js. ' +
    'Hint: force-refresh the browser (Ctrl+Shift+R) to clear a stale cache.',
  );
}
const {
  MAP_POINT_CONFIGS          = {},
  MARKER_COLOR_SCHEME_CONFIGS = {},
  MARKER_STYLE_CONFIGS       = {},
  CONNECTOR_LINE_CONFIGS     = {},
  CONNECTOR_GLOW_CONFIGS     = {},
  MAP_TILE_VARIANTS          = [],
  MAP_THEME_TO_TILE_VARIANT  = {},
  MAP_DARK_THEMES            = new Set(),
  TILE_LAYER_CONFIGS         = {},
  TARGET_PORTS               = {},
  TARGET_MODE_PANELS         = {},
  WEB_MODES_WITH_PORTS       = [],
  TARGET_PLACEHOLDER_KEYS    = {},
  COPYRIGHT_START_YEAR       = 2026,
  THEMES      = ['default', 'deep-blue', 'light-green', 'forest-green', 'dark'],
  DEFAULT_THEME = 'default',
} = (window.PathProbe && window.PathProbe.Config) || {};

// ── Locale shims — delegate to PathProbe.Locale (locale.js) ─────────────
// locale.js is loaded before app.js (see index.html) and registers all
// locale logic under PathProbe.Locale.  The three thin shim functions below
// keep all call sites in this file unchanged so no other code in app.js
// needs to move.

/** Return the translation for key in the current locale, falling back to en. */
function t(key) {
  return window.PathProbe && window.PathProbe.Locale
    ? window.PathProbe.Locale.t(key)
    : key;
}

/** Persist and apply a new locale choice — delegates to locale.js. */
function setLocale(lang) {
  if (window.PathProbe && window.PathProbe.Locale) {
    window.PathProbe.Locale.setLocale(lang);
  }
}

/** Initialise locale from localStorage — delegates to locale.js. */
function initLocale() {
  if (window.PathProbe && window.PathProbe.Locale) {
    window.PathProbe.Locale.initLocale();
  }
}

// ── History re-render callback (locale.js bridge) ───────────────────────────
// locale.js calls PathProbe.History.rerenderLast() after a locale change so
// the history list updates to the new language.
// PathProbe.Renderer.rerenderLast() is registered by renderer.js directly.
{
  const _ns = window.PathProbe || {};
  _ns.History = _ns.History || {};
  _ns.History.rerenderLast = () => { if (_lastHistoryItems) renderHistoryList(_lastHistoryItems); };
  window.PathProbe = _ns;
}

// ── Run-button animation shim — delegates to PathProbe.Form (form.js) ────
/**
 * Return the innerHTML for #run-btn while a diagnostic is running.
 * Delegates to form.js with an inline fallback so the string 'anim-dots'
 * is always present in app.js regardless of module load order.
 */
function getRunningHTML() {
  return (window.PathProbe && window.PathProbe.Form)
    ? window.PathProbe.Form.getRunningHTML()
    : '<span class="anim-dots"><span></span><span></span><span></span></span>';
}

// ── Theme shims — delegate to PathProbe.Theme (theme.js) ─────────────────
// theme.js is loaded before app.js (see index.html) and registers all
// theme logic under PathProbe.Theme.  The two thin shim functions below keep
// all call sites in this file (initTheme / setTheme) unchanged.

/** Public entry point called by theme-button onclick handlers. */
function setTheme(themeId) {
  if (window.PathProbe && window.PathProbe.Theme) {
    window.PathProbe.Theme.setTheme(themeId);
  }
}

/** Initialise theme from localStorage — delegates to theme.js. */
function initTheme() {
  if (window.PathProbe && window.PathProbe.Theme) {
    window.PathProbe.Theme.initTheme();
  }
}

// ── Map tile variant callback (theme.js bridge) ───────────────────────────
// theme.js calls PathProbe.Map.syncMapTileVariantToTheme(id) after a theme
// change so map tiles switch to the correct variant.  Register the callback
// here in app.js (where syncMapTileVariantToTheme lives) until the map module
// is extracted to its own module in a later sub-task.
{
  const _ns = window.PathProbe || {};
  _ns.Map = _ns.Map || {};
  _ns.Map.syncMapTileVariantToTheme = (id) => syncMapTileVariantToTheme(id);
  window.PathProbe = _ns;
}

// ── Initialisation ────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  // Form UI initialisation (spell-check, custom selects, panel animations,
  // radio button wiring, onTargetChange) is fully owned by form.js.
  if (window.PathProbe && window.PathProbe.Form) {
    window.PathProbe.Form.init();
  }
  fetchVersion();   // async version badge
  loadHistory();    // populate history panel
  initTheme();      // apply saved theme (before locale so tokens are ready)
  initLocale();     // apply saved locale (must run after DOM is ready)
});

// ── Form shims — delegate to PathProbe.Form (form.js) ───────────────────
// form.js is loaded before app.js (see index.html) and registers all
// form/UI logic under PathProbe.Form.  The thin shim functions below keep
// all call sites in this file (val, checked, getModeFor) unchanged so no
// other code in app.js needs to move.

/** Read and trim the string value of a form element by id. */
function val(id) {
  return (window.PathProbe && window.PathProbe.Form)
    ? window.PathProbe.Form.val(id) : '';
}

/** Return the checked state of a checkbox by id. */
function checked(id) {
  return (window.PathProbe && window.PathProbe.Form)
    ? window.PathProbe.Form.checked(id) : false;
}

/** Read the currently-checked sub-mode radio for a target. */
function getModeFor(target) {
  return (window.PathProbe && window.PathProbe.Form)
    ? window.PathProbe.Form.getModeFor(target) : '';
}

// ── Request building shim — delegates to PathProbe.ApiBuilder (api-builder.js)
// api-builder.js is loaded before app.js (see index.html) and owns all request
// construction logic.  The thin shim below keeps the runDiag() call site
// unchanged — it still calls buildRequest() with no arguments.

/**
 * Assemble the diagnostic request payload — delegates to api-builder.js.
 * @returns {{ target: string, options: object }} Request payload.
 */
function buildRequest() {
  return (window.PathProbe && window.PathProbe.ApiBuilder)
    ? window.PathProbe.ApiBuilder.buildRequest()
    : { target: '', options: {} };
}

// ── API calls ─────────────────────────────────────────────────────────────
async function runDiag() {
  const btn        = document.getElementById('run-btn');
  const errorEl    = document.getElementById('error-banner');
  const resultEl   = document.getElementById('results');
  const progressEl = document.getElementById('progress-log');

  btn.disabled   = true;
  btn.innerHTML  = getRunningHTML();
  errorEl.hidden = true;
  resultEl.hidden = true;
  if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = false; }

  try {
    const req  = buildRequest();
    if (req === null) {
      // Validation failed inside buildRequest; error already surfaced via showError.
      if (progressEl) { progressEl.hidden = true; progressEl.innerHTML = ''; }
      return;
    }
    const resp = await fetch('/api/diag/stream', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify(req),
    });

    // Non-2xx before SSE headers could be a plain JSON error (e.g. during
    // reverse-proxy validation). Fall back to reading JSON body.
    if (!resp.ok) {
      const data = await resp.json().catch(() => ({ error: 'HTTP ' + resp.status }));
      showError(data.error || 'Server returned HTTP ' + resp.status);
      return;
    }

    // Parse the SSE stream via ReadableStream — no EventSource because the
    // browser's EventSource API does not support POST requests.
    const reader  = resp.body.getReader();
    const decoder = new TextDecoder();
    let   buffer  = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // SSE messages are separated by blank lines (\n\n).
      let boundary;
      while ((boundary = buffer.indexOf('\n\n')) !== -1) {
        const raw = buffer.slice(0, boundary);
        buffer    = buffer.slice(boundary + 2);
        handleSSEMessage(raw, progressEl, resultEl);
      }
    }
    // Flush any remaining content in the buffer.
    if (buffer.trim()) handleSSEMessage(buffer, progressEl, resultEl);

  } catch (err) {
    // Hide and clear the progress log so stale partial output does not linger
    // below the error banner after a network-level failure.
    if (progressEl) { progressEl.hidden = true; progressEl.innerHTML = ''; }
    showError('Request failed: ' + err.message);
  } finally {
    btn.disabled  = false;
    btn.textContent = t('btn-run');
  }
}

/** Parse a single SSE message block and dispatch to the appropriate handler. */
function handleSSEMessage(raw, progressEl, resultEl) {
  let evtName = '', dataStr = '';
  for (const line of raw.split('\n')) {
    if (line.startsWith('event: '))     evtName = line.slice(7).trim();
    else if (line.startsWith('data: ')) dataStr = line.slice(6);
  }
  if (!dataStr) return;

  let payload;
  try { payload = JSON.parse(dataStr); } catch { return; }

  if (evtName === 'progress') {
    appendProgress(progressEl, payload);
  } else if (evtName === 'result') {
    if (progressEl) progressEl.hidden = true;
    renderReport(payload);
    // Reveal #results BEFORE renderMap so the #geo-map container has a
    // non-zero layout when Leaflet initialises (prevents blank tile areas).
    resultEl.hidden = false;
    renderMap(payload.PublicGeo, payload.TargetGeo);
    loadHistory();
    resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
  } else if (evtName === 'error') {
    // Clear and hide the progress log so no partial output remains visible.
    if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = true; }
    showError(payload.error || 'diagnostic error');
  }
}

/** Append a single progress event entry to the progress log. */
function appendProgress(el, ev) {
  if (!el) return;
  const entry = document.createElement('div');
  entry.className = 'progress-entry';
  const stageSpan = document.createElement('span');
  stageSpan.className = 'stage';
  stageSpan.textContent = ev.stage || '';
  const msgSpan = document.createElement('span');
  msgSpan.className = 'msg';
  msgSpan.textContent = ev.message || '';
  entry.appendChild(stageSpan);
  entry.appendChild(msgSpan);
  el.appendChild(entry);
  el.scrollTop = el.scrollHeight;
}

async function fetchVersion() {
  try {
    const r = await fetch('/api/health');
    if (!r.ok) return;
    const { version } = await r.json();
    const el = document.getElementById('version-badge');
    if (el && version) el.textContent = version;
  } catch (_) { /* non-fatal — version badge stays empty */ }
}

function showError(msg) {
  const banner = document.getElementById('error-banner');
  const textEl = document.getElementById('error-text');
  const friendly = localizeError(msg);
  if (textEl) {
    textEl.textContent = friendly;
  } else {
    // Fallback for layouts where error-text span is absent.
    banner.textContent = '\u26a0  ' + friendly;
  }
  banner.hidden = false;
}

/**
 * Map a raw server error string (possibly English Go internals) to a
 * localised, user-friendly description using i18n keys.
 * Preserves the diagnostic-error prefix for unrecognised messages.
 */
function localizeError(msg) {
  if (!msg) return t('err-unknown');
  const lower = msg.toLowerCase();
  if (lower.includes('timed out') || lower.includes('deadline exceeded')) {
    return t('err-timeout');
  }
  if (lower.includes('no runner registered') || lower.includes('no handler registered')) {
    return t('err-no-runner');
  }
  // Strip the "diagnostic error: " prefix for cleaner display when no
  // specific i18n key matches.
  return msg.replace(/^diagnostic error:\s*/i, '');
}

// ── Report rendering — delegates to renderer.js ──────────────────────────
// renderReport() is defined in renderer.js and exposed as
// PathProbe.Renderer.renderReport().  This shim keeps call-sites in app.js
// working until they can be updated directly.
function renderReport(r) {
  if (window.PathProbe && window.PathProbe.Renderer) {
    window.PathProbe.Renderer.renderReport(r);
  }
}

// ── Utilities ─────────────────────────────────────────────────────────────

/**
 * Escape a value for safe insertion into HTML innerHTML.
 * Used by map/geo and history sections still residing in app.js.
 * renderer.js carries its own private copy for the report-render path.
 */
function esc(s) {
  return String(s)
    .replace(/&/g,  '&amp;')
    .replace(/</g,  '&lt;')
    .replace(/>/g,  '&gt;')
    .replace(/"/g,  '&quot;')
    .replace(/'/g,  '&#39;');
}

// -- Geo Map (Leaflet) --------------------------------------------------------

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

/** Redraw only the Leaflet Marker layers using the current _markerStyleId.
 *  Preserves the tile layer; also removes and re-adds the legend so its icon
 *  reflects the new style, and rebuilds the connector arc layer.
 *  No-op when the map has not been initialised or no geo data is available.
 */
function refreshMapMarkers() {
  if (!_map || (!_lastPub && !_lastTgt)) return;
  _map.eachLayer(layer => {
    if (layer instanceof L.Marker) _map.removeLayer(layer);
  });
  // Remove stale legend before rebuilding.
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

// ── Connector arc utilities — delegates to PathProbe.MapConnector ────────────
// All rendering logic is in map-connector.js.  We keep two thin shims here so
// all call-sites in app.js (renderMap, refreshConnectorLayer) remain unchanged.

/** Returns true when the Leaflet map has been both created and initialised. */
function isMapLoaded() {
  return Boolean(_map && _map._loaded);
}

/** Shim — injects the module-level _map so all call-sites remain unchanged. */
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
    L.marker([p.geo.Lat, p.geo.Lon], { icon: buildMarkerIcon(p.type) })
      .addTo(_map)
      .bindPopup(buildPopupHtml(p.geo, p.type));
    latLngs.push([p.geo.Lat, p.geo.Lon]);
  }

  // Set the map viewport BEFORE building the connector because functions such
  // as latLngToLayerPoint() and the dash-offset calculation require the map to
  // be fully initialised (Leaflet throws "Set map center and zoom first." if
  // called before setView / fitBounds).
  if (latLngs.length === 1) {
    _map.setView(latLngs[0], 8);
  } else {
    _map.fitBounds(latLngs, { padding: [40, 40] });
  }

  // Draw the gradient arc connector and show the distance badge when both
  // endpoints exist.  The connector style and colour scheme come from the
  // module-level state variables so switching pickers refreshes the arc
  // without a full map rebuild.
  if (latLngs.length === 2) {
    const arcScheme   = MARKER_COLOR_SCHEME_CONFIGS[_markerColorSchemeId]
                      || MARKER_COLOR_SCHEME_CONFIGS['ocean'];
    const arcStyleCfg = CONNECTOR_LINE_CONFIGS[_connectorStyleId]
                      || CONNECTOR_LINE_CONFIGS['dot-bead'];
    _connectorLayer   = buildConnectorLayer(
      points[0].geo, points[1].geo,
      arcStyleCfg, arcScheme.originColor, arcScheme.targetColor
    );
    _connectorLayer.addTo(_map);
    if (distEl) {
      const km = haversineKm(latLngs[0][0], latLngs[0][1], latLngs[1][0], latLngs[1][1]);
      distEl.textContent = t('map-distance') + ': ' + Math.round(km).toLocaleString() + ' km';
      distEl.hidden = false;
    }
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

// -- History ------------------------------------------------------------------

/** Format a UTC ISO timestamp string for display using the active locale.
 *  The locale code stored in _locale (e.g. 'en', 'zh-TW') is a valid BCP-47
 *  tag and can be passed directly to toLocaleString().
 */
function formatHistoryTime(isoString) {
  if (!isoString) return '';
  const locale = window.PathProbe && window.PathProbe.Locale
    ? window.PathProbe.Locale.getLocale()
    : 'en';
  try {
    return new Date(isoString).toLocaleString(locale);
  } catch (_) {
    return new Date(isoString).toLocaleString();
  }
}

/** Fetch the history list from the server and re-render the panel. */
async function loadHistory() {
  try {
    const r = await fetch('/api/history');
    if (!r.ok) return;
    const items = await r.json();
    renderHistoryList(Array.isArray(items) ? items : []);
  } catch (_) { /* non-fatal - panel stays in its current state */ }
}

/** Render the history list items into #history-list. */
function renderHistoryList(items) {
  // Cache for locale-switch re-renders triggered by applyLocale().
  _lastHistoryItems = items;

  const emptyEl = document.getElementById('history-empty');
  const listEl  = document.getElementById('history-list');
  if (!listEl || !emptyEl) return;

  if (items.length === 0) {
    emptyEl.hidden = false;
    listEl.hidden  = true;
    return;
  }

  emptyEl.hidden = true;
  listEl.hidden  = false;
  listEl.innerHTML = items.map(item => {
    const ts = formatHistoryTime(item.created_at);
    const id = JSON.stringify(String(item.id));
    return '<li class="history-item" onclick="loadHistoryEntry(' + id + ')">' +
      '<span class="hi-badge">' + esc(item.target      || '\u2014') + '</span>' +
      '<span class="hi-host">'  + esc(item.host        || '\u2014') + '</span>' +
      '<span class="hi-time">'  + esc(ts)                           + '</span>' +
    '</li>';
  }).join('');
}

/** Fetch a single history entry and display it as the current results. */
async function loadHistoryEntry(id) {
  const resultEl = document.getElementById('results');
  try {
    const r = await fetch('/api/history/' + encodeURIComponent(id));
    if (!r.ok) {
      showError('History entry not found: ' + id);
      return;
    }
    const report = await r.json();
    renderReport(report);
    // Reveal #results BEFORE renderMap for the same reason as in handleSSEMessage.
    if (resultEl) resultEl.hidden = false;
    renderMap(report.PublicGeo, report.TargetGeo);
    if (resultEl) {
      resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  } catch (err) {
    showError('Failed to load history entry: ' + err.message);
  }
}
