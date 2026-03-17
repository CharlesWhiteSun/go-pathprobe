'use strict';
// ── api-builder.js — API request construction (PathProbe.ApiBuilder) ──────
// Depends on: config.js (PathProbe.Config), form.js (PathProbe.Form)
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

/**
 * Parse a Go duration string (e.g. "30s", "2m") into whole seconds.
 * Returns defaultSec when the string is empty or unparseable.
 */
function parseTimeoutSec(s, defaultSec) {
  if (!s) return defaultSec;
  const m = s.match(/^(\d+(?:\.\d+)?)(s|m)$/);
  if (!m) return defaultSec;
  const v = parseFloat(m[1]);
  return m[2] === 'm' ? v * 60 : v;
}

function buildWebOpts() {
  const getModeFor = PathProbe.Form.getModeFor;
  const val        = PathProbe.Form.val;
  const checked    = PathProbe.Form.checked;
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
  const { getModeFor, val, checked } = PathProbe.Form;
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
  const { getModeFor, val, checked } = PathProbe.Form;
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
  const { getModeFor, val } = PathProbe.Form;
  const mode = getModeFor('sftp') || 'auth';
  return {
    mode,
    username: val('sftp-user'),
    password: val('sftp-pass'),
  };
}

function buildRequest() {
  const { WEB_MODES_WITH_PORTS } = PathProbe.Config;
  const { val, checked, getModeFor } = PathProbe.Form;
  const target   = val('target');
  const mtrCount = Math.max(1, parseInt(val('mtr-count'), 10) || 5);
  const timeout  = val('diag-timeout') || '30s';
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
      ports = [];
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

  // For Route Trace, ensure the request timeout covers the worst-case time.
  if (target === 'web' && getModeFor('web') === 'traceroute') {
    const maxHops = parseInt(val('traceroute-max-hops'), 10) || 30;
    const minSec  = maxHops * mtrCount * 2 + 15;
    if (parseTimeoutSec(opts.timeout, 30) < minSec) {
      opts.timeout = minSec + 's';
    }
  }

  return { target, options: opts };
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.ApiBuilder = { buildRequest, parseTimeoutSec };
