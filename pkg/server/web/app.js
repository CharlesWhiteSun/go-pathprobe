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

// ── Map point role configurations ─────────────────────────────────────────
// Each entry defines the CSS class and i18n label key used for a geocoded
// point on the Leaflet map.  Adding a new role requires only a new key here —
// renderMap(), buildMarkerIcon(), buildPopupHtml(), and buildMapLegend() all
// read this object so no other code needs to change.
const MAP_POINT_CONFIGS = {
  origin: { cssClass: 'geo-marker--origin', i18nKey: 'map-origin' },
  target: { cssClass: 'geo-marker--target', i18nKey: 'map-target' },
};

// Ordered list of map tile variant identifiers shown as the three buttons above
// the map.  Order determines the left→right button layout.
const MAP_TILE_VARIANTS = ['light', 'osm', 'dark'];

// Maps each application theme to its default map tile variant.
// Only this table needs updating when a new app theme is added.
const MAP_THEME_TO_TILE_VARIANT = {
  'default':       'light',
  'light-green':   'light',
  'deep-blue':     'dark',
  'forest-green':  'dark',
  'dark':          'dark',
};

// Theme IDs that should use the dark tile variant.  All other themes fall back
// to the light/neutral style.  Must stay in sync with THEMES (above).
const MAP_DARK_THEMES = new Set(['dark', 'deep-blue', 'forest-green']);

// Tile layer configurations keyed by variant ('light' | 'osm' | 'dark').
// Using CARTO basemaps for light/dark; osm is the canonical OpenStreetMap style.
// Only change the URLs here if a different provider is desired — no other code
// needs to be touched.
// bgColor: the representative base background colour of each tile set.  Applied
// to the #geo-map container immediately after a tile-layer swap so that any
// not-yet-loaded tile gaps show the expected colour instead of white, preventing
// the white-flash artefact visible when switching to a dark tile variant.
const TILE_LAYER_CONFIGS = {
  light: {
    url:         'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png',
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    i18nKey:     'map-tile-light',
    bgColor:     '#f5f5f0',  // CARTO light_all base background
  },
  osm: {
    url:         'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
    i18nKey:     'map-tile-osm',
    bgColor:     '#f2efe9',  // OpenStreetMap standard base background
  },
  dark: {
    url:         'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png',
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    i18nKey:     'map-tile-dark',
    bgColor:     '#1a1a1a',  // CARTO dark_all base background
  },
};

// ── Per-target default ports ──────────────────────────────────────────────
const TARGET_PORTS = {
  web:  [80, 443],
  smtp: [25, 465, 587],
  imap: [143, 993],
  pop:  [110, 995],
  ftp:  [21, 990],
  sftp: [22],
};

// Maps target → { panelId: [modes that show it] }.
// Only panels that are conditionally visible need an entry here.
const TARGET_MODE_PANELS = {
  web: {
    'web-fields-dns':        ['dns'],
    'web-fields-http':       ['http'],
    'web-fields-traceroute': ['traceroute'],
  },
  smtp: {
    'smtp-fields-auth': ['auth', 'send'],
    'smtp-fields-send': ['send'],
  },
};

// Web modes that require the port-group text input to be shown.
// public-ip/dns/http/traceroute derive ports from protocol defaults or
// do not perform port-level connectivity checks, so they suppress port-group.
const WEB_MODES_WITH_PORTS = ['port'];

// ── Per-target host placeholder i18n keys ─────────────────────────────────
const TARGET_PLACEHOLDER_KEYS = {
  web:  'ph-web',
  smtp: 'ph-smtp',
  imap: 'ph-imap',
  pop:  'ph-pop',
  ftp:  'ph-ftp',
  sftp: 'ph-sftp',
};

// ── Locale management ─────────────────────────────────────────────────────
let _locale = 'en';

// The year this project was first published.  Used to build a copyright range
// that automatically extends as calendar years advance — e.g. "2026" in 2026,
// "2026–2027" in 2027, etc.  Only this constant ever needs to change.
const COPYRIGHT_START_YEAR = 2026;

