'use strict';
// ── api-client.js — API calls: SSE streaming, fetch, error handling ────────
// Depends on: locale.js, form.js, api-builder.js, renderer.js, map.js, history.js
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

function t(key) { return PathProbe.Locale.t(key); }

/**
 * Map a raw server error string to a localised, user-friendly description.
 * Preserves the message for unrecognised errors.
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
  return msg.replace(/^diagnostic error:\s*/i, '');
}

function showError(msg) {
  const banner = document.getElementById('error-banner');
  const textEl = document.getElementById('error-text');
  const friendly = localizeError(msg);
  if (textEl) {
    textEl.textContent = friendly;
  } else {
    banner.textContent = '\u26a0  ' + friendly;
  }
  banner.hidden = false;
}

/** Append a single progress event entry to the progress log. */
function appendProgress(el, ev) {
  if (!el) return;
  const entry     = document.createElement('div');
  entry.className = 'progress-entry';
  // Use textContent to prevent XSS from progress message values.
  const stageSpan = document.createElement('span');
  stageSpan.className   = 'stage';
  stageSpan.textContent = ev.stage || '';
  const msgSpan = document.createElement('span');
  msgSpan.className   = 'msg';
  msgSpan.textContent = ev.message || '';
  entry.appendChild(stageSpan);
  entry.appendChild(msgSpan);
  el.appendChild(entry);
  el.scrollTop = el.scrollHeight;
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
    PathProbe.Renderer.renderReport(payload);
    // Reveal #results BEFORE renderMap so #geo-map has a non-zero layout.
    resultEl.hidden = false;
    PathProbe.Map.renderMap(payload.PublicGeo, payload.TargetGeo);
    PathProbe.History.loadHistory();
    resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
  } else if (evtName === 'error') {
    if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = true; }
    showError(payload.error || 'diagnostic error');
  }
}

async function runDiag() {
  const btn        = document.getElementById('run-btn');
  const errorEl    = document.getElementById('error-banner');
  const resultEl   = document.getElementById('results');
  const progressEl = document.getElementById('progress-log');

  btn.disabled    = true;
  btn.innerHTML   = PathProbe.Form.getRunningHTML();
  errorEl.hidden  = true;
  resultEl.hidden = true;
  if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = false; }

  try {
    const req  = PathProbe.ApiBuilder.buildRequest();
    if (req === null) {
      if (progressEl) { progressEl.hidden = true; progressEl.innerHTML = ''; }
      return;
    }
    const resp = await fetch('/api/diag/stream', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify(req),
    });

    if (!resp.ok) {
      const data = await resp.json().catch(() => ({ error: 'HTTP ' + resp.status }));
      showError(data.error || 'Server returned HTTP ' + resp.status);
      return;
    }

    const reader  = resp.body.getReader();
    const decoder = new TextDecoder();
    let   buffer  = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let boundary;
      while ((boundary = buffer.indexOf('\n\n')) !== -1) {
        const raw = buffer.slice(0, boundary);
        buffer    = buffer.slice(boundary + 2);
        handleSSEMessage(raw, progressEl, resultEl);
      }
    }
    if (buffer.trim()) handleSSEMessage(buffer, progressEl, resultEl);

  } catch (err) {
    if (progressEl) { progressEl.hidden = true; progressEl.innerHTML = ''; }
    showError('Request failed: ' + err.message);
  } finally {
    btn.disabled    = false;
    btn.textContent = t('btn-run');
  }
}

async function fetchVersion() {
  try {
    const r = await fetch('/api/health');
    if (!r.ok) return;
    const { version } = await r.json();
    const el = document.getElementById('version-badge');
    if (el && version) el.textContent = version;
  } catch (_) { /* non-fatal */ }
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.ApiClient = { runDiag, fetchVersion, showError, localizeError };

// Expose globally for HTML event handlers.
window.runDiag = runDiag;
