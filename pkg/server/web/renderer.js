'use strict';
// ── renderer.js — report rendering (PathProbe.Renderer) ───────────────────
// Depends on: locale.js (PathProbe.Locale)
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

let _lastReport = null;

function t(key) { return PathProbe.Locale.t(key); }

/**
 * Escape a value for safe insertion into HTML innerHTML.
 * All dynamic content from API responses passes through this function
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
        '<div class="key">' + esc(k)            + '</div>' +
        '<div class="val">' + esc(v || '\u2014') + '</div>' +
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
      '<td>' + esc(String(h.TTL)) + '</td>' +
      '<td>' + ipCell             + '</td>' +
      '<td>' + asnCell            + '</td>' +
      '<td>' + esc(country)       + '</td>' +
      '<td>' + loss               + '</td>' +
      '<td>' + rtt                + '</td>' +
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
      '<td><strong>' + esc(String(p.Port))               + '</strong></td>' +
      '<td>'         + esc(String(p.Sent))                + '</td>' +
      '<td>'         + esc(String(p.Received))            + '</td>' +
      '<td>'         + esc((p.LossPct || 0).toFixed(1))  + '%</td>' +
      '<td>'         + esc(p.MinRTT)                      + '</td>' +
      '<td>'         + esc(p.AvgRTT)                      + '</td>' +
      '<td>'         + esc(p.MaxRTT)                      + '</td>' +
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

/** Re-render the last report in the current locale (called by applyLocale). */
function rerenderLast() {
  if (_lastReport) renderReport(_lastReport);
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.Renderer = { renderReport, rerenderLast, esc };
