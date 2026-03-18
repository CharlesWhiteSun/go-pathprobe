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
// Last rendered diagnostic report; retained so applyLocale() can re-render
// the results section in the new language whenever the user switches locale.
let _lastReport = null;
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

// ── Renderer / History re-render callbacks (locale.js bridge) ─────────────
// locale.js calls PathProbe.Renderer.rerenderLast() and
// PathProbe.History.rerenderLast() after a locale change so the results
// section and history list update to the new language.  Register these
// callbacks here in app.js (where renderReport / renderHistoryList live)
// until those functions are extracted to their own modules in later sub-tasks.
{
  const _ns = window.PathProbe || {};
  _ns.Renderer = _ns.Renderer || {};
  _ns.Renderer.rerenderLast = () => { if (_lastReport) renderReport(_lastReport); };
  _ns.History = _ns.History || {};
  _ns.History.rerenderLast  = () => { if (_lastHistoryItems) renderHistoryList(_lastHistoryItems); };
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

// ── Request building ──────────────────────────────────────────────────────

/**
 * Parse a Go duration string (e.g. "30s", "2m") into whole seconds.
 * Returns the defaultSec fallback when the string is empty or unparseable.
 */
function parseTimeoutSec(s, defaultSec) {
  if (!s) return defaultSec;
  const m = s.match(/^(\d+(?:\.\d+)?)(s|m)$/);
  if (!m) return defaultSec;
  const v = parseFloat(m[1]);
  return m[2] === 'm' ? v * 60 : v;
}

function buildRequest() {
  const target   = val('target');
  const mtrCount = Math.max(1, parseInt(val('mtr-count'), 10) || 5);
  let   timeout  = val('diag-timeout') || '30s';
  const insecure = checked('insecure');
  let   ports;
  if (target === 'web') {
    const mode = getModeFor('web');
    if (WEB_MODES_WITH_PORTS.includes(mode)) {
      ports = val('ports')
        .split(',')
        .map(s => parseInt(s.trim(), 10))
        .filter(n => n > 0 && n <= 65535);
    } else {
      ports = []; // other web modes (public-ip/dns/http/traceroute) don't use ports
    }
  } else {
    ports = val('ports')
      .split(',')
      .map(s => parseInt(s.trim(), 10))
      .filter(n => n > 0 && n <= 65535);
  }

  const opts = {
    mtr_count:   mtrCount,
    timeout,
    insecure,
    disable_geo: !checked('geo-enabled'),
    net: { host: val('host'), ports },
  };

  switch (target) {
    case 'web':  Object.assign(opts, { web:  buildWebOpts()  }); break;
    case 'smtp': Object.assign(opts, { smtp: buildSMTPOpts() }); break;
    case 'ftp':  Object.assign(opts, { ftp:  buildFTPOpts()  }); break;
    case 'sftp': Object.assign(opts, { sftp: buildSFTPOpts() }); break;
    // imap / pop: no protocol-specific options beyond net
  }

  // For Route Trace, ensure the request timeout covers the worst-case probe
  // time: maxHops × mtrCount × 2 s (backend hopTimeout) + 15 s buffer.
  // This prevents spurious "context deadline exceeded" errors on slow paths.
  if (target === 'web' && getModeFor('web') === 'traceroute') {
    const maxHops = parseInt(val('traceroute-max-hops'), 10) || 30;
    const minSec = maxHops * mtrCount * 2 + 15;
    if (parseTimeoutSec(opts.timeout, 30) < minSec) {
      opts.timeout = minSec + 's';
    }
  }

  return { target, options: opts };
}

function buildWebOpts() {
  const mode = getModeFor('web') || 'public-ip';
  const opts = { mode };
  if (mode === 'dns') {
    opts.domains = val('dns-domains').split(',').map(s => s.trim()).filter(Boolean);
    opts.types   = ['A', 'AAAA', 'MX'].filter(t => checked('dns-' + t));
  } else if (mode === 'http') {
    opts.url = val('http-url');
  } else if (mode === 'traceroute') {
    const maxHops = parseInt(val('traceroute-max-hops'), 10);
    if (maxHops > 0) opts.max_hops = maxHops;
  }
  return opts;
}

function buildSMTPOpts() {
  const mode = getModeFor('smtp') || 'handshake';
  return {
    mode,
    domain:       val('smtp-domain'),
    username:     val('smtp-user'),
    password:     val('smtp-pass'),
    from:         val('smtp-from'),
    to:           val('smtp-to').split(',').map(s => s.trim()).filter(Boolean),
    start_tls:    checked('smtp-starttls'),
    use_tls:      checked('smtp-ssl'),
    mx_probe_all: checked('smtp-mx-all'),
  };
}

function buildFTPOpts() {
  const mode = getModeFor('ftp') || 'login';
  return {
    mode,
    username: val('ftp-user'),
    password: val('ftp-pass'),
    use_tls:  checked('ftp-ssl'),
    auth_tls: checked('ftp-auth-tls'),
  };
}

function buildSFTPOpts() {
  const mode = getModeFor('sftp') || 'auth';
  return {
    mode,
    username: val('sftp-user'),
    password: val('sftp-pass'),
  };
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
  const entry     = document.createElement('div');
  entry.className = 'progress-entry';
  entry.innerHTML =
    '<span class="stage">' + esc(ev.stage   || '') + '</span>' +
    '<span class="msg">'  + esc(ev.message || '') + '</span>';
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

// ── Report rendering ──────────────────────────────────────────────────────
function renderReport(r) {
  _lastReport = r;
  document.getElementById('results-inner').innerHTML = [
    renderSummary(r),
    renderPortsSection(r.Ports),
    renderProtosSection(r.Protos),
    renderRouteSection(r.Route),
    renderGeoSection(r.PublicGeo, r.TargetGeo),
  ].filter(Boolean).join('');
}

function renderSummary(r) {
  const items = [
    [t('key-target'),    r.Target],
    [t('key-host'),      r.Host],
    [t('key-generated'), r.GeneratedAt],
  ];
  if (r.PublicGeo && r.PublicGeo.IP) {
    items.push([t('key-public-ip'), r.PublicGeo.IP]);
  }
  return '<div class="results-summary">' +
    items.map(([k, v]) =>
      '<div class="summary-item">' +
        '<div class="key">'  + esc(k)       + '</div>' +
        '<div class="val">'  + esc(v || '\u2014') + '</div>' +
      '</div>'
    ).join('') +
  '</div>';
}

function renderRouteSection(hops) {
  if (!hops || !hops.length) return '';
  const rows = hops.map(h => {
    const timedout = !h.IP;
    const rowClass = timedout ? ' class="hop-timedout"' : '';
    const ipCell   = timedout
      ? '<em>???</em>'
      : (h.Hostname && h.Hostname !== h.IP
          ? esc(h.IP) + ' <span class="hop-host">(' + esc(h.Hostname) + ')</span>'
          : esc(h.IP));
    const asnCell  = h.ASN ? 'AS' + esc(String(h.ASN)) : '';
    const country  = h.Country || '\u2014';
    const loss     = timedout ? '\u2014' : (h.LossPct || 0).toFixed(1) + '%';
    const rtt      = esc(h.AvgRTT || '\u2014');
    return '<tr' + rowClass + '>' +
      '<td>'         + esc(String(h.TTL)) + '</td>' +
      '<td>'         + ipCell             + '</td>' +
      '<td>'         + asnCell            + '</td>' +
      '<td>'         + esc(country)       + '</td>' +
      '<td>'         + loss               + '</td>' +
      '<td>'         + rtt                + '</td>' +
    '</tr>';
  }).join('');
  return '<div class="result-section">' +
    '<h3>' + esc(t('section-route')) + '</h3>' +
    '<table class="result-table route-table">' +
      '<thead><tr>' +
        '<th>' + esc(t('th-ttl'))     + '</th>' +
        '<th>' + esc(t('th-ip-host')) + '</th>' +
        '<th>' + esc(t('th-asn'))     + '</th>' +
        '<th>' + esc(t('th-country')) + '</th>' +
        '<th>' + esc(t('th-loss'))    + '</th>' +
        '<th>' + esc(t('th-avg-rtt')) + '</th>' +
      '</tr></thead>' +
      '<tbody>' + rows + '</tbody>' +
    '</table></div>';
}

function renderPortsSection(ports) {
  if (!ports || ports.length === 0) return '';
  const rows = ports.map(p =>
    '<tr>' +
      '<td><strong>' + esc(String(p.Port))          + '</strong></td>' +
      '<td>'         + esc(String(p.Sent))           + '</td>' +
      '<td>'         + esc(String(p.Received))       + '</td>' +
      '<td>'         + esc((p.LossPct || 0).toFixed(1)) + '%</td>' +
      '<td>'         + esc(p.MinRTT)                 + '</td>' +
      '<td>'         + esc(p.AvgRTT)                 + '</td>' +
      '<td>'         + esc(p.MaxRTT)                 + '</td>' +
    '</tr>'
  ).join('');
  return '<div class="result-section">' +
    '<h3>' + esc(t('section-ports')) + '</h3>' +
    '<table class="result-table">' +
      '<thead><tr>' +
        '<th>' + esc(t('th-port'))    + '</th>' +
        '<th>' + esc(t('th-sent'))    + '</th>' +
        '<th>' + esc(t('th-recv'))    + '</th>' +
        '<th>' + esc(t('th-loss'))    + '</th>' +
        '<th>' + esc(t('th-min-rtt'))+ '</th>' +
        '<th>' + esc(t('th-avg-rtt'))+ '</th>' +
        '<th>' + esc(t('th-max-rtt'))+ '</th>' +
      '</tr></thead>' +
      '<tbody>' + rows + '</tbody>' +
    '</table></div>';
}

function renderProtosSection(protos) {
  if (!protos || protos.length === 0) return '';
  const rows = protos.map(p => {
    const badge = p.OK
      ? '<span class="badge badge-ok">OK</span>'
      : '<span class="badge badge-fail">FAIL</span>';
    return '<tr>' +
      '<td><strong>' + esc(p.Protocol)     + '</strong></td>' +
      '<td>'         + esc(p.Host)         + '</td>' +
      '<td>'         + esc(String(p.Port)) + '</td>' +
      '<td>'         + badge               + '</td>' +
      '<td>'         + esc(p.Summary)      + '</td>' +
    '</tr>';
  }).join('');
  return '<div class="result-section">' +
    '<h3>' + esc(t('section-protos')) + '</h3>' +
    '<table class="result-table">' +
      '<thead><tr>' +
        '<th>' + esc(t('th-protocol'))+ '</th>' +
        '<th>' + esc(t('th-host'))    + '</th>' +
        '<th>' + esc(t('th-port'))    + '</th>' +
        '<th>' + esc(t('th-status'))  + '</th>' +
        '<th>' + esc(t('th-summary')) + '</th>' +
      '</tr></thead>' +
      '<tbody>' + rows + '</tbody>' +
    '</table></div>';
}

function renderGeoSection(pub, tgt) {
  const hasAny = (pub && pub.HasLocation) || (tgt && tgt.HasLocation);
  if (!hasAny) return '';
  return '<div class="result-section">' +
    '<h3>' + esc(t('section-geo')) + '</h3>' +
    '<div class="geo-grid">' +
      renderGeoBlock(t('geo-public-ip'),   pub) +
      renderGeoBlock(t('geo-target-host'), tgt) +
    '</div></div>';
}

function renderGeoBlock(label, geo) {
  if (!geo || !geo.IP) {
    return '<div class="geo-block"><h4>' + esc(label) + '</h4>' +
           '<p class="empty-note">' + esc(t('geo-no-data')) + '</p></div>';
  }
  const rows = [
    [t('geo-kv-ip'),      geo.IP],
    geo.City        ? [t('geo-kv-city'),    geo.City]                                        : null,
    geo.CountryName ? [t('geo-kv-country'), geo.CountryName + ' (' + geo.CountryCode + ')'] : null,
    geo.OrgName     ? [t('geo-kv-asn'),     'AS' + geo.ASN + ' ' + geo.OrgName]             : null,
  ].filter(Boolean);

  const kvHtml = rows.map(([k, v]) =>
    '<span class="k">' + esc(k) + '</span><span>' + esc(v) + '</span>'
  ).join('');

  return '<div class="geo-block">' +
    '<h4>' + esc(label) + '</h4>' +
    '<div class="geo-kv">' + kvHtml + '</div>' +
  '</div>';
}

// ── Utilities ─────────────────────────────────────────────────────────────

/**
 * Escape a value for safe insertion into HTML innerHTML.
 * All dynamic content from API responses is passed through this function
 * to prevent XSS from untrusted server banners or summaries.
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

// ── Connector arc utilities ───────────────────────────────────────────────────

/** Linearly interpolate between two hex colours.
 *  t is a value in [0, 1]; 0 returns hex1, 1 returns hex2.
 */
function lerpHex(hex1, hex2, t) {
  const parse = h => [parseInt(h.slice(1, 3), 16), parseInt(h.slice(3, 5), 16), parseInt(h.slice(5, 7), 16)];
  const toHex = n => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0');
  const [r1, g1, b1] = parse(hex1);
  const [r2, g2, b2] = parse(hex2);
  return '#' + toHex(r1 + (r2 - r1) * t) + toHex(g1 + (g2 - g1) * t) + toHex(b1 + (b2 - b1) * t);
}

/** Generate lat/lon waypoints along a northward quadratic-bezier arc.
 *  The Bézier control point is computed in Web-Mercator (EPSG:3857) space so
 *  the rendered curve appears geometrically smooth on the Leaflet Mercator map
 *  regardless of geographic scale or latitude.  arcFactor scales the control-
 *  point offset as a fraction of the Mercator straight-line distance.
 *  Returns an array of [lat, lon] pairs.
 */
function buildArcLatLngs(lat1, lon1, lat2, lon2, arcFactor, numSegments) {
  // Helpers: geographic ↔ Web-Mercator (metres, EPSG:3857).
  const R        = 6378137;
  const toMerc   = (lat, lon) => ({
    x: lon * Math.PI / 180 * R,
    y: Math.log(Math.tan(Math.PI / 4 + lat * Math.PI / 360)) * R,
  });
  const fromMerc = (x, y) => [
    (2 * Math.atan(Math.exp(y / R)) - Math.PI / 2) * 180 / Math.PI,
    x / R * 180 / Math.PI,
  ];
  const m1   = toMerc(lat1, lon1);
  const m2   = toMerc(lat2, lon2);
  const midX = (m1.x + m2.x) / 2;
  const midY = (m1.y + m2.y) / 2;
  const dist = Math.sqrt((m2.x - m1.x) ** 2 + (m2.y - m1.y) ** 2);
  // Control point bowed northward (positive Y in Mercator = north on the map).
  const ctlX = midX;
  const ctlY = midY + arcFactor * dist;
  const pts  = [];
  for (let i = 0; i <= numSegments; i++) {
    const t = i / numSegments;
    const u = 1 - t;
    pts.push(fromMerc(
      u * u * m1.x + 2 * u * t * ctlX + t * t * m2.x,
      u * u * m1.y + 2 * u * t * ctlY + t * t * m2.y,
    ));
  }
  return pts;
}

/** Render one directional arrowhead as an inline SVG element for use inside a
 *  Leaflet divIcon.  All shapes are defined on a normalised 10×10 viewBox and
 *  scaled to sz×sz screen pixels.  The SVG's own transform attribute rotates
 *  around the viewBox centre (5,5) so the anchor stays pixel-perfect.
 *
 *  shape values: 'triangle' | 'fat' | 'chevron' | 'double' | 'open' | 'pointer'
 */
function buildArrowSVG(shape, sz, color, opacity, rotateDeg) {
  const f  = ' fill="'   + color + '" opacity="' + opacity + '"';
  const sw = ' stroke="' + color + '" opacity="' + opacity +
             '" stroke-linecap="round" fill="none" stroke-width="1.8"';
  let inner;
  switch (shape) {
    case 'fat':     inner = '<polygon points="0,0.5 10,5 0,9.5"'    + f  + '/>'; break;
    case 'chevron': inner = '<polygon points="0,0 8,5 0,10 2,5"'    + f  + '/>'; break;
    case 'double':  inner = '<polygon points="0,0 5,5 0,10 1,5"'    + f  + '/>' +
                            '<polygon points="4,0 10,5 4,10 5.5,5"' + f  + '/>'; break;
    case 'open':    inner = '<polyline points="0.5,1 9,5 0.5,9"'    + sw + '/>'; break;
    case 'pointer': inner = '<polygon points="0,1.5 8,5 0,8.5 3,5"' + f  + '/>'; break;
    default:        inner = '<polygon points="0,1 10,5 0,9"'         + f  + '/>'; break;
  }
  return '<svg width="'  + sz + '" height="' + sz +
         '" viewBox="0 0 10 10" xmlns="http://www.w3.org/2000/svg">' +
         '<g transform="rotate(' + rotateDeg + ',5,5)">' + inner + '</g>' +
         '</svg>';
}

/** Returns true when the Leaflet map has been both created and initialised
 *  (setView / fitBounds has been called).  Operations that require a loaded
 *  map — such as latLngToLayerPoint() — must guard with isMapLoaded() to
 *  avoid Leaflet throwing "Set map center and zoom first."
 */
function isMapLoaded() {
  return Boolean(_map && _map._loaded);
}

/** Convert a '#rrggbb' hex colour and an alpha value [0,1] into an rgba() CSS
 *  string suitable for use as a canvas strokeStyle or fillStyle.
 */
function hexToRgba(hex, alpha) {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return 'rgba(' + r + ',' + g + ',' + b + ',' + alpha + ')';
}

/** ConnectorArcLayer — a Leaflet custom layer that draws the gradient arc
 *  connector on a dedicated HTML5 canvas in ONE drawing pass.
 *
 *  This is the correct architecture for seamless dot/dash patterns with a
 *  gradient colour.  Drawing the full arc as a single canvas path with
 *  createLinearGradient and setLineDash completely eliminates the three
 *  failure modes of the old N-polyline approach:
 *
 *    1. Sub-pixel gaps between adjacent SVG/canvas polyline segments.
 *    2. Doubled round end-caps where two segments meet (doubled dot at
 *       every junction point).
 *    3. Floating-point drift in the per-segment dashOffset accumulation.
 *
 *  Lifecycle
 *  ---------
 *  onAdd    Creates a <canvas> element inside the map container, sized to
 *           the visible viewport, and binds map event listeners.
 *  _redraw  Projects arc waypoints via latLngToContainerPoint(), builds one
 *           canvas path, applies a linear gradient (createLinearGradient)
 *           and an optional dot/dash rhythm (setLineDash), then strokes.
 *           Runs on every 'move', 'zoom', 'zoomend', and 'resize' event
 *           so the arc always tracks the live viewport.
 *  onRemove Unregisters listeners and removes the <canvas> element.
 */
const ConnectorArcLayer = L.Layer.extend({
  initialize: function(pts, styleCfg, originColor, targetColor) {
    this._pts         = pts;
    this._styleCfg    = styleCfg;
    this._originColor = originColor;
    this._targetColor = targetColor;
    this._canvas      = null;
    this._onRedraw    = null;
  },

  onAdd: function(map) {
    this._map = map;
    const canvas = document.createElement('canvas');
    // Place the canvas directly in the map container (not inside a pane)
    // so it is never affected by the CSS transforms Leaflet applies to
    // panes during animated pan / zoom.  z-index 450 puts it above the
    // overlayPane (400) but below the markerPane (600).
    canvas.style.cssText =
      'position:absolute;left:0;top:0;pointer-events:none;z-index:450;';
    map.getContainer().appendChild(canvas);
    this._canvas   = canvas;
    this._onRedraw = this._redraw.bind(this);
    map.on('move zoom zoomend resize', this._onRedraw);
    this._redraw();
    return this;
  },

  onRemove: function(map) {
    map.off('move zoom zoomend resize', this._onRedraw);
    if (this._canvas && this._canvas.parentNode) {
      this._canvas.parentNode.removeChild(this._canvas);
    }
    this._canvas   = null;
    this._onRedraw = null;
    this._map      = null;
  },

  _redraw: function() {
    if (!this._map || !this._canvas || !this._map._loaded) return;
    const map    = this._map;
    const size   = map.getSize();
    const canvas = this._canvas;
    const cfg    = this._styleCfg;
    const pts    = this._pts;
    if (pts.length < 2) return;

    // Resize canvas to cover the current viewport exactly.
    canvas.width  = size.x;
    canvas.height = size.y;

    // Project geographic arc waypoints to container-relative pixel coords.
    const sp  = pts.map(p => map.latLngToContainerPoint(L.latLng(p[0], p[1])));
    const ctx = canvas.getContext('2d');

    // Single continuous arc path — no segment boundaries, no repeated end-caps.
    ctx.beginPath();
    ctx.moveTo(sp[0].x, sp[0].y);
    for (let i = 1; i < sp.length; i++) ctx.lineTo(sp[i].x, sp[i].y);

    // Linear gradient from arc start to arc end gives a smooth colour flow
    // that closely follows the arc direction without per-segment complexity.
    const grad = ctx.createLinearGradient(
      sp[0].x, sp[0].y, sp[sp.length - 1].x, sp[sp.length - 1].y,
    );
    grad.addColorStop(0, hexToRgba(this._originColor, cfg.opacity));
    grad.addColorStop(1, hexToRgba(this._targetColor, cfg.opacity));

    ctx.strokeStyle = grad;
    ctx.lineWidth   = cfg.weight;
    ctx.lineCap     = 'round';
    ctx.lineJoin    = 'round';
    if (cfg.dashArray) {
      ctx.setLineDash(cfg.dashArray.split(/\s+/).map(Number));
    } else {
      ctx.setLineDash([]);
    }
    ctx.stroke();
  },
});

/** ConnectorGlowLayer — a Leaflet custom layer that animates a glowing "meteor"
 *  orb travelling from origin to target along the arc and looping with a brief
 *  pause at the destination.  Runs entirely on a dedicated HTML5 canvas using
 *  requestAnimationFrame so it never blocks the UI thread.
 *
 *  Visual anatomy
 *  --------------
 *  • Comet tail  — a series of fading radial-gradient circles receding behind
 *                  the head, shrinking and becoming transparent toward the back.
 *  • Outer glow  — a large radial-gradient circle centered on the current head
 *                  position, providing the "light source" halo effect.
 *  • Bright core — a small, near-white radial-gradient circle at the very front
 *                  that gives the impression of intense illumination.
 *
 *  Colour interpolates from originColor to targetColor as the orb travels,
 *  matching the arc's own gradient palette for visual coherence.
 *
 *  Lifecycle (same contract as ConnectorArcLayer)
 *  -----------------------------------------------
 *  onAdd    Creates a canvas above the arc layer, starts the rAF loop.
 *  onRemove Cancels the loop, removes the canvas, resets all state.
 */
const ConnectorGlowLayer = L.Layer.extend({
  initialize: function(pts, originColor, targetColor, glowCfg) {
    this._pts         = pts;
    this._originColor = originColor;
    this._targetColor = targetColor;
    this._glowCfg     = glowCfg;
    this._canvas      = null;
    this._rafId       = null;
    this._startTs     = null;
    this._scrPts      = null;
    this._cumDist     = null;
    this._needsResize = true;
    this._map         = null;
    this._onMapEvent  = null;
    this._boundTick   = null;
  },

  onAdd: function(map) {
    this._map = map;
    const canvas = document.createElement('canvas');
    // z-index 452: above ConnectorArcLayer (450) but below markers (600).
    canvas.style.cssText =
      'position:absolute;left:0;top:0;pointer-events:none;z-index:452;';
    map.getContainer().appendChild(canvas);
    this._canvas     = canvas;
    this._needsResize = true;
    this._onMapEvent = () => { this._scrPts = null; this._needsResize = true; };
    map.on('move zoom zoomend resize', this._onMapEvent);
    this._boundTick = this._tick.bind(this);
    this._rafId     = requestAnimationFrame(this._boundTick);
    return this;
  },

  onRemove: function(map) {
    if (this._onMapEvent) map.off('move zoom zoomend resize', this._onMapEvent);
    if (this._rafId !== null) { cancelAnimationFrame(this._rafId); this._rafId = null; }
    if (this._canvas && this._canvas.parentNode) {
      this._canvas.parentNode.removeChild(this._canvas);
    }
    this._canvas     = null;
    this._scrPts     = null;
    this._cumDist    = null;
    this._map        = null;
    this._onMapEvent = null;
    this._boundTick  = null;
  },

  _tick: function(ts) {
    if (!this._map || !this._canvas || !this._map._loaded) {
      this._rafId = requestAnimationFrame(this._boundTick);
      return;
    }
    if (this._startTs === null) this._startTs = ts;
    // Resize the canvas lazily whenever the viewport changes.
    if (this._needsResize) {
      const sz          = this._map.getSize();
      this._canvas.width  = sz.x;
      this._canvas.height = sz.y;
      this._needsResize   = false;
    }
    const cfg      = this._glowCfg;
    const fadeMs   = cfg.fadeMs   || 0;
    const cycleDur = cfg.travelMs + fadeMs + cfg.pauseMs;
    const elapsed  = (ts - this._startTs) % cycleDur;

    // Three-phase animation cycle:
    //   Phase 1 — travel  : head moves 0 → 1 along the arc at full brightness.
    //   Phase 2 — fade-out: head stays at destination, tail converges, all light
    //                        dissolves to 0 ("extinguish / breathing-lamp" effect).
    //   Phase 3 — dark    : canvas is blank until the next cycle begins.
    let progress;    // head position along arc [0, 1]
    let masterAlpha; // global brightness/opacity multiplier [0, 1]

    const ctx = this._canvas.getContext('2d');
    ctx.clearRect(0, 0, this._canvas.width, this._canvas.height);

    if (elapsed < cfg.travelMs) {
      // Phase 1: travelling.
      progress    = elapsed / cfg.travelMs;
      masterAlpha = 1.0;
    } else if (fadeMs > 0 && elapsed < cfg.travelMs + fadeMs) {
      // Phase 2: fade-out — head fixed at destination, linear brightness ramp.
      progress    = 1.0;
      masterAlpha = 1.0 - (elapsed - cfg.travelMs) / fadeMs;
    } else {
      // Phase 3: dark — canvas already cleared above, nothing to paint.
      this._rafId = requestAnimationFrame(this._boundTick);
      return;
    }

    this._drawGlow(ctx, progress, masterAlpha);
    this._rafId = requestAnimationFrame(this._boundTick);
  },

  /** Return cached screen-pixel projection; recomputes when invalidated. */
  _getScreenPts: function() {
    if (this._scrPts) return this._scrPts;
    const map = this._map;
    this._scrPts = this._pts.map(p =>
      map.latLngToContainerPoint(L.latLng(p[0], p[1])));
    const sp  = this._scrPts;
    const cum = [0];
    for (let i = 1; i < sp.length; i++) {
      const dx = sp[i].x - sp[i - 1].x;
      const dy = sp[i].y - sp[i - 1].y;
      cum.push(cum[i - 1] + Math.sqrt(dx * dx + dy * dy));
    }
    this._cumDist = cum;
    return sp;
  },

  /** Interpolate a screen-space {x,y} position at targetPx along the arc. */
  _posAtPx: function(sp, cum, targetPx) {
    const maxPx = cum[cum.length - 1];
    targetPx    = Math.max(0, Math.min(targetPx, maxPx));
    let j = 0;
    while (j < sp.length - 2 && cum[j + 1] < targetPx) j++;
    const segLen = cum[j + 1] - cum[j];
    const t      = segLen > 0 ? (targetPx - cum[j]) / segLen : 0;
    return {
      x: sp[j].x + t * (sp[j + 1].x - sp[j].x),
      y: sp[j].y + t * (sp[j + 1].y - sp[j].y),
    };
  },

  /** Draw the comet tail, outer glow halo, and bright core at `progress`.
   *  masterAlpha [0, 1] is a global brightness multiplier applied to every
   *  opacity value.  When it ramps from 1 → 0 during the fade-out phase:
   *    • tailPx shrinks proportionally so the tail converges into the head.
   *    • All radial-gradient alphas fade to 0, creating the extinguish effect.
   */
  _drawGlow: function(ctx, progress, masterAlpha) {
    const sp  = this._getScreenPts();
    const cum = this._cumDist;
    if (!sp || sp.length < 2 || !cum) return;
    const totalPx = cum[cum.length - 1];
    if (totalPx < 1) return;

    const cfg    = this._glowCfg;
    const headPx = progress * totalPx;
    const radius = cfg.glowRadius;
    // Scale base opacity by masterAlpha so all elements fade in unison.
    const alpha  = cfg.glowOpacity * masterAlpha;

    // ── Comet tail: shrinks with masterAlpha → converges into the head on fade-out ──
    const tailPx   = (cfg.tailLength || 0.18) * totalPx * masterAlpha;
    const TAIL_SAMP = 18;
    for (let i = TAIL_SAMP; i > 0; i--) {
      const ratio    = i / TAIL_SAMP;          // 1 = farthest back, 0 ≈ head
      const samplePx = Math.max(0, headPx - tailPx * ratio);
      const frac     = samplePx / totalPx;
      const pos      = this._posAtPx(sp, cum, samplePx);
      const color    = lerpHex(this._originColor, this._targetColor, frac);
      const a        = (1 - ratio) * alpha * 0.55;
      const r        = radius * (1 - ratio * 0.75);
      const grd      = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, r);
      grd.addColorStop(0, hexToRgba(color, a));
      grd.addColorStop(1, hexToRgba(color, 0));
      ctx.beginPath();
      ctx.arc(pos.x, pos.y, r, 0, Math.PI * 2);
      ctx.fillStyle = grd;
      ctx.fill();
    }

    // ── Outer glow halo ──────────────────────────────────────────────────────
    const headPos   = this._posAtPx(sp, cum, headPx);
    const headColor = lerpHex(this._originColor, this._targetColor, progress);
    const outerGrd  = ctx.createRadialGradient(
      headPos.x, headPos.y, 0, headPos.x, headPos.y, radius,
    );
    outerGrd.addColorStop(0,    hexToRgba(headColor, alpha));
    outerGrd.addColorStop(0.45, hexToRgba(headColor, alpha * 0.45));
    outerGrd.addColorStop(1,    hexToRgba(headColor, 0));
    ctx.beginPath();
    ctx.arc(headPos.x, headPos.y, radius, 0, Math.PI * 2);
    ctx.fillStyle = outerGrd;
    ctx.fill();

    // ── Bright white core ────────────────────────────────────────────────────
    const coreR   = Math.max(2, radius * 0.28);
    const coreGrd = ctx.createRadialGradient(
      headPos.x, headPos.y, 0, headPos.x, headPos.y, coreR,
    );
    coreGrd.addColorStop(0,   hexToRgba('#ffffff', 0.96 * masterAlpha));
    coreGrd.addColorStop(0.6, hexToRgba(headColor, 0.85 * masterAlpha));
    coreGrd.addColorStop(1,   hexToRgba(headColor, 0));
    ctx.beginPath();
    ctx.arc(headPos.x, headPos.y, coreR, 0, Math.PI * 2);
    ctx.fillStyle = coreGrd;
    ctx.fill();
  },
});

/** Build a Leaflet LayerGroup containing gradient-coloured divIcon arrow symbols
 *  distributed along the arc at a fixed screen-pixel spacing.  By working in
 *  pixel space the visual density is consistent at every zoom level and
 *  geographic scale, avoiding the sparse "   >    >    >   " appearance that
 *  arises when symbols are placed at equal geographic intervals.  Each symbol
 *  is rotated to align with the local arc tangent direction.
 *  Used by styles with type === 'arrows'.
 */
function buildArrowConnectorLayer(pub, tgt, styleCfg, originColor, targetColor) {
  const pts = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                               styleCfg.arcFactor, styleCfg.segments);
  const group = L.layerGroup();
  if (!isMapLoaded() || pts.length < 2) return group;

  // ── Optional spine ── a thin gradient polyline rendered behind the arrow icons.
  // When spineWeight > 0 the full arc path is always visible even at wide icon
  // spacing, preventing a "floating arrows in empty space" appearance.
  const spineW = styleCfg.spineWeight || 0;
  if (spineW > 0) {
    // Spine uses ConnectorArcLayer (single canvas path) for the same seamless
    // gradient rendering quality as the dot-family connector styles.
    const spineCfg = { weight: spineW, opacity: styleCfg.opacity, dashArray: null };
    group.addLayer(new ConnectorArcLayer(pts, spineCfg, originColor, targetColor));
  }

  // Convert all arc waypoints to screen pixels (consistent with current zoom/pan).
  const scrPts = pts.map(p => _map.latLngToLayerPoint(L.latLng(p[0], p[1])));
  const n      = scrPts.length;

  // Build a cumulative pixel-distance table so each symbol can be placed at a
  // precise fixed screen-pixel interval regardless of segment length variation.
  const cum = [0];
  for (let i = 1; i < n; i++) {
    const dx = scrPts[i].x - scrPts[i - 1].x;
    const dy = scrPts[i].y - scrPts[i - 1].y;
    cum.push(cum[i - 1] + Math.sqrt(dx * dx + dy * dy));
  }
  const totalPx = cum[n - 1];
  if (totalPx < 1) return group;

  const spacing = styleCfg.arrowSpacing || 40; // screen-px between successive symbols
  const sz      = styleCfg.arrowSize    || 14;
  let   nextPx  = spacing / 2;                 // start centred in first interval
  let   j       = 0;

  while (nextPx < totalPx) {
    // Advance segment pointer until the current segment straddles nextPx.
    while (j < n - 2 && cum[j + 1] < nextPx) j++;
    const segLen = cum[j + 1] - cum[j];
    const t      = segLen > 0 ? (nextPx - cum[j]) / segLen : 0;
    // Interpolated geographic position for the marker anchor.
    const lat = pts[j][0] + t * (pts[j + 1][0] - pts[j][0]);
    const lon = pts[j][1] + t * (pts[j + 1][1] - pts[j][1]);
    // Screen-space tangent for accurate per-symbol rotation (origin → target).
    const dx        = scrPts[j + 1].x - scrPts[j].x;
    const dy        = scrPts[j + 1].y - scrPts[j].y;
    const rotateDeg = Math.round(Math.atan2(dy, dx) * 180 / Math.PI);
    // Gradient colour proportional to pixel distance along the arc.
    const color = lerpHex(originColor, targetColor, nextPx / totalPx);
    L.marker([lat, lon], {
      icon: L.divIcon({
        className: 'connector-arrow-icon',
        html: buildArrowSVG(
          styleCfg.arrowShape || 'triangle', sz, color, styleCfg.opacity, rotateDeg),
        iconSize:   [sz, sz],
        iconAnchor: [sz / 2, sz / 2],
      }),
      interactive: false,
      keyboard:    false,
    }).addTo(group);
    nextPx += spacing;
  }
  return group;
}