/** Return the translation for key in the current locale, falling back to en. */
function t(key) {
  const locs = window.LOCALES || {};
  return (locs[_locale] || {})[key] || (locs.en || {})[key] || key;
}

/**
 * Re-write the copyright year after applyLocale() has set the raw i18n text.
 * Builds a range string: just the start year when it equals the current year,
 * otherwise "startYear–currentYear" (en-dash U+2013).  The regex targets the
 * first four-digit year sequence in the copyright text so the logic is locale-
 * independent and requires no changes to the i18n dictionary.
 */
function updateCopyrightYear() {
  const now = new Date().getFullYear();
  const yearStr = now > COPYRIGHT_START_YEAR
    ? COPYRIGHT_START_YEAR + '\u2013' + now
    : String(COPYRIGHT_START_YEAR);
  document.querySelectorAll('[data-i18n="footer-copyright"]').forEach(el => {
    el.textContent = el.textContent.replace(/\d{4}/, yearStr);
  });
}

/** Apply the current locale to all [data-i18n] elements and refresh dynamic UI. */
function applyLocale() {
  document.querySelectorAll('[data-i18n]').forEach(el => {
    el.textContent = t(el.dataset.i18n);
  });
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
    el.placeholder = t(el.dataset.i18nPlaceholder);
  });
  // Refresh host placeholder (depends on current target selection)
  const target = val('target');
  const hostEl = document.getElementById('host');
  if (hostEl) hostEl.placeholder = t(TARGET_PLACEHOLDER_KEYS[target] || 'ph-host-default');
  // Update run button (unless currently running)
  const runBtn = document.getElementById('run-btn');
  if (runBtn && !runBtn.disabled) runBtn.textContent = t('btn-run');
  // Highlight the active language button
  document.querySelectorAll('.lang-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.lang === _locale);
  });
  document.documentElement.lang = _locale;
  // Update footer copyright year range after i18n strings have been applied.
  updateCopyrightYear();
}

/** Persist and apply a new locale choice. */
function setLocale(lang) {
  _locale = lang;
  try { localStorage.setItem('lang', lang); } catch (_) {}
  applyLocale();
}

/** Initialise locale from localStorage (defaults to 'en'). */
function initLocale() {
  try { _locale = localStorage.getItem('lang') || 'en'; } catch (_) { _locale = 'en'; }
  applyLocale();
}

// ── Run-button animation ───────────────────────────────────────────────────
/**
 * Return the innerHTML to inject into #run-btn while a diagnostic is running.
 * Uses the dots animation (three bouncing dots).
 */
function getRunningHTML() {
  return '<span class="anim-dots"><span></span><span></span><span></span></span>';
}

// ── Theme management ──────────────────────────────────────────────────────
/**
 * All valid theme IDs. The CSS file drives the actual appearance; adding a
 * new theme only requires a new [data-theme="id"] block there — no JS change.
 */
const THEMES = ['default', 'deep-blue', 'light-green', 'forest-green', 'dark'];

/**
 * Fallback theme used when (a) no user preference is stored and (b) the HTML
 * data-default-theme attribute is absent or invalid.  Mirrors the value that
 * the server embeds in <html data-default-theme>.
 */
const DEFAULT_THEME = 'default';

/** Apply themeId to <html data-theme> with a targeted fade transition.
 *  Only the .main content area fades out while the theme variables switch.
 *  Header and footer remain fully visible and cross-fade their own
 *  background / text colours via dedicated CSS transitions, so the chrome
 *  always stays on screen during a theme change.
 */
function applyTheme(themeId) {
  const id = THEMES.includes(themeId) ? themeId : DEFAULT_THEME;
  const body = document.body;
  const mainEl = document.querySelector('.main');
  body.classList.add('theme-transitioning');
  // Wait for .main's opacity fade-out to complete before swapping the theme.
  // Using .main as the listener target means only the main-content opacity
  // transition (not header/footer colour transitions) triggers the swap.
  const listenTarget = mainEl || body;
  const onFaded = (e) => {
    if (e.target !== listenTarget || e.propertyName !== 'opacity') return;
    listenTarget.removeEventListener('transitionend', onFaded);
    document.documentElement.dataset.theme = id;
    try { localStorage.setItem('pp-theme', id); } catch (_) {}
    // Highlight the matching dot-button; clear all others.
    document.querySelectorAll('.theme-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.theme === id);
    });
    // Silently swap map tiles while main content is invisible — no map fade needed.
    syncMapTileVariantToTheme(id);
    // Remove the class on the next frame so the fade-in transition fires.
    requestAnimationFrame(() => body.classList.remove('theme-transitioning'));
  };
  listenTarget.addEventListener('transitionend', onFaded);
}

