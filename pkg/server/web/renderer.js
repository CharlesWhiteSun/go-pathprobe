'use strict';
// ── renderer.js — report rendering module ────────────────────────────────
// Extracted from app.js (sub-task 3.6).
// Exposes: PathProbe.Renderer = { renderReport, rerenderLast }
// All internal helpers (esc, renderSummary, render*Section, renderGeoBlock)
// are private to this IIFE and do not leak into the global scope.
(() => {

  // Locale helper (runtime-resolved so locale.js may be loaded in any order).
  function _t(key) {
    return (window.PathProbe && window.PathProbe.Locale && window.PathProbe.Locale.t)
      ? window.PathProbe.Locale.t(key)
      : key;
  }

  // Private state — last rendered report, kept for rerenderLast().
  let _lastReport = null;

  // ── Private: HTML escape (XSS protection) ──────────────────────────────
  function esc(s) {
    return String(s)
      .replace(/&/g,  '&amp;')
      .replace(/</g,  '&lt;')
      .replace(/>/g,  '&gt;')
      .replace(/"/g,  '&quot;')
      .replace(/'/g,  '&#39;');
  }

  // ── Private: section renderers ─────────────────────────────────────────

  function renderSummary(r) {
    const items = [
      [_t('key-target'),    r.Target],
      [_t('key-host'),      r.Host],
      [_t('key-generated'), r.GeneratedAt],
    ];
    if (r.PublicGeo && r.PublicGeo.IP) {
      items.push([_t('key-public-ip'), r.PublicGeo.IP]);
    }
    return '<div class="results-summary">' +
      items.map(([k, v]) =>
        '<div class="summary-item">' +
          '<div class="key">'  + esc(k)            + '</div>' +
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
      '<h3>' + esc(_t('section-route')) + '</h3>' +
      '<table class="result-table route-table">' +
        '<thead><tr>' +
          '<th>' + esc(_t('th-ttl'))     + '</th>' +
          '<th>' + esc(_t('th-ip-host')) + '</th>' +
          '<th>' + esc(_t('th-asn'))     + '</th>' +
          '<th>' + esc(_t('th-country')) + '</th>' +
          '<th>' + esc(_t('th-loss'))    + '</th>' +
          '<th>' + esc(_t('th-avg-rtt')) + '</th>' +
        '</tr></thead>' +
        '<tbody>' + rows + '</tbody>' +
      '</table></div>';
  }

  function renderPortsSection(ports) {
    if (!ports || ports.length === 0) return '';
    const rows = ports.map(p =>
      '<tr>' +
        '<td><strong>' + esc(String(p.Port))             + '</strong></td>' +
        '<td>'         + esc(String(p.Sent))             + '</td>' +
        '<td>'         + esc(String(p.Received))         + '</td>' +
        '<td>'         + esc((p.LossPct || 0).toFixed(1)) + '%</td>' +
        '<td>'         + esc(p.MinRTT)                   + '</td>' +
        '<td>'         + esc(p.AvgRTT)                   + '</td>' +
        '<td>'         + esc(p.MaxRTT)                   + '</td>' +
      '</tr>'
    ).join('');
    return '<div class="result-section">' +
      '<h3>' + esc(_t('section-ports')) + '</h3>' +
      '<table class="result-table">' +
        '<thead><tr>' +
          '<th>' + esc(_t('th-port'))    + '</th>' +
          '<th>' + esc(_t('th-sent'))    + '</th>' +
          '<th>' + esc(_t('th-recv'))    + '</th>' +
          '<th>' + esc(_t('th-loss'))    + '</th>' +
          '<th>' + esc(_t('th-min-rtt')) + '</th>' +
          '<th>' + esc(_t('th-avg-rtt')) + '</th>' +
          '<th>' + esc(_t('th-max-rtt')) + '</th>' +
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
      '<h3>' + esc(_t('section-protos')) + '</h3>' +
      '<table class="result-table">' +
        '<thead><tr>' +
          '<th>' + esc(_t('th-protocol')) + '</th>' +
          '<th>' + esc(_t('th-host'))     + '</th>' +
          '<th>' + esc(_t('th-port'))     + '</th>' +
          '<th>' + esc(_t('th-status'))   + '</th>' +
          '<th>' + esc(_t('th-summary'))  + '</th>' +
        '</tr></thead>' +
        '<tbody>' + rows + '</tbody>' +
      '</table></div>';
  }

  // Render DNS comparison results as a card-per-entry layout.
  // Each card groups all per-resolver answers under a single header that shows
  // the domain name, record-type badge, and the five-state consistency badge.
  //
  // Design rationale:
  //   • Domain + Type identity belongs at the GROUP level (card header),
  //     not as repeated table columns — eliminates empty-cell rows.
  //   • The consistency/error status is shown once in the card header,
  //     decoupled from the resolver-detail table below it.
  //   • The inner table has exactly 3 columns (Resolver | Records | RTT).
  //     Error category badges appear in the Resolver column (next to the
  //     resolver name) rather than in Records, keeping Records for actual
  //     DNS record values only.
  //
  // Five-state badge priority (highest → lowest):
  //   AllFailed     → badge-fail  + dns-all-failed   (every resolver errored)
  //   HasDivergence → badge-fail  + dns-divergent    (resolvers disagree)
  //   NoneFound     → badge-warn  + dns-no-records   (no records: mix of errors+empty)
  //   AllEmpty      → badge-warn  + dns-no-records   (no records, no errors — subset of NoneFound)
  //   consistent    → badge-ok    + dns-consistent   (resolvers agree on non-empty data)
  function renderDNSSection(dnsEntries) {
    if (!dnsEntries || dnsEntries.length === 0) return '';

    // Lookup table: ErrorCategory string → i18n key for the resolver-column badge.
    const _errCatKey = {
      'input':    'dns-cat-input',
      'nxdomain': 'dns-cat-nxdomain',
      'network':  'dns-cat-network',
      'resolver': 'dns-cat-resolver',
    };

    const groups = dnsEntries.map(entry => {
      // ── Five-state status badge ──────────────────────────────────────────
      // AllFailed and NoneFound are checked before HasDivergence so that
      // "all errors" and "no records at all" are labelled correctly rather
      // than falling through to "Consistent".
      let badge;
      if (entry.AllFailed) {
        badge = '<span class="badge badge-fail">' + esc(_t('dns-all-failed'))  + '</span>';
      } else if (entry.HasDivergence) {
        badge = '<span class="badge badge-fail">' + esc(_t('dns-divergent'))   + '</span>';
      } else if (entry.NoneFound || entry.AllEmpty) {
        badge = '<span class="badge badge-warn">' + esc(_t('dns-no-records'))  + '</span>';
      } else {
        badge = '<span class="badge badge-ok">'   + esc(_t('dns-consistent'))  + '</span>';
      }

      // ── Card header: domain | record-type pill | status badge ────────────
      const header =
        '<div class="dns-group-header">' +
          '<span class="dns-group-domain">' + esc(entry.Domain) + '</span>' +
          '<span class="dns-group-type">'   + esc(entry.Type)   + '</span>' +
          badge +
        '</div>';

      // ── Per-resolver answer rows (3 columns: Resolver | Records | RTT) ───
      //
      // Error category badge lives in the Resolver column alongside the
      // resolver name, because it describes the resolver's behaviour — not
      // the record content.  The Records column shows only actual DNS values
      // or a dash when nothing was returned.
      const answerRows = (entry.Answers || []).map(ans => {
        // Resolver cell: source name + optional error-category badge (tooltip
        // carries the raw technical error string for debugging).
        let resolverCell = esc(ans.Source);
        if (ans.LookupError) {
          const catKey = _errCatKey[ans.ErrorCategory] || 'dns-cat-unknown';
          resolverCell +=
            ' <span class="badge badge-fail dns-err-label" title="' +
            esc(ans.LookupError) + '">' + esc(_t(catKey)) + '</span>';
        }

        // Records cell: actual DNS values only.  Errors and empty results
        // both show a dash — error detail is already in the Resolver column.
        let recordsCell;
        if (ans.Values && ans.Values.length) {
          recordsCell = ans.Values
            .map(v => '<span class="dns-record-value">' + esc(v) + '</span>')
            .join('');
        } else {
          recordsCell = '<span class="dns-no-value">\u2014</span>';
        }

        // RTT is meaningless when the resolver errored — hide it for clarity.
        const rttCell = ans.LookupError ? '\u2014' : esc(ans.RTT);

        return '<tr class="dns-answer-row">' +
          '<td class="dns-resolver">' + resolverCell + '</td>' +
          '<td class="dns-records">'  + recordsCell  + '</td>' +
          '<td class="dns-rtt">'      + rttCell      + '</td>' +
        '</tr>';
      }).join('');

      // ── Contextual hint banner (inside card, below resolver table) ────────
      // Shown when the Go layer computed a HintKey (always when AllFailed).
      // Uses a <div> rather than a table row so hint styling is independent
      // of the resolver table's column structure.
      const hintHtml = entry.HintKey
        ? '<div class="dns-hint">' + esc(_t(entry.HintKey)) + '</div>'
        : '';

      return '<div class="dns-group">' +
        header +
        '<table class="dns-answer-table">' +
          '<thead><tr>' +
            '<th>' + esc(_t('th-dns-resolver')) + '</th>' +
            '<th>' + esc(_t('th-dns-records'))  + '</th>' +
            '<th>' + esc(_t('th-dns-rtt'))      + '</th>' +
          '</tr></thead>' +
          '<tbody>' + answerRows + '</tbody>' +
        '</table>' +
        hintHtml +
      '</div>';
    }).join('');

    return '<div class="result-section">' +
      '<h3>' + esc(_t('section-dns')) + '</h3>' +
      '<div class="dns-groups">' + groups + '</div>' +
    '</div>';
  }

  function renderGeoSection(pub, tgt) {
    const hasAny = (pub && pub.HasLocation) || (tgt && tgt.HasLocation);
    if (!hasAny) return '';
    return '<div class="result-section">' +
      '<h3>' + esc(_t('section-geo')) + '</h3>' +
      '<div class="geo-grid">' +
        renderGeoBlock(_t('geo-public-ip'),   pub) +
        renderGeoBlock(_t('geo-target-host'), tgt) +
      '</div></div>';
  }

  function renderGeoBlock(label, geo) {
    if (!geo || !geo.IP) {
      return '<div class="geo-block"><h4>' + esc(label) + '</h4>' +
             '<p class="empty-note">' + esc(_t('geo-no-data')) + '</p></div>';
    }
    const rows = [
      [_t('geo-kv-ip'),      geo.IP],
      geo.City        ? [_t('geo-kv-city'),    geo.City]                                         : null,
      geo.CountryName ? [_t('geo-kv-country'), geo.CountryName + ' (' + geo.CountryCode + ')']  : null,
      geo.OrgName     ? [_t('geo-kv-asn'),     'AS' + geo.ASN + ' ' + geo.OrgName]              : null,
    ].filter(Boolean);

    const kvHtml = rows.map(([k, v]) =>
      '<span class="k">' + esc(k) + '</span><span>' + esc(v) + '</span>'
    ).join('');

    return '<div class="geo-block">' +
      '<h4>' + esc(label) + '</h4>' +
      '<div class="geo-kv">' + kvHtml + '</div>' +
    '</div>';
  }

  // ── Public: main render entry point ────────────────────────────────────

  function renderReport(r) {
    _lastReport = r;
    document.getElementById('results-inner').innerHTML = [
      renderSummary(r),
      renderPortsSection(r.Ports),
      renderProtosSection(r.Protos),
      renderDNSSection(r.DNS),
      renderRouteSection(r.Route),
      renderGeoSection(r.PublicGeo, r.TargetGeo),
    ].filter(Boolean).join('');
  }

  // ── Public: re-render last report after a locale change ────────────────

  function rerenderLast() {
    if (_lastReport) renderReport(_lastReport);
  }

  // ── Export ─────────────────────────────────────────────────────────────
  const _ns = window.PathProbe || {};
  _ns.Renderer = { renderReport, rerenderLast };
  window.PathProbe = _ns;
})();
