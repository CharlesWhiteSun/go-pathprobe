'use strict';

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

// ── History namespace registration ──────────────────────────────────────────
// locale.js 在語言切換後透過 PathProbe.History.rerenderLast() 重繪歷史清單。
// api-client.js 在 SSE result 事件後透過 PathProbe.History.loadHistory() 刷新清單。
// PathProbe.Renderer.rerenderLast() 由 renderer.js 直接注冊。
{
  const _ns = window.PathProbe || {};
  _ns.History = _ns.History || {};
  _ns.History.rerenderLast = () => { if (_lastHistoryItems) renderHistoryList(_lastHistoryItems); };
  _ns.History.loadHistory  = () => loadHistory();
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

// ── Map shims — delegate to PathProbe.Map (map.js) ────────────────────

/** Render (or remove) the Leaflet map — delegates to map.js. */
function renderMap(pub, tgt) {
  if (window.PathProbe && window.PathProbe.Map) {
    window.PathProbe.Map.renderMap(pub, tgt);
  }
}

/** Refresh map markers after a style/colour-scheme change — delegates to map.js. */
function refreshMapMarkers() {
  if (window.PathProbe && window.PathProbe.Map) {
    window.PathProbe.Map.refreshMapMarkers();
  }
}

// ── Initialisation ────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  // Form UI initialisation (spell-check, custom selects, panel animations,
  // radio button wiring, onTargetChange) is fully owned by form.js.
  if (window.PathProbe && window.PathProbe.Form) {
    window.PathProbe.Form.init();
  }
  fetchVersion();   // async version badge — delegates to api-client.js
  loadHistory();    // populate history panel
  initTheme();      // apply saved theme (before locale so tokens are ready)
  initLocale();     // apply saved locale (must run after DOM is ready)
});

// ── api-client.js shims ──────────────────────────────────────────────────
// fetchVersion 和 showError 在此保留薄層 shim，維持 DOMContentLoaded 與
// loadHistoryEntry 中的呼叫點不變，並將實際邏輯委派給 api-client.js。

/** 取得版本號碼 — 委派給 api-client.js。 */
function fetchVersion() {
  if (window.PathProbe && window.PathProbe.ApiClient) {
    window.PathProbe.ApiClient.fetchVersion();
  }
}

/** 顯示錯誤橫幅 — 委派給 api-client.js。 */
function showError(msg) {
  if (window.PathProbe && window.PathProbe.ApiClient) {
    window.PathProbe.ApiClient.showError(msg);
  }
}

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
