'use strict';

// ── theme.js — visual theme management (PathProbe.Theme) ─────────────────
// Self-contained module responsible for:
//   • applying a theme to <html data-theme> with a targeted fade transition
//   • persisting the user's theme choice in localStorage
//   • initialising the theme from localStorage on page load
//
// Dependencies (runtime-resolved, no hard import):
//   • window.PathProbe.Config — THEMES, DEFAULT_THEME (declared in config.js)
//   • window.PathProbe.Map.syncMapTileVariantToTheme — optional; called after
//     the theme is applied so map tiles switch to the matching variant.  The
//     call is guard-checked at call time so theme.js has no hard dependency on
//     the map module's load order.
//
// This module must be loaded AFTER config.js but BEFORE app.js.
(() => {
  // ── Config aliases (runtime-resolved) ─────────────────────────────────
  function _cfg() {
    return (window.PathProbe && window.PathProbe.Config) || {};
  }

  function _themes()      { return _cfg().THEMES       || ['default', 'deep-blue', 'light-green', 'forest-green', 'dark']; }
  function _defaultTheme(){ return _cfg().DEFAULT_THEME || 'default'; }

  // ── Core theme application ─────────────────────────────────────────────

  /**
   * Apply themeId to <html data-theme> with a targeted fade transition.
   * Only the .main content area fades out while the theme variables switch.
   * Header and footer remain fully visible and cross-fade their own
   * background / text colours via dedicated CSS transitions, so the chrome
   * always stays on screen during a theme change.
   */
  function applyTheme(themeId) {
    const themes = _themes();
    const defaultTheme = _defaultTheme();
    const id = themes.includes(themeId) ? themeId : defaultTheme;
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
      // Runtime-resolved: avoids a hard load-order dependency on the map module.
      if (window.PathProbe && window.PathProbe.Map && window.PathProbe.Map.syncMapTileVariantToTheme) {
        window.PathProbe.Map.syncMapTileVariantToTheme(id);
      }
      // Remove the class on the next frame so the fade-in transition fires.
      requestAnimationFrame(() => body.classList.remove('theme-transitioning'));
    };
    listenTarget.addEventListener('transitionend', onFaded);
  }

  // ── Public API ────────────────────────────────────────────────────────

  /** Public entry point called by theme-button onclick handlers. */
  function setTheme(themeId) { applyTheme(themeId); }

  /**
   * Restore saved theme from localStorage; fall back to the server-declared
   * default (data-default-theme on <html>) so a service restart always starts
   * on the intended theme when no user preference exists.
   * Applies the theme without the fade animation (page is not yet visible).
   */
  function initTheme() {
    const themes = _themes();
    const defaultTheme = _defaultTheme();
    // Server-declared default: read from HTML attribute set by the server.
    const htmlDefault = (document.documentElement.dataset.defaultTheme || '').trim();
    const serverDefault = themes.includes(htmlDefault) ? htmlDefault : defaultTheme;
    let saved = serverDefault;
    try { saved = localStorage.getItem('pp-theme') || serverDefault; } catch (_) {}
    // Apply without animation: set theme vars immediately so there is no flash.
    const id = themes.includes(saved) ? saved : defaultTheme;
    document.documentElement.dataset.theme = id;
    try { localStorage.setItem('pp-theme', id); } catch (_) {}
    document.querySelectorAll('.theme-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.theme === id);
    });
    // Runtime-resolved sync for map tiles.
    if (window.PathProbe && window.PathProbe.Map && window.PathProbe.Map.syncMapTileVariantToTheme) {
      window.PathProbe.Map.syncMapTileVariantToTheme(id);
    }
  }

  // ── Namespace registration ────────────────────────────────────────────
  const PathProbe = window.PathProbe || {};
  PathProbe.Theme = { applyTheme, setTheme, initTheme };
  window.PathProbe = PathProbe;

  // Expose setTheme as a global so inline onclick="setTheme('dark')" attributes
  // in index.html continue to work without modification.
  window.setTheme = setTheme;
})(); // end theme IIFE