/** Public entry point called by the <select onchange> handler. */
function setTheme(themeId) { applyTheme(themeId); }

/** Restore saved theme from localStorage; fall back to the server-declared
 *  default (data-default-theme on <html>) so a service restart always starts
 *  on the intended theme when no user preference exists.
 *  Applies the theme without the fade animation (page is not yet visible). */
function initTheme() {
  // Server-declared default: read from HTML attribute set by the server.
  const htmlDefault = (document.documentElement.dataset.defaultTheme || '').trim();
  const serverDefault = THEMES.includes(htmlDefault) ? htmlDefault : DEFAULT_THEME;
  let saved = serverDefault;
  try { saved = localStorage.getItem('pp-theme') || serverDefault; } catch (_) {}
  // Apply without animation: set theme vars immediately so there is no flash.
  const id = THEMES.includes(saved) ? saved : DEFAULT_THEME;
  document.documentElement.dataset.theme = id;
  try { localStorage.setItem('pp-theme', id); } catch (_) {}
  document.querySelectorAll('.theme-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.theme === id);
  });
  syncMapTileVariantToTheme(id);
}

// ── Initialisation ────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  // Disable browser spell-check, auto-correct and auto-capitalise on every
  // text input.  All fields contain technical identifiers (hostnames, URLs,
  // ports, credentials) where the browser's natural-language heuristics
  // produce misleading red-underline noise rather than useful feedback.
  // Centralising this here (rather than per-element HTML attributes) means
  // newly added inputs are covered automatically — no per-field opt-out needed.
  document.querySelectorAll('input[type="text"]').forEach(el => {
    el.spellcheck = false;
    el.setAttribute('autocorrect', 'off');
    el.setAttribute('autocapitalize', 'none');
  });

  // Track whether the user has manually edited the auto-filled fields.
  ['host', 'ports'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('input', () => { el.dataset.userEdited = 'true'; });
  });

  onTargetChange(); // populate defaults for initial selection
  // Hook up all sub-mode radio buttons generically.
  document.querySelectorAll('input[type="radio"][name$="-mode"]').forEach(radio => {
    radio.addEventListener('change', () => {
      const target = radio.name.replace(/-mode$/, '');
      applyModePanels(target);
      updatePortGroup(target, getModeFor(target));
      // Auto-fill ports when switching to a port-needing web mode
      // (mirrors the auto-fill that onTargetChange() does on target switch).
      if (target === 'web') {
        const portEl = document.getElementById('ports');
        if (portEl && portEl.dataset.userEdited !== 'true') {
          const mode = getModeFor(target);
          if (WEB_MODES_WITH_PORTS.includes(mode)) {
            portEl.value = (TARGET_PORTS[target] || []).join(', ');
          }
        }
      }
    });
  });
  // Initialise all custom-select widgets on the page.
  document.querySelectorAll('.cs-wrap').forEach(wrap => initCustomSelect(wrap));
  initAdvancedOpts();   // animated expand/collapse for the advanced options panel
  fetchVersion();   // async version badge
  loadHistory();    // populate history panel
  initTheme();      // apply saved theme (before locale so tokens are ready)
  initLocale();     // apply saved locale (must run after DOM is ready)
});

// ── Advanced-options animated expand/collapse ────────────────────────────────
/**
 * Wire up animated open/close for the Advanced Options <details> element.
 * Intercepts summary clicks and drives a height transition on .adv-body
 * (mirroring the panel-stage mechanism) together with a fade+slide animation
 * on .adv-inner, reusing the panel-appear / panel-leave keyframes and the
 * --panel-anim-dur token so vivid / off modes apply automatically.
 */
