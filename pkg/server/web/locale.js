'use strict';

// ── locale.js — i18n / locale management (PathProbe.Locale) ─────────────
// Self-contained module responsible for:
//   • maintaining the active locale identifier (_locale)
//   • translating keys via t()
//   • applying translations to the DOM (applyLocale())
//   • persisting the user's choice (setLocale())
//   • initialising locale on page load (initLocale())
//
// Dependencies (runtime-resolved, no hard import):
//   • window.LOCALES          — i18n dictionary loaded by i18n.js
//   • window.PathProbe.Config — COPYRIGHT_START_YEAR, TARGET_PLACEHOLDER_KEYS
//   • window.PathProbe.Renderer.rerenderLast — optional, called after locale
//     change so the results section is re-rendered in the new language
//   • window.PathProbe.History.rerenderLast  — optional, called after locale
//     change so the history list is re-rendered with locale-aware timestamps
//
// Cross-module calls are intentionally resolved at call time (not at module
// load time) to avoid load-order coupling.  If a dependent module has not yet
// registered itself, the guard expression evaluates to false and the call is
// silently skipped (graceful degradation).
//
// This module must be loaded AFTER i18n.js and config.js but BEFORE app.js.
(() => {
  // ── Private state ────────────────────────────────────────────────────────
  let _locale = 'en';

  // ── Translation helper ───────────────────────────────────────────────────

  /** Return the translation for key in the current locale, falling back to en. */
  function t(key) {
    const locs = window.LOCALES || {};
    return (locs[_locale] || {})[key] || (locs.en || {})[key] || key;
  }

  // ── Copyright year helper ─────────────────────────────────────────────────

  /**
   * Re-write the copyright year in all [data-i18n="footer-copyright"] elements.
   * Builds a range string: the start year alone when it equals the current
   * year, otherwise a range separated by an en-dash.  The regex targets the
   * first four-digit sequence so the logic is locale-independent.
   *
   * The start year is read from PathProbe.Config at call time so this module
   * never hard-codes the year value.
   */
  function updateCopyrightYear() {
    const cfg = (window.PathProbe && window.PathProbe.Config) || {};
    const startYear = cfg.COPYRIGHT_START_YEAR || new Date().getFullYear();
    const now = new Date().getFullYear();
    const yearStr = now > startYear
      ? startYear + '\u2013' + now
      : String(startYear);
    document.querySelectorAll('[data-i18n="footer-copyright"]').forEach(el => {
      el.textContent = el.textContent.replace(/\d{4}/, yearStr);
    });
  }

  // ── DOM application ──────────────────────────────────────────────────────

  /** Apply the current locale to all [data-i18n] elements and refresh dynamic UI. */
  function applyLocale() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
      el.textContent = t(el.dataset.i18n);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
      el.placeholder = t(el.dataset.i18nPlaceholder);
    });

    // Refresh host placeholder (depends on current target selection).
    // val() lives in app.js; access defensively so locale.js has no hard
    // dependency on app.js load order.
    const cfg = (window.PathProbe && window.PathProbe.Config) || {};
    const placeholderKeys = cfg.TARGET_PLACEHOLDER_KEYS || {};
    const targetEl = document.getElementById('target');
    const target = targetEl ? targetEl.value : '';
    const hostEl = document.getElementById('host');
    if (hostEl) hostEl.placeholder = t(placeholderKeys[target] || 'ph-host-default');

    // Update run button label (unless currently running — disabled state
    // means a spinner is shown and must not be overwritten).
    const runBtn = document.getElementById('run-btn');
    if (runBtn && !runBtn.disabled) runBtn.textContent = t('btn-run');

    // Highlight the active language button.
    document.querySelectorAll('.lang-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.lang === _locale);
    });
    document.documentElement.lang = _locale;

    // Update footer copyright year range after i18n strings have been applied.
    updateCopyrightYear();

    // ── Runtime-resolved cross-module calls ─────────────────────────────
    // These are intentionally resolved at call time rather than import time so
    // locale.js does not create a hard load-order dependency on renderer.js or
    // history.js.  The guard expressions safely evaluate to undefined/false
    // when the target module has not yet registered itself.
    if (window.PathProbe && window.PathProbe.Renderer && window.PathProbe.Renderer.rerenderLast) {
      window.PathProbe.Renderer.rerenderLast();
    }
    if (window.PathProbe && window.PathProbe.History && window.PathProbe.History.rerenderLast) {
      window.PathProbe.History.rerenderLast();
    }
    // Re-apply i18n-dependent map labels (geo-precision-notice, tile-bar
    // aria-labels, Leaflet popup HTML, distance badge) without a full map rebuild.
    // map.js registers PathProbe.Map.rerenderLabels — guard so locale.js remains
    // independent of map.js load order.
    if (window.PathProbe && window.PathProbe.Map && window.PathProbe.Map.rerenderLabels) {
      window.PathProbe.Map.rerenderLabels();
    }
  }

  // ── Public API ───────────────────────────────────────────────────────────

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

  /** Return the current active locale identifier. */
  function getLocale() { return _locale; }

  // ── Namespace registration ───────────────────────────────────────────────
  const PathProbe = window.PathProbe || {};
  PathProbe.Locale = { t, getLocale, setLocale, initLocale, applyLocale };
  window.PathProbe = PathProbe;

  // Expose setLocale as a global so inline onclick="setLocale('en')" attributes
  // in index.html continue to work without modification.
  window.setLocale = setLocale;
})(); // end locale IIFE
