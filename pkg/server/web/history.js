'use strict';
// ── history.js — history panel management (PathProbe.History) ─────────────
// Depends on: locale.js, renderer.js, map.js
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

let _lastHistoryItems = null;

function esc(s) {
  return String(s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

/** Format a UTC ISO timestamp for display using the active locale. */
function formatHistoryTime(isoString) {
  if (!isoString) return '';
  // _locale is private to locale.js; use document.documentElement.lang as proxy.
  const locale = document.documentElement.lang || 'en';
  try   { return new Date(isoString).toLocaleString(locale); }
  catch (_) { return new Date(isoString).toLocaleString(); }
}

/** Render the history list items into #history-list. */
function renderHistoryList(items) {
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
      '<span class="hi-badge">' + esc(item.target || '\u2014') + '</span>' +
      '<span class="hi-host">'  + esc(item.host   || '\u2014') + '</span>' +
      '<span class="hi-time">'  + esc(ts)                      + '</span>' +
    '</li>';
  }).join('');
}

/** Fetch the history list from the server and re-render the panel. */
async function loadHistory() {
  try {
    const r = await fetch('/api/history');
    if (!r.ok) return;
    const items = await r.json();
    renderHistoryList(Array.isArray(items) ? items : []);
  } catch (_) { /* non-fatal */ }
}

/** Fetch a single history entry and display it as the current results. */
async function loadHistoryEntry(id) {
  const resultEl = document.getElementById('results');
  try {
    const r = await fetch('/api/history/' + encodeURIComponent(id));
    if (!r.ok) {
      PathProbe.ApiClient.showError('History entry not found: ' + id);
      return;
    }
    const report = await r.json();
    PathProbe.Renderer.renderReport(report);
    if (resultEl) resultEl.hidden = false;
    PathProbe.Map.renderMap(report.PublicGeo, report.TargetGeo);
    if (resultEl) resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
  } catch (err) {
    PathProbe.ApiClient.showError('Failed to load history entry: ' + err.message);
  }
}

/** Re-render the history list in the current locale (called by applyLocale). */
function rerenderLast() {
  if (_lastHistoryItems) renderHistoryList(_lastHistoryItems);
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.History = { loadHistory, renderHistoryList, loadHistoryEntry, rerenderLast };

// Expose globally for inline onclick in dynamically generated list HTML.
window.loadHistoryEntry = loadHistoryEntry;