function initAdvancedOpts() {
  const details = document.getElementById('advanced-opts');
  if (!details) return;
  const summary = details.querySelector(':scope > summary');
  const body    = details.querySelector('.adv-body');
  if (!summary || !body) return;

  summary.addEventListener('click', e => {
    e.preventDefault();

    if (details.open) {
      // ── Collapse: animate from current height to 0, then toggle off ────
      details.classList.remove('adv-is-open'); // start chevron rotation immediately
      const currentH = body.scrollHeight;
      body.style.height = currentH + 'px';
      void body.offsetHeight;                    // flush reflow before transition
      body.classList.remove('adv-entering');
      body.classList.add('adv-leaving');
      body.style.height = '0px';

      body.addEventListener('transitionend', () => {
        details.open = false;
        body.classList.remove('adv-leaving');
        body.style.height = '';
      }, { once: true });
    } else {
      // ── Expand: open, measure, animate from 0 to full height ───────────
      details.open = true;
      details.classList.add('adv-is-open');    // start chevron rotation immediately
      const targetH = body.scrollHeight;
      body.style.height = '0px';
      void body.offsetHeight;                    // flush reflow before transition
      body.classList.remove('adv-leaving');
      body.classList.add('adv-entering');
      body.style.height = targetH + 'px';

      body.addEventListener('transitionend', () => {
        body.classList.remove('adv-entering');
        body.style.height = '';
      }, { once: true });
    }
  });
}

// ── Custom select component (cs-*) ──────────────────────────────────────────
/**
 * Initialise one .cs-wrap widget.
 * Syncs the hidden native <select> so val() continues to work without
 * modifications elsewhere.  Full keyboard support (Enter/Space to open,
 * ↑↓ to navigate, Escape/Tab to close).
 */
function initCustomSelect(wrap) {
  const trigger = wrap.querySelector('.cs-trigger');
  const label   = wrap.querySelector('.cs-label');
  const list    = wrap.querySelector('.cs-list');
  const select  = wrap.querySelector('select');
  const items   = Array.from(wrap.querySelectorAll('.cs-item'));
  if (!trigger || !list || !items.length) return;

  // Mark the widget as having a selection as soon as it is initialised.
  // This applies the persistent primary-border indicator (like radio :checked)
  // regardless of keyboard focus state.
  wrap.classList.add('has-selection');

  /**
   * Close the popup.
   * @param {boolean} [restoreFocus=true] When true, return keyboard focus to
   *   the trigger (normal close via key or item select).  Pass false when
   *   closing because the user clicked OUTSIDE the widget so we do not steal
   *   focus away from whichever element they just clicked.
   */
  function close(restoreFocus = true) {
    wrap.classList.remove('open');
    trigger.setAttribute('aria-expanded', 'false');
    if (restoreFocus) trigger.focus();
  }

  function open() {
    wrap.classList.add('open');
    trigger.setAttribute('aria-expanded', 'true');
    const sel = list.querySelector('[aria-selected="true"]') || items[0];
    if (sel) sel.focus();
  }

  function selectItem(item) {
    items.forEach(it => it.removeAttribute('aria-selected'));
    item.setAttribute('aria-selected', 'true');
    // Ensure persistent selection indicator remains active after choice.
    wrap.classList.add('has-selection');
    // Sync the visible label (keep data-i18n so applyLocale() can re-translate).
    if (label) {
      label.textContent  = item.textContent;
      label.dataset.i18n = item.dataset.i18n || '';
    }
    // Sync hidden native select so val('target') always reads the correct value.
    if (select) select.value = item.dataset.value || '';
    // Trigger dependent logic directly (no change event on a hidden element).
    if (wrap.id === 'target-wrap') onTargetChange();
    close();
  }

  trigger.addEventListener('click', () => {
    wrap.classList.contains('open') ? close() : open();
  });

  trigger.addEventListener('keydown', e => {
    if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(); }
    if (e.key === 'Escape') close();
    if (e.key === 'ArrowDown') { e.preventDefault(); open(); }
  });

  items.forEach((item, idx) => {
    item.setAttribute('tabindex', '-1');
    item.addEventListener('click', () => selectItem(item));
    item.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ' ')  { e.preventDefault(); selectItem(item); }
      if (e.key === 'ArrowDown') { e.preventDefault(); (items[idx + 1] || items[idx]).focus(); }
      if (e.key === 'ArrowUp')   { e.preventDefault(); (items[idx - 1] || items[idx]).focus(); }
      if (e.key === 'Escape' || e.key === 'Tab') { e.preventDefault(); close(); }
    });
  });

  // Close when focus moves outside the widget via an outside click.
  // restoreFocus=false: do NOT steal focus from the element the user just clicked.
  document.addEventListener('click', e => {
    if (!wrap.contains(e.target)) close(false);
  }, true);
}

