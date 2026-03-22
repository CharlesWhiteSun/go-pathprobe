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

  // Threshold for collapsing DNS record values.  When a resolver returns more
  // than this many records, the excess is hidden behind an expand toggle.
  // Kept as a named constant so callers, tests and CSS class names all share
  // one source of truth without hard-coding a magic number.
  const DNS_VALUE_COLLAPSE_THRESHOLD = 3;

  // ── Private: _renderDnsValues ─────────────────────────────────────────
  // Renders DNS record values as inline pills.  When the list exceeds
  // DNS_VALUE_COLLAPSE_THRESHOLD the extras are wrapped in a collapsible
  // <span> and a toggle <button> is appended.  Event handling is done via
  // event delegation in form.js (no inline handlers → CSP-safe).
  function _renderDnsValues(values) {
    if (!values || !values.length) {
      return '<span class="dns-no-value">—</span>';
    }
    if (values.length <= DNS_VALUE_COLLAPSE_THRESHOLD) {
      return values.map(v => '<span class="dns-record-value">' + esc(v) + '</span>').join('');
    }
    const visible   = values.slice(0, DNS_VALUE_COLLAPSE_THRESHOLD);
    const overflow  = values.slice(DNS_VALUE_COLLAPSE_THRESHOLD);
    const moreLabel = _t('dns-records-more').replace('{n}', overflow.length);
    const lessLabel = _t('dns-records-less');
    return (
      visible.map(v => '<span class="dns-record-value">' + esc(v) + '</span>').join('') +
      '<span class="dns-records-overflow">' +
        overflow.map(v => '<span class="dns-record-value">' + esc(v) + '</span>').join('') +
      '</span>' +
      '<button type="button" class="dns-records-toggle" aria-expanded="false" ' +
        'data-label-more="' + esc(moreLabel) + '" ' +
        'data-label-less="' + esc(lessLabel) + '">' +
        esc(moreLabel) +
      '</button>'
    );
  }

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

  // ── Private: _hopIpBadge ─────────────────────────────────────────────────
  // Returns an HTML badge string for private / loopback / link-local IPs,
  // or an empty string for public IPs.  Delegates classification to the
  // HopClassifier module so this function stays free of IP-range literals.
  function _hopIpBadge(ip) {
    // Scope CSS classes (listed for tooling): hop-ip-badge--private hop-ip-badge--loopback hop-ip-badge--link-local
    const clf = window.PathProbe && window.PathProbe.HopClassifier;
    if (!clf) return '';
    const scope = clf.classifyIP(ip);
    if (!scope) return '';
    return ' <span class="hop-ip-badge hop-ip-badge--' + scope + '">' +
      esc(_t('hop-type-' + scope)) +
    '</span>';
  }

  // ── Private: _hopTypeTags ─────────────────────────────────────────────────
  // Returns zero or more classification badge spans for the "Type" column.
  // Delegates to HopClassifier.classifyIPTags so all range logic lives in
  // hop-classifier.js.  Each tag key maps to 'hop-type-{key}' in i18n and
  // 'hop-ip-badge--{key}' in CSS.
  //
  // Extended CSS classes referenced here (for tooling / grep):
  //   hop-ip-badge--class-a  hop-ip-badge--class-b  hop-ip-badge--class-c
  //   hop-ip-badge--class-d  hop-ip-badge--class-e  hop-ip-badge--public
  //   hop-ip-badge--cgnat    hop-ip-badge--multicast hop-ip-badge--reserved
  //   hop-ip-badge--broadcast hop-ip-badge--documentation hop-ip-badge--6to4-relay
  //   hop-ip-badge--ipv6
  function _hopTypeTags(ip) {
    const clf = window.PathProbe && window.PathProbe.HopClassifier;
    if (!clf || !clf.classifyIPTags) return '';
    const tags = clf.classifyIPTags(ip);
    if (!tags || !tags.length) return '';
    return tags.map(function(key) {
      return '<span class="hop-ip-badge hop-ip-badge--' + key + '">' +
        esc(_t('hop-type-' + key)) + '</span>';
    }).join(' ');
  }

  // ── Private: renderRouteStats ─────────────────────────────────────────────
  // Renders a compact statistics card below the hop table summarising the
  // overall route quality: total hops, responsive vs. silent, average loss,
  // terminal RTT, countries traversed, and whether the destination was reached.
  function renderRouteStats(hops) {
    if (!hops || !hops.length) return '';

    const total      = hops.length;
    const responsive = hops.filter(h => h.IP).length;
    const timedout   = total - responsive;
    const avgLoss    = hops.reduce((s, h) => s + (h.LossPct || 0), 0) / total;

    // Terminal RTT: AvgRTT of the highest TTL that actually responded.
    const lastOk = hops.slice().reverse().find(h => h.IP && h.AvgRTT && h.AvgRTT !== '\u2014');
    const termRTT = lastOk ? lastOk.AvgRTT : '\u2014';

    // Unique country codes seen across all hops (geo annotation, may be empty).
    const countries = [];
    hops.forEach(h => { if (h.Country && !countries.includes(h.Country)) countries.push(h.Country); });

    // Destination reached when the last hop has a non-empty IP.
    const reached = !!(hops[hops.length - 1].IP);
    const reachedBadge = reached
      ? '<span class="badge badge-ok">'   + esc(_t('route-stats-reached'))     + '</span>'
      : '<span class="badge badge-warn">' + esc(_t('route-stats-not-reached')) + '</span>';

    const statItems = [
      { label: _t('route-stats-total'),      html: esc(String(total)) },
      { label: _t('route-stats-responsive'), html: esc(String(responsive)) },
      { label: _t('route-stats-timeout'),    html: esc(String(timedout)) },
      { label: _t('route-stats-avg-loss'),   html: esc(avgLoss.toFixed(1)) + '%' },
      { label: _t('route-stats-max-rtt'),    html: esc(termRTT) },
      { label: _t('route-stats-countries'),  html: countries.length ? esc(countries.join(', ')) : '\u2014' },
      { label: _t('route-stats-reached'),    html: reachedBadge },
    ];

    const itemsHtml = statItems.map(item =>
      '<div class="route-stat-item">' +
        '<span class="route-stat-label">' + esc(item.label) + '</span>' +
        '<span class="route-stat-value">' + item.html       + '</span>' +
      '</div>'
    ).join('');

    return '<div class="route-stats-card">' +
      '<h4 class="route-stats-title">' + esc(_t('route-stats-title')) + '</h4>' +
      '<div class="route-stats-grid">' + itemsHtml + '</div>' +
    '</div>';
  }

  function renderRouteSection(hops) {
    if (!hops || !hops.length) return '';
    const tipText = _t('hop-timeout-tip');
    const rows = hops.map(h => {
      const timedout  = !h.IP;
      const rowClass  = timedout ? ' class="hop-timedout"' : '';

      // ── IP column: clean address; ??? for timed-out hops ─────────────────
      const ipCell = timedout
        ? '<em class="hop-timeout-marker" title="' + esc(tipText) + '">???</em>'
        : esc(h.IP);

      // ── Type column: multi-tag classification badges ───────────────────
      const typeCell = timedout ? '' : _hopTypeTags(h.IP);

      // ── Hostname column: separate from IP ──────────────────────────────
      const hostCell = (!h.Hostname || h.Hostname === h.IP) ? '\u2014' : esc(h.Hostname);

      const asnCell = h.ASN ? 'AS' + esc(String(h.ASN)) : '';
      const country = h.Country || '\u2014';

      // Loss% is always shown numerically (timedout hops report 100.0% from the backend).
      const loss = (h.LossPct || 0).toFixed(1) + '%';
      const rtt  = timedout || !h.AvgRTT ? '\u2014' : esc(h.AvgRTT);

      return '<tr' + rowClass + '>' +
        '<td>'                        + esc(String(h.TTL)) + '</td>' +
        '<td>'                        + ipCell              + '</td>' +
        '<td class="hop-type-col">'   + typeCell            + '</td>' +
        '<td>'                        + hostCell            + '</td>' +
        '<td>'                        + asnCell             + '</td>' +
        '<td>'                        + esc(country)        + '</td>' +
        '<td class="hop-loss-col">'   + loss                + '</td>' +
        '<td>'                        + rtt                 + '</td>' +
      '</tr>';
    }).join('');
    return '<div class="result-section">' +
      '<h3>' + esc(_t('section-route')) + '</h3>' +
      '<table class="result-table route-table">' +
        '<thead><tr>' +
          '<th>' + esc(_t('th-ttl'))      + '</th>' +
          '<th>' + esc(_t('th-ip'))       + '</th>' +
          '<th>' + esc(_t('th-type'))     + '</th>' +
          '<th>' + esc(_t('th-hostname')) + '</th>' +
          '<th>' + esc(_t('th-asn'))      + '</th>' +
          '<th>' + esc(_t('th-country'))  + '</th>' +
          '<th>' + esc(_t('th-loss'))     + '</th>' +
          '<th>' + esc(_t('th-avg-rtt'))  + '</th>' +
        '</tr></thead>' +
        '<tbody>' + rows + '</tbody>' +
      '</table>' +
      renderRouteStats(hops) +
    '</div>';
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
        // Long value lists are collapsed via _renderDnsValues() to prevent
        // the Records column from overflowing and pushing RTT out of view.
        const recordsCell = _renderDnsValues(ans.Values);

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

  // ── Traceroute live progress helpers ───────────────────────────────────

  let _trStartTime = 0;
  let _trMaxHops   = 30;
  let _trHopCount  = 0;

  /**
   * Show the traceroute progress panel and initialise its fields.
   * @param {string} host       - destination host being traced
   * @param {number} maxHops    - configured max TTL
   * @param {number} mtrCount   - probes per hop (used for ETA estimate)
   */
  function initTracerouteProgress(host, maxHops, mtrCount) {
    _trStartTime = Date.now();
    _trMaxHops   = maxHops || 30;
    _trHopCount  = 0;

    const el = document.getElementById('traceroute-progress');
    if (!el) return;

    const title = _t('traceroute-progress-title');
    const trTitle = document.getElementById('tr-title');
    if (trTitle) trTitle.textContent = host ? title + ' ' + host : title;

    const trHopCount = document.getElementById('tr-hop-count');
    if (trHopCount) trHopCount.textContent = '';

    const trBar = document.getElementById('tr-bar');
    if (trBar) trBar.style.width = '0%';

    const estMin = Math.ceil((_trMaxHops * (mtrCount || 5) * 2 + 15) / 60);
    const trEta = document.getElementById('tr-eta');
    if (trEta) trEta.textContent = _t('traceroute-max-wait').replace('{n}', estMin);

    const trBody = document.getElementById('tr-live-body');
    if (trBody) trBody.innerHTML = '';

    const trLiveSection = document.getElementById('tr-live-section');
    if (trLiveSection) trLiveSection.hidden = true;

    // Restore spinner visibility in case a previous finalizeTracerouteProgress hid it.
    const spinner = el.querySelector('.tr-spin');
    if (spinner) spinner.hidden = false;

    el.hidden = false;
  }

  /**
   * Append a single hop row to the live progress table and advance the bar.
   * @param {Object} hopData - HopProgressData JSON from the backend
   */
  function appendLiveHop(hopData) {
    if (!hopData) return;
    _trHopCount++;

    const pct = Math.min(100, Math.round(_trHopCount / (_trMaxHops || 30) * 100));
    const trBar = document.getElementById('tr-bar');
    if (trBar) trBar.style.width = pct + '%';

    const counter = document.getElementById('tr-hop-count');
    if (counter) counter.textContent = _t('traceroute-hop-count').replace('{n}', _trHopCount);

    const tbody = document.getElementById('tr-live-body');
    if (!tbody) return;

    const timedout = !hopData.ip;
    const row = document.createElement('tr');
    if (timedout) row.className = 'hop-timedout';

    // IP cell: clean address; timeout → ???
    let ipCell;
    if (timedout) {
      ipCell = '<em class="hop-timeout-marker" title="' + esc(_t('hop-timeout-tip')) + '">???</em>';
    } else {
      ipCell = esc(hopData.ip);
    }

    // Type cell: multi-tag classification badges (empty for timed-out hops)
    const typeCell = timedout ? '' : _hopTypeTags(hopData.ip);

    // Hostname cell: separate column; dash when absent or same as IP
    const hostCell = (hopData.hostname && hopData.hostname !== hopData.ip)
      ? esc(hopData.hostname)
      : '\u2014';

    // Loss% is always numeric (timedout hops report 100.0 from the backend)
    const loss = (hopData.loss_pct || 0).toFixed(1) + '%';
    const rtt  = timedout ? '\u2014' : esc(hopData.avg_rtt || '\u2014');
    row.innerHTML = '<td>' + esc(String(hopData.ttl)) + '</td>' +
      '<td>' + ipCell    + '</td>' +
      '<td>' + typeCell  + '</td>' +
      '<td>' + hostCell  + '</td>' +
      '<td>' + loss      + '</td>' +
      '<td>' + rtt       + '</td>';
    tbody.appendChild(row);

    const liveSection = document.getElementById('tr-live-section');
    if (liveSection) liveSection.hidden = false;
    tbody.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }

  /**
   * Transition the traceroute progress panel to a final state without hiding it.
   * The spinner is replaced by a status message; the live hop table remains visible.
   * Called when the trace ends naturally (timeout) or the backend reports context canceled.
   * @param {string} titleKey - i18n key for the status title (e.g. 'traceroute-timeout')
   */
  function finalizeTracerouteProgress(titleKey) {
    const el = document.getElementById('traceroute-progress');
    if (!el || el.hidden) return;

    // Hide the spinner — no longer in-progress.
    const spinner = el.querySelector('.tr-spin');
    if (spinner) spinner.hidden = true;

    // Update title to reflect the final status.
    const trTitle = document.getElementById('tr-title');
    if (trTitle) trTitle.textContent = _t(titleKey).replace('{n}', _trHopCount);

    // Replace ETA with "N hops recorded" summary.
    const trEta = document.getElementById('tr-eta');
    if (trEta) trEta.textContent = _t('traceroute-hop-count').replace('{n}', _trHopCount);
  }

  /**
   * Hide the traceroute progress panel (called when result arrives or on error).
   */
  function hideTracerouteProgress() {
    const el = document.getElementById('traceroute-progress');
    if (el) el.hidden = true;
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
  _ns.Renderer = { renderReport, rerenderLast, initTracerouteProgress, appendLiveHop, hideTracerouteProgress, finalizeTracerouteProgress };
  window.PathProbe = _ns;
})();
