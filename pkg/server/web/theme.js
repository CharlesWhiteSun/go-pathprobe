'use strict';
// ── theme.js — theme management (PathProbe.Theme) ─────────────────────────
// Depends on: config.js (PathProbe.Config)
// Cross-module calls (resolved at runtime):
//   PathProbe.Map.syncMapTileVariantToTheme(id) — swap map tiles on theme change
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

/** Apply themeId to <html data-theme> with a targeted fade transition.
 *  Only the .main content area fades out while the theme variables switch.
 *  Header and footer remain fully visible.
 */
function applyTheme(themeId) {
  const { THEMES, DEFAULT_THEME } = PathProbe.Config;
  const id      = THEMES.includes(themeId) ? themeId : DEFAULT_THEME;
  const body    = document.body;
  const mainEl  = document.querySelector('.main');
  body.classList.add('theme-transitioning');
  const listenTarget = mainEl || body;
  const onFaded = (e) => {
    if (e.target !== listenTarget || e.propertyName !== 'opacity') return;
    listenTarget.removeEventListener('transitionend', onFaded);
    document.documentElement.dataset.theme = id;
    try { localStorage.setItem('pp-theme', id); } catch (_) {}
    document.querySelectorAll('.theme-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.theme === id);
    });
    // Silently swap map tiles while main content is invisible.
    if (PathProbe.Map && PathProbe.Map.syncMapTileVariantToTheme) {
      PathProbe.Map.syncMapTileVariantToTheme(id);
    }
    requestAnimationFrame(() => body.classList.remove('theme-transitioning'));
  };
  listenTarget.addEventListener('transitionend', onFaded);
}

/** Public entry point called by the <button onclick> handler. */
function setTheme(themeId) { applyTheme(themeId); }

/** Restore saved theme from localStorage; fall back to the server-declared
 *  default (data-default-theme on <html>).  Applied without fade animation.
 */
function initTheme() {
  const { THEMES, DEFAULT_THEME } = PathProbe.Config;
  const htmlDefault  = (document.documentElement.dataset.defaultTheme || '').trim();
  const serverDefault = THEMES.includes(htmlDefault) ? htmlDefault : DEFAULT_THEME;
  let saved = serverDefault;
  try { saved = localStorage.getItem('pp-theme') || serverDefault; } catch (_) {}
  const id = THEMES.includes(saved) ? saved : DEFAULT_THEME;
  document.documentElement.dataset.theme = id;
  try { localStorage.setItem('pp-theme', id); } catch (_) {}
  document.querySelectorAll('.theme-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.theme === id);
  });
  if (PathProbe.Map && PathProbe.Map.syncMapTileVariantToTheme) {
    PathProbe.Map.syncMapTileVariantToTheme(id);
  }
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.Theme = { applyTheme, setTheme, initTheme };

// Expose globally for HTML event handlers.
window.setTheme = setTheme;