// ── Form dynamics ────────────────────────────────────────────────────────────
function getModeFor(target) {
  const el = document.querySelector(`input[name="${target}-mode"]:checked`);
  return el ? el.value : '';
}

function applyModePanels(target) {
  const mode = getModeFor(target);
  const panels = (TARGET_MODE_PANELS[target] || {});
  Object.entries(panels).forEach(([id, visibleModes]) => {
    const panel = document.getElementById(id);
    if (panel) panel.hidden = !visibleModes.includes(mode);
  });
}

/**
 * Show or hide the #port-group column and its text-input variant based on the
 * current target and mode.  Driven by WEB_MODES_WITH_PORTS so adding a new
 * web mode that needs ports only requires updating that constant.
 *
 * Rules:
 *   - web + mode in WEB_MODES_WITH_PORTS → show port-group + text input
 *   - web + other modes                  → hide port-group entirely
 *   - non-web targets                    → show port-group + text input
 */
function updatePortGroup(target, mode) {
  const group   = document.getElementById('port-group');
  const textGrp = document.getElementById('ports-text-group');

  const needsPorts = ((target === 'web') && WEB_MODES_WITH_PORTS.includes(mode)) || (target !== 'web');

  if (group)   group.hidden   = !needsPorts;
  if (textGrp) textGrp.hidden = !needsPorts;
}

// Track the first onTargetChange() call so no enter-animation plays on cold
// page load (the form's initial state is already fully visible in the HTML).
let _initTargetDone = false;

/**
 * Cleanup function for any in-flight sequential panel transition.
 * Calling it cancels the pending animationend listeners and immediately
 * hides all departing panels so a rapid target switch always wins.
 */
let _pendingReveal = null;

/**
 * Measure the layout height a panel element would occupy inside the stage
 * when visible.  The value matches what panel-stage.scrollHeight returns with
 * that panel present: offsetHeight (content + padding + border) plus any CSS
 * margins.  Using scrollHeight instead would be off by the border widths,
 * causing a visible snap when height:auto is restored after the transition.
 * Uses a detached clone so the live DOM is never modified.
 * @param  {HTMLElement} el         The panel to measure.
 * @param  {number}      stageWidth The layout width to simulate (matches .panel-stage).
 * @returns {number} Height in CSS pixels.
 */
function measurePanelHeight(el, stageWidth) {
  const clone = el.cloneNode(true);
  clone.hidden = false;
  clone.style.cssText = [
    'position: absolute',
    'top: -9999px',
    'left: 0',
    'width: ' + (stageWidth || 300) + 'px',
    'visibility: hidden',
    'pointer-events: none',
  ].join('; ');
  document.body.appendChild(clone);
  // offsetHeight includes content + padding + border (unlike scrollHeight which
  // excludes border), so it exactly matches the child's contribution to the
  // parent container's scrollHeight.  Add CSS margins on top to get the total
  // space the element occupies inside an overflow:hidden stage.
  const cs           = getComputedStyle(clone);
  const marginTop    = parseFloat(cs.marginTop)    || 0;
  const marginBottom = parseFloat(cs.marginBottom) || 0;
  const h            = clone.offsetHeight + marginTop + marginBottom;
  document.body.removeChild(clone);
  return h;
}