/** Build a Leaflet LayerGroup that draws the origin→target connector arc.
 *  Dispatches to buildArrowConnectorLayer() for styles with type === 'arrows'.
 *  For polyline styles, a single ConnectorArcLayer is used: the whole arc is
 *  drawn on one HTML5 canvas path with createLinearGradient + setLineDash,
 *  giving a seamless gradient dot/dash pattern with no inter-segment gaps.
 *  When styleCfg.glowEnabled is true a ConnectorGlowLayer is added on top,
 *  rendering the meteor light-pulse animation without coupling to the base arc.
 */
function buildConnectorLayer(pub, tgt, styleCfg, originColor, targetColor) {
  let group;
  if ((styleCfg.type || 'polyline') === 'arrows') {
    group = buildArrowConnectorLayer(pub, tgt, styleCfg, originColor, targetColor);
  } else {
    const pts = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                 styleCfg.arcFactor, styleCfg.segments);
    group = L.layerGroup();
    group.addLayer(new ConnectorArcLayer(pts, styleCfg, originColor, targetColor));
  }
  // Overlay the glowing meteor animation when the style opts in.
  if (styleCfg.glowEnabled) {
    const pts     = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                    styleCfg.arcFactor, styleCfg.segments);
    const glowCfg = CONNECTOR_GLOW_CONFIGS[styleCfg.glowConfig || 'default']
                  || CONNECTOR_GLOW_CONFIGS['default'];
    group.addLayer(new ConnectorGlowLayer(pts, originColor, targetColor, glowCfg));
  }
  return group;
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
