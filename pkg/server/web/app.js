'use strict';

// ── Per-target default ports ──────────────────────────────────────────────
const TARGET_PORTS = {
  web:  [443],
  smtp: [25, 465, 587],
  imap: [143, 993],
  pop:  [110, 995],
  ftp:  [21, 990],
  sftp: [22],
};

// ── Per-target host placeholder text ─────────────────────────────────────
const TARGET_PLACEHOLDERS = {
  web:  'e.g. google.com',
  smtp: 'e.g. mail.example.com',
  imap: 'e.g. mail.example.com',
  pop:  'e.g. mail.example.com',
  ftp:  'e.g. ftp.example.com',
  sftp: 'e.g. sftp.example.com',
};

// ── Initialisation ────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  // Track whether the user has manually edited the auto-filled fields.
  ['host', 'ports'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('input', () => { el.dataset.userEdited = 'true'; });
  });

  onTargetChange(); // populate defaults for initial selection
  fetchVersion();   // async version badge
});

// ── Form dynamics ─────────────────────────────────────────────────────────
function onTargetChange() {
  const target = val('target');

  // Show the fieldset for the current target; hide all others.
  document.querySelectorAll('.target-fields').forEach(fs => {
    fs.hidden = (fs.id !== 'fields-' + target);
  });

  // Auto-fill ports only when the user has not overridden them.
  const portEl = document.getElementById('ports');
  if (portEl && portEl.dataset.userEdited !== 'true') {
    portEl.value = (TARGET_PORTS[target] || []).join(', ');
  }

  // Update host placeholder based on target.
  const hostEl = document.getElementById('host');
  if (hostEl) {
    hostEl.placeholder = TARGET_PLACEHOLDERS[target] || 'hostname or IP';
  }
}

// ── Request building ──────────────────────────────────────────────────────
function buildRequest() {
  const target   = val('target');
  const mtrCount = Math.max(1, parseInt(val('mtr-count'), 10) || 5);
  const timeout  = val('diag-timeout') || '30s';
  const insecure = checked('insecure');
  const ports    = val('ports')
    .split(',')
    .map(s => parseInt(s.trim(), 10))
    .filter(n => n > 0 && n <= 65535);

  const opts = {
    mtr_count: mtrCount,
    timeout,
    insecure,
    net: { host: val('host'), ports },
  };

  switch (target) {
    case 'web':  Object.assign(opts, { web:  buildWebOpts()  }); break;
    case 'smtp': Object.assign(opts, { smtp: buildSMTPOpts() }); break;
    case 'ftp':  Object.assign(opts, { ftp:  buildFTPOpts()  }); break;
    case 'sftp': Object.assign(opts, { sftp: buildSFTPOpts() }); break;
    // imap / pop: no protocol-specific options beyond net
  }

  return { target, options: opts };
}

function buildWebOpts() {
  const types   = ['A', 'AAAA', 'MX'].filter(t => checked('dns-' + t));
  const domains = val('dns-domains').split(',').map(s => s.trim()).filter(Boolean);
  return { domains, types, url: val('http-url') };
}

function buildSMTPOpts() {
  return {
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
  return {
    username: val('ftp-user'),
    password: val('ftp-pass'),
    use_tls:  checked('ftp-ssl'),
    auth_tls: checked('ftp-auth-tls'),
    run_list: checked('ftp-list'),
  };
}

function buildSFTPOpts() {
  return {
    username: val('sftp-user'),
    password: val('sftp-pass'),
    run_ls:   checked('sftp-ls'),
  };
}

// ── API calls ─────────────────────────────────────────────────────────────
async function runDiag() {
  const btn      = document.getElementById('run-btn');
  const errorEl  = document.getElementById('error-banner');
  const resultEl = document.getElementById('results');

  btn.disabled    = true;
  btn.innerHTML   = '<span class="spinner"></span>Running\u2026';
  errorEl.hidden  = true;
  resultEl.hidden = true;

  try {
    const req  = buildRequest();
    const resp = await fetch('/api/diag', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify(req),
    });
    const data = await resp.json();
    if (!resp.ok) {
      showError(data.error || 'Server returned HTTP ' + resp.status);
      return;
    }
    renderReport(data);
    resultEl.hidden = false;
    resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
  } catch (err) {
    showError('Request failed: ' + err.message);
  } finally {
    btn.disabled  = false;
    btn.innerHTML = '&#9654; Run Diagnostic';
  }
}

async function fetchVersion() {
  try {
    const r = await fetch('/api/health');
    if (!r.ok) return;
    const { version, built_at } = await r.json();
    const el = document.getElementById('version-badge');
    if (el) {
      el.textContent = version +
        (built_at && built_at !== 'unknown' ? '  \u00b7  ' + built_at : '');
    }
  } catch (_) { /* non-fatal — version badge stays empty */ }
}

function showError(msg) {
  const el   = document.getElementById('error-banner');
  el.textContent = '\u26a0  ' + msg;
  el.hidden  = false;
}

// ── Report rendering ──────────────────────────────────────────────────────
function renderReport(r) {
  document.getElementById('results-inner').innerHTML = [
    renderSummary(r),
    renderPortsSection(r.Ports),
    renderProtosSection(r.Protos),
    renderGeoSection(r.PublicGeo, r.TargetGeo),
  ].filter(Boolean).join('');
}

function renderSummary(r) {
  const items = [
    ['Target',    r.Target],
    ['Host',      r.Host],
    ['Generated', r.GeneratedAt],
  ];
  if (r.PublicGeo && r.PublicGeo.IP) {
    items.push(['Public IP', r.PublicGeo.IP]);
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
    '<h3>Port Connectivity</h3>' +
    '<table class="result-table">' +
      '<thead><tr>' +
        '<th>Port</th><th>Sent</th><th>Recv</th>' +
        '<th>Loss%</th><th>Min RTT</th><th>Avg RTT</th><th>Max RTT</th>' +
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
    '<h3>Protocol Results</h3>' +
    '<table class="result-table">' +
      '<thead><tr>' +
        '<th>Protocol</th><th>Host</th><th>Port</th><th>Status</th><th>Summary</th>' +
      '</tr></thead>' +
      '<tbody>' + rows + '</tbody>' +
    '</table></div>';
}

function renderGeoSection(pub, tgt) {
  const hasAny = (pub && pub.HasLocation) || (tgt && tgt.HasLocation);
  if (!hasAny) return '';
  return '<div class="result-section">' +
    '<h3>Geo Information</h3>' +
    '<div class="geo-grid">' +
      renderGeoBlock('Public IP',    pub) +
      renderGeoBlock('Target Host',  tgt) +
    '</div></div>';
}

function renderGeoBlock(label, geo) {
  if (!geo || !geo.IP) {
    return '<div class="geo-block"><h4>' + esc(label) + '</h4>' +
           '<p class="empty-note">No data</p></div>';
  }
  const rows = [
    ['IP',      geo.IP],
    geo.City        ? ['City',    geo.City]                                        : null,
    geo.CountryName ? ['Country', geo.CountryName + ' (' + geo.CountryCode + ')'] : null,
    geo.OrgName     ? ['ASN',     'AS' + geo.ASN + ' ' + geo.OrgName]             : null,
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

/** Read and trim the string value of a form element by id. */
function val(id) {
  const el = document.getElementById(id);
  return el ? el.value.trim() : '';
}

/** Return the checked state of a checkbox by id. */
function checked(id) {
  const el = document.getElementById(id);
  return el ? el.checked : false;
}

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