function onTargetChange() {
  const target  = val('target');
  const animate = _initTargetDone;
  _initTargetDone = true;

  // Cancel any previous in-flight transition before starting a new one.
  if (_pendingReveal) {
    _pendingReveal();
    _pendingReveal = null;
  }

  const incoming = document.getElementById('fields-' + target);
  if (!incoming) return;

  // Panels marked data-panel-empty carry no form content (e.g. imap, pop).
  // All departing panels are still hidden, but the incoming panel is never
  // revealed — so the user never sees an empty bordered box.
  const isEmptyPanel = incoming.dataset.panelEmpty === 'true';

  const stage = document.getElementById('panel-stage');

  // Collect all currently-visible panels that need to depart.
  const departing = Array.from(document.querySelectorAll('.target-fields'))
    .filter(fs => fs !== incoming && !fs.hidden);

  /**
   * Reveal the incoming panel with an optional enter animation.
   * Called only after all departing panels have finished their exit, so the
   * two animations are strictly sequential — no layout overlap.
   * The stage height is already at the incoming panel\'s measured height
   * (set during the departure phase), so no visual jump occurs on reveal.
   */
  function revealIncoming() {
    _pendingReveal = null;
    incoming.classList.remove('panel-leaving');
    if (isEmptyPanel) {
      // Empty panel: keep the fieldset hidden; collapse the stage back to
      // auto height (naturally 0 since no visible children remain).
      if (stage) stage.style.height = '';
      return;
    }
    incoming.hidden = false;
    if (animate) {
      incoming.classList.remove('panel-entering');
      void incoming.offsetWidth; // force reflow so animation restarts cleanly
      incoming.classList.add('panel-entering');
      // Restore auto height once the entrance animation is done so the stage
      // can grow/shrink naturally afterwards (e.g. sub-mode panel toggles).
      incoming.addEventListener('animationend', () => {
        if (stage) stage.style.height = '';
      }, { once: true });
    } else if (stage) {
      stage.style.height = '';
    }
  }

  if (animate && departing.length > 0) {
    // ── Sequential + height-animated transition ──────────────────────────
    // 1. Lock the stage at its current pixel height (enables CSS transition).
    // 2. Measure the incoming panel height via a detached clone.
    // 3. Set stage to the incoming height — both the height transition and the
    //    departure animation run in parallel over the same --panel-anim-dur.
    // 4. After all departing panels finish, reveal the incoming panel.
    if (stage) {
      const currentH  = stage.scrollHeight;
      // Empty panels target height 0 so the stage collapses smoothly.
      const incomingH = isEmptyPanel ? 0 : measurePanelHeight(incoming, stage.offsetWidth);
      stage.style.height = currentH + 'px';   // lock to pixels so transition works
      void stage.offsetHeight;                 // force reflow
      stage.style.height = incomingH + 'px';  // trigger height CSS transition
    }

    // Keep the incoming panel hidden while the outgoing content departs.
    incoming.hidden = true;

    let pending = departing.length;
    const listeners = [];

    departing.forEach(fs => {
      fs.classList.remove('panel-entering');
      fs.classList.add('panel-leaving');

      const handler = () => {
        fs.hidden = true;
        fs.classList.remove('panel-leaving');
        pending -= 1;
        if (pending === 0) revealIncoming();
      };

      fs.addEventListener('animationend', handler, { once: true });
      listeners.push({ fs, handler });
    });

    // Store cleanup so a rapid switch can cancel this flight.
    _pendingReveal = () => {
      if (stage) stage.style.height = '';
      listeners.forEach(({ fs, handler }) => {
        fs.removeEventListener('animationend', handler);
        fs.hidden = true;
        fs.classList.remove('panel-leaving', 'panel-entering');
      });
    };
  } else if (animate && !isEmptyPanel && stage) {
    // ── Grow from empty stage ─────────────────────────────────────────────
    // The previous target was an empty panel (never revealed, so departing=[]).
    // The stage is at height 0; animate it up to the incoming panel height
    // while the incoming panel fades in — producing the same smooth effect as
    // the symmetric "collapse to empty" transition in the opposite direction.
    document.querySelectorAll('.target-fields').forEach(fs => {
      if (fs !== incoming) {
        fs.hidden = true;
        fs.classList.remove('panel-entering', 'panel-leaving');
      }
    });
    const incomingH = measurePanelHeight(incoming, stage.offsetWidth);
    stage.style.height = '0px';       // lock at 0 so CSS transition has a start
    void stage.offsetHeight;           // force reflow
    stage.style.height = incomingH + 'px'; // trigger height CSS transition
    revealIncoming();
  } else {
    // Cold load or no visible departing panel and no animation: instant switch.
    document.querySelectorAll('.target-fields').forEach(fs => {
      if (fs !== incoming) {
        fs.hidden = true;
        fs.classList.remove('panel-entering', 'panel-leaving');
      }
    });
    if (stage) stage.style.height = '';
    revealIncoming();
  }

  // Non-animation updates — apply immediately, do not wait for transition.
  const portEl = document.getElementById('ports');
  if (portEl && portEl.dataset.userEdited !== 'true') {
    portEl.value = (TARGET_PORTS[target] || []).join(', ');
  }
  const hostEl = document.getElementById('host');
  if (hostEl) {
    hostEl.placeholder = t(TARGET_PLACEHOLDER_KEYS[target] || 'ph-host-default');
  }
  applyModePanels(target);
  updatePortGroup(target, getModeFor(target));
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
  const cfg = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
  return L.divIcon({
    className:   'geo-marker ' + cfg.cssClass,
    html:        '<span class="geo-marker__dot"></span>',
    iconSize:    [20, 20],
    iconAnchor:  [10, 10],
    popupAnchor: [0, -14],
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
 */
function buildMapLegend(pointTypes) {
  const legend = L.control({ position: 'bottomright' });
  legend.onAdd = function () {
    const div = L.DomUtil.create('div', 'geo-legend');
    div.innerHTML = pointTypes.map(type => {
      const cfg = MAP_POINT_CONFIGS[type] || MAP_POINT_CONFIGS.target;
      return '<div class="geo-legend__item">' +
        '<span class="' + cfg.cssClass + ' geo-legend__dot"></span>' +
        '<span>' + esc(t(cfg.i18nKey)) + '</span>' +
        '</div>';
    }).join('');
    return div;
  };
  return legend;
}

/** Render (or remove) the Leaflet map based on geo results.
 *  pub / tgt are GeoAnnotation objects that may be null/undefined.
 */
function renderMap(pub, tgt) {
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
    if (_map) { _map.remove(); _map = null; _tileLayer = null; }
    return;
  }

  // Reveal the outer wrapper (bar + map) before showing the map itself.
  const outerEl = document.getElementById('geo-map-outer');
  if (outerEl) outerEl.hidden = false;
  container.classList.add('visible');

  // Render the map tile bar (light/osm/dark) above the map on first creation.
  renderMapBar();

  // Destroy any existing map instance before creating a new one.
  if (_map) { _map.remove(); _map = null; _tileLayer = null; }

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

  // Draw a dashed polyline and show the distance badge when both endpoints exist.
  if (latLngs.length === 2) {
    L.polyline(latLngs, {
      color: '#5b8dee', weight: 2, dashArray: '6 5', opacity: 0.75,
    }).addTo(_map);
    if (distEl) {
      const km = haversineKm(latLngs[0][0], latLngs[0][1], latLngs[1][0], latLngs[1][1]);
      distEl.textContent = t('map-distance') + ': ' + Math.round(km).toLocaleString() + ' km';
      distEl.hidden = false;
    }
  }

  // Add a legend when multiple marker types are present so users can
  // distinguish origin (偵測起點) from target (偵測目標).
  if (points.length > 1) {
    buildMapLegend(points.map(p => p.type)).addTo(_map);
  }

  if (latLngs.length === 1) {
    _map.setView(latLngs[0], 8);
  } else {
    _map.fitBounds(latLngs, { padding: [40, 40] });
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
    const ts = item.created_at
      ? new Date(item.created_at).toLocaleString()
      : '';
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
