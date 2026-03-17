'use strict';
// ── locale.js — i18n / locale management (PathProbe.Locale) ──────────────
// Depends on: config.js (PathProbe.Config)
// Cross-module calls (resolved at runtime, no load-time circular deps):
//   PathProbe.Renderer.rerenderLast()  — re-render results in new locale
//   PathProbe.History.rerenderLast()   — re-render history in new locale
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

let _locale = 'en';

/** Return the translation for key in the current locale, falling back to en. */
function t(key) {
  const locs = window.LOCALES || {};
  return (locs[_locale] || {})[key] || (locs.en || {})[key] || key;
}

/**
 * Re-write the copyright year after applyLocale() has set the raw i18n text.
 * Builds a range string: just the start year when equal to the current year,
 * otherwise "startYear–currentYear" (en-dash U+2013).
 */
function updateCopyrightYear() {
  const start = PathProbe.Config.COPYRIGHT_START_YEAR;
  const now   = new Date().getFullYear();
  const yearStr = now > start
    ? start + '\u2013' + now
    : String(start);
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
  // Refresh host placeholder (depends on current target selection).
  // Read DOM directly to avoid a circular dependency with form.js.
  const targetEl = document.getElementById('target');
  const target   = targetEl ? targetEl.value.trim() : '';
  const hostEl   = document.getElementById('host');
  if (hostEl) {
    const key = PathProbe.Config.TARGET_PLACEHOLDER_KEYS[target] || 'ph-host-default';
    hostEl.placeholder = t(key);
  }
  // Update run button (unless currently running).
  const runBtn = document.getElementById('run-btn');
  if (runBtn && !runBtn.disabled) runBtn.textContent = t('btn-run');
  // Highlight the active language button.
  document.querySelectorAll('.lang-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.lang === _locale);
  });
  document.documentElement.lang = _locale;
  updateCopyrightYear();
  // Re-render results and history in the new locale (resolved at runtime).
  if (PathProbe.Renderer && PathProbe.Renderer.rerenderLast) {
    PathProbe.Renderer.rerenderLast();
  }
  if (PathProbe.History && PathProbe.History.rerenderLast) {
    PathProbe.History.rerenderLast();
  }
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

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.Locale = { t, applyLocale, setLocale, initLocale };

// Expose globally for HTML event handlers.
window.setLocale = setLocale;
