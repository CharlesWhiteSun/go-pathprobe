'use strict';

// ── api-builder.js — request construction module ─────────────────────────────
//
// Responsibility:
//   Build a validated diagnostic request object from the current form state.
//   All protocol-specific option objects are assembled here; the caller
//   (runDiag in app.js) only needs to JSON-stringify the result and POST it.
//
// Dependencies (all runtime-resolved — no hard import):
//   PathProbe.Config  — WEB_MODES_WITH_PORTS, TARGET_PORTS (config.js)
//   PathProbe.Form    — val(), checked(), getModeFor()      (form.js)
//
// Public API:
//   PathProbe.ApiBuilder = { buildRequest }
//
// Load order:
//   config.js → locale.js → theme.js → form.js → api-builder.js → app.js

(() => {
  // ── Config aliases (runtime-resolved) ──────────────────────────────────
  function _cfg()               { return (window.PathProbe && window.PathProbe.Config) || {}; }
  function _webModesWithPorts() { return _cfg().WEB_MODES_WITH_PORTS || []; }
  function _targetPorts()       { return _cfg().TARGET_PORTS          || {}; }

  // ── Form accessors (runtime-resolved via PathProbe.Form) ────────────────
  /** Read and trim a form field value. */
  function _val(id) {
    return (window.PathProbe && window.PathProbe.Form)
      ? window.PathProbe.Form.val(id) : '';
  }

  /** Return the checked state of a checkbox. */
  function _checked(id) {
    return (window.PathProbe && window.PathProbe.Form)
      ? window.PathProbe.Form.checked(id) : false;
  }

  /** Return the currently-selected sub-mode radio value for a target. */
  function _getModeFor(target) {
    return (window.PathProbe && window.PathProbe.Form)
      ? window.PathProbe.Form.getModeFor(target) : '';
  }

  // ── Timeout parsing ─────────────────────────────────────────────────────
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

  // ── Protocol-specific option builders ──────────────────────────────────
  /**
   * Build the web-target option object from the current form state.
   * Mode-specific fields (DNS domain list, HTTP URL, traceroute max-hops)
   * are only included when the active sub-mode requires them.
   */
  function buildWebOpts() {
    const mode = _getModeFor('web') || 'public-ip';
    const opts = { mode };
    if (mode === 'dns') {
      opts.domains = _val('dns-domains').split(',').map(s => s.trim()).filter(Boolean);
      opts.types   = ['A', 'AAAA', 'MX'].filter(t => _checked('dns-' + t));
    } else if (mode === 'http') {
      opts.url = _val('http-url');
    } else if (mode === 'traceroute') {
      const maxHops = parseInt(_val('traceroute-max-hops'), 10);
      if (maxHops > 0) opts.max_hops = maxHops;
    }
    return opts;
  }

  /**
   * Build the SMTP-target option object from the current form state.
   * All credential and option fields are included; the backend ignores
   * irrelevant ones depending on the selected mode.
   */
  function buildSMTPOpts() {
    const mode = _getModeFor('smtp') || 'handshake';
    return {
      mode,
      domain:       _val('smtp-domain'),
      username:     _val('smtp-user'),
      password:     _val('smtp-pass'),
      from:         _val('smtp-from'),
      to:           _val('smtp-to').split(',').map(s => s.trim()).filter(Boolean),
      start_tls:    _checked('smtp-starttls'),
      use_tls:      _checked('smtp-ssl'),
      mx_probe_all: _checked('smtp-mx-all'),
    };
  }

  /**
   * Build the FTP-target option object from the current form state.
   */
  function buildFTPOpts() {
    const mode = _getModeFor('ftp') || 'login';
    return {
      mode,
      username: _val('ftp-user'),
      password: _val('ftp-pass'),
      use_tls:  _checked('ftp-ssl'),
      auth_tls: _checked('ftp-auth-tls'),
    };
  }

  /**
   * Build the SFTP-target option object from the current form state.
   */
  function buildSFTPOpts() {
    const mode = _getModeFor('sftp') || 'auth';
    return {
      mode,
      username: _val('sftp-user'),
      password: _val('sftp-pass'),
    };
  }

  // ── Main request builder ────────────────────────────────────────────────
  /**
   * Assemble the full diagnostic request payload from the current form state.
   *
   * Port handling:
   *   - web target + mode in WEB_MODES_WITH_PORTS  → read the shared text input
   *   - web target + other modes (public-ip, dns, http, traceroute) → empty list
   *   - non-web targets → always read the shared text input
   *
   * Traceroute timeout auto-compute:
   *   For web/traceroute the backend performs maxHops × mtrCount round-trips
   *   each with a 2 s hopTimeout, plus a fixed 15 s buffer.  When the user's
   *   chosen timeout is shorter than this worst-case estimate, it is silently
   *   raised to prevent spurious "context deadline exceeded" errors.
   *   Formula: maxHops * mtrCount * 2 + 15
   *
   * @returns {{ target: string, options: object }} Request payload.
   *   Never returns null; validation is left to the backend.
   */
  function buildRequest() {
    const target   = _val('target');
    const mtrCount = Math.max(1, parseInt(_val('mtr-count'), 10) || 5);
    let   timeout  = _val('diag-timeout') || '30s';
    const insecure = _checked('insecure');

    // Determine which ports to include based on target and active sub-mode.
    let ports;
    if (target === 'web') {
      const mode = _getModeFor('web');
      if (_webModesWithPorts().includes(mode)) {
        // ports-text-group: shared text input read path for web/port mode
        ports = _val('ports')
          .split(',')
          .map(s => parseInt(s.trim(), 10))
          .filter(n => n > 0 && n <= 65535);
      } else {
        ports = []; // other web modes (public-ip/dns/http/traceroute) don't use ports
      }
    } else {
      ports = _val('ports')
        .split(',')
        .map(s => parseInt(s.trim(), 10))
        .filter(n => n > 0 && n <= 65535);
    }

    const opts = {
      mtr_count:   mtrCount,
      timeout,
      insecure,
      disable_geo: !_checked('geo-enabled'),
      net: { host: _val('host'), ports },
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
    if (target === 'web' && _getModeFor('web') === 'traceroute') {
      const maxHops = parseInt(_val('traceroute-max-hops'), 10) || 30;
      const minSec  = maxHops * mtrCount * 2 + 15;
      if (parseTimeoutSec(opts.timeout, 30) < minSec) {
        opts.timeout = minSec + 's';
      }
    }

    return { target, options: opts };
  }

  // ── Namespace export ────────────────────────────────────────────────────
  const PathProbe = window.PathProbe || {};
  PathProbe.ApiBuilder = { buildRequest };
  window.PathProbe = PathProbe;
})();
