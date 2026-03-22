'use strict';

// ── config.js — all application-wide constants (PathProbe.Config) ─────────
// All declarations are wrapped in an arrow IIFE so that the top-level const
// identifiers remain local to this closure and do NOT pollute the shared
// script scope that all classic <script> elements on the page share.  Without
// this wrapper, a browser loading both config.js and app.js in the same
// realm would throw "redeclaration of const" because app.js destructures the
// same identifier names from PathProbe.Config using identical local names.
//
// This module has NO dependencies on other PathProbe modules.
// It must be loaded before app.js so that PathProbe.Config is available when
// the main module runs.
// eslint-disable-next-line no-extra-parens
(() => {
const PathProbe = window.PathProbe || {};

// ── Map point role configurations ─────────────────────────────────────────
// Each entry defines the CSS class and i18n label key used for a geocoded
// point on the Leaflet map.  Adding a new role requires only a new key here —
// renderMap(), buildMarkerIcon(), buildPopupHtml(), and buildMapLegend() all
// read this object so no other code needs to change.
const MAP_POINT_CONFIGS = {
  'origin': { cssClass: 'geo-marker--origin', i18nKey: 'map-origin', shortLabel: 'A' },
  'target': { cssClass: 'geo-marker--target', i18nKey: 'map-target', shortLabel: 'B' },
};

// ── Marker colour scheme configurations ───────────────────────────────────────
// Each entry provides originColor and targetColor for the two map roles.
// The active scheme is applied by applyMarkerColorScheme(), which writes
// --mc-origin / --mc-target CSS custom properties onto the <html> element.
// All marker CSS rules use var(--mc-origin) / var(--mc-target) so switching
// schemes requires no DOM rebuild—only a single property update.
//
// To add a new scheme: add a key here + a matching i18n key (en + zh-TW).
// No other code changes are required.
const MARKER_COLOR_SCHEME_CONFIGS = {
  // ocean — teal-blue origin / warm amber target
  'ocean': { originColor: '#0891b2', targetColor: '#f59e0b', i18nKey: 'marker-color-ocean' },
};

// ── Marker style configuration ────────────────────────────────────────────
// buildHtml(roleCfg) receives the MAP_POINT_CONFIGS entry for the current role
// and returns an HTML string used as the Leaflet divIcon inner HTML.
// The pulse variant uses CSS tokens (--marker-border / --marker-inner /
// --marker-shadow) and role colour tokens (--mc-origin / --mc-target) so it
// adapts automatically when the active [data-theme] changes.
const MARKER_STYLE_CONFIGS = {
  'diamond-pulse': {
    i18nKey:     'marker-style-diamond-pulse',
    iconSize:    [36, 36],
    iconAnchor:  [18, 18],
    popupAnchor: [0, -20],
    buildHtml:   (_rc) =>
      '<span class="geo-marker__dia-pulse-ring"></span>' +
      '<span class="geo-marker__dia-pulse-core"></span>',
  },
};

// ── Connector line style configurations ──────────────────────────────────────────────
// Each entry defines how the gradient arc connector between origin and target
// is rendered.  All styles use a northward quadratic-bezier arch.
//
// arcFactor:    height of the arch (0 ≈ flat, 0.65 = very high arch).
// weight:       stroke width in pixels (polyline type).
// opacity:      stroke opacity (0–1).
// dashArray:    null for solid; an SVG stroke-dasharray string for patterned lines.
//               For non-sticky dots use '0.1 <gap>' where gap > weight so
//               rounded caps never overlap each other.
// segments:     number of arc waypoints (higher → smoother Bézier and finer
//               gradient; 120 is a good balance for smooth rendering).
// type:         'polyline' (default) — gradient SVG sub-polylines.
//               'arrows'            — divIcon arrow symbols spaced along the arc.
// arrowSymbol:  (arrows) Unicode glyph placed at each position.
// arrowSize:    (arrows) font-size + icon bounding box in px (default 14).
// arrowSpacing: (arrows) screen-pixel gap between successive arrow symbols.
//               Density stays visually consistent at every zoom level.
// groupStart:   true on the first entry of each style family (dash, arrow);
//               triggers a flex line-break in the picker bar.
//
// To add a new style: add a key here + matching translations in i18n.js.
// No other code changes are required.
const CONNECTOR_LINE_CONFIGS = {
  // ── Tick family (›) ── compact open-chevron indicator aligned to arc tangent ───
  // arrowSize 4 px + arrowSpacing 6 px gives a subtle directional texture that
  // reads as flow without overpowering the map.  spineWeight 0 = no spine arc.
  // glowEnabled / glowConfig enable the ConnectorGlowLayer meteor animation.
  'tick-xs': { i18nKey: 'connector-tick-xs', arcFactor: 0.25, weight: 1, opacity: 0.85, dashArray: null, segments: 120, type: 'arrows', arrowShape: 'open', arrowSize: 4, arrowSpacing: 6, spineWeight: 0, glowEnabled: true, glowConfig: 'default' },
};

// ── Connector glow animation configuration ────────────────────────────────────
// Controls the "meteor" light-pulse animation that travels along the connector
// arc from origin to target, then pauses before looping smoothly.
//
// travelMs:    milliseconds for the glowing orb to travel from origin to target.
// pauseMs:     milliseconds the orb lingers at the target before restarting.
// glowRadius:  outer radial-gradient radius of the orb in screen pixels.
// glowOpacity: peak opacity of the outer glow (0–1).
// tailLength:  fraction of the total arc pixel-length used for the comet tail.
// fadeMs:      milliseconds to fade the glow from full brightness to zero after
//              the head arrives at the destination.  During this phase the tail
//              converges back into the head (tailPx shrinks proportionally) and
//              all opacity values are multiplied by a linear ramp 1 → 0.  This
//              creates the "extinguish / breathing-lamp" effect.
// pauseMs:     milliseconds of complete darkness after the fade-out, before
//              the next travel cycle begins.  Canvas is blank during this phase.
//
// Animation cycle: travel → fade-out → dark → travel → …
//   cycleDur = travelMs + fadeMs + pauseMs
//
// To add a new preset: add a key here and reference it via
// CONNECTOR_LINE_CONFIGS[id].glowConfig.  No other code changes required.
const CONNECTOR_GLOW_CONFIGS = {
  'default': {
    travelMs:    2500,
    fadeMs:      700,
    pauseMs:     1000,
    glowRadius:  14,
    glowOpacity: 0.85,
    tailLength:  0.18,
  },
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
// to the light/neutral style.  Must stay in sync with THEMES (below).
const MAP_DARK_THEMES = new Set(['dark', 'deep-blue', 'forest-green']);

// ── Geo proximity threshold ────────────────────────────────────────────────
// When two geocoded points are closer than this distance (km) AND at least one
// has country-level location precision, the map switches from fitBounds() to a
// fixed-zoom setView() centred on the midpoint.  This prevents Leaflet from
// over-zooming on identical or near-identical country centroids and producing
// an uninformative blank-ocean view.
const GEO_SAME_REGION_THRESHOLD_KM = 200;

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

// Web modes that hide the standard Target Host input entirely.
// dns operates on domain names entered separately; http uses its own URL input.
const WEB_MODES_HIDE_HOST = ['dns', 'http'];

// Web modes that show the HTTP URL input (#http-url-group) in place of the
// Target Host input.  This is a strict subset of WEB_MODES_HIDE_HOST.
const WEB_MODES_SHOW_HTTP_URL = ['http'];

// Web modes that show the DNS Domains input (#dns-domains-group) in place of
// the Target Host input.  This is a strict subset of WEB_MODES_HIDE_HOST.
const WEB_MODES_SHOW_DNS_DOMAINS = ['dns'];

// Advanced options applicability matrix.
// Maps target → { mode: string[] } where the array lists supported option keys.
// 'timeout' is universally applicable and is intentionally omitted.
// An unrecognised target or mode falls back silently to showing ALL options
// (fail-open: new modes never accidentally hide useful controls).
//
// Valid option keys (match 'adv-opt-{key}' wrapper IDs in index.html):
//   'mtr-count' — probe count per hop / per port batch (ConnectivityRunner + TracerouteRunner)
//   'insecure'  — skip TLS certificate verification (HTTPRunner, FTPRunner, SMTPRunner)
//   'geo'       — enable geo IP annotation and map display
//
// Extending:
//   • New web mode  → add an entry under web.{mode}.
//   • New target    → add a top-level entry: target: { '': [...] }.
//   • New option    → add the key to all relevant mode arrays AND add a wrapper
//                      element with id="adv-opt-{key}" in index.html.
const ADV_OPT_SUPPORT = {
  web: {
    'public-ip':  ['geo'],
    'dns':        [],
    'http':       ['insecure'],          // geo not applicable: HTTP results carry no server-IP geo data in this mode
    'port':       ['mtr-count'],          // geo not applicable: port-scan results do not include geo annotation
    'traceroute': ['mtr-count', 'geo'],
    '':           ['mtr-count', 'insecure', 'geo'],  // legacy all-in-one mode
  },
  smtp: {
    '':          ['mtr-count', 'insecure', 'geo'],
    'handshake': ['mtr-count', 'insecure', 'geo'],
    'auth':      ['mtr-count', 'insecure', 'geo'],
    'send':      ['mtr-count', 'insecure', 'geo'],
  },
  imap: { '': ['mtr-count', 'geo'] },
  pop:  { '': ['mtr-count', 'geo'] },
  ftp: {
    '':      ['mtr-count', 'insecure', 'geo'],
    'login': ['mtr-count', 'insecure', 'geo'],
    'list':  ['mtr-count', 'insecure', 'geo'],
  },
  sftp: {
    '':     ['mtr-count', 'geo'],
    'auth': ['mtr-count', 'geo'],
    'ls':   ['mtr-count', 'geo'],
  },
};

// The year this project was first published.  Used to build a copyright range
// that automatically extends as calendar years advance — e.g. "2026" in 2026,
// "2026–2027" in 2027, etc.  Only this constant ever needs to change.
const COPYRIGHT_START_YEAR = 2026;

// ── Theme constants ───────────────────────────────────────────────────────
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

// ── Namespace export ──────────────────────────────────────────────────────
PathProbe.Config = {
  MAP_POINT_CONFIGS,
  MARKER_COLOR_SCHEME_CONFIGS,
  MARKER_STYLE_CONFIGS,
  CONNECTOR_LINE_CONFIGS,
  CONNECTOR_GLOW_CONFIGS,
  MAP_TILE_VARIANTS,
  MAP_THEME_TO_TILE_VARIANT,
  MAP_DARK_THEMES,
  TILE_LAYER_CONFIGS,
  GEO_SAME_REGION_THRESHOLD_KM,
  TARGET_PORTS,
  TARGET_MODE_PANELS,
  WEB_MODES_WITH_PORTS,
  WEB_MODES_HIDE_HOST,
  WEB_MODES_SHOW_HTTP_URL,
  WEB_MODES_SHOW_DNS_DOMAINS,
  ADV_OPT_SUPPORT,
  COPYRIGHT_START_YEAR,
  THEMES,
  DEFAULT_THEME,
};
window.PathProbe = PathProbe;
})(); // end config IIFE
