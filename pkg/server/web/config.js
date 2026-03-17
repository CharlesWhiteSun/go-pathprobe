'use strict';
// ── config.js — all application-wide constants (PathProbe.Config) ──────────
// This module has no dependencies on other PathProbe modules.
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

// ── Map point role configurations ─────────────────────────────────────────
// Each entry defines the CSS class and i18n label key used for a geocoded
// point on the Leaflet map.  Adding a new role requires only a new key here.
const MAP_POINT_CONFIGS = {
  origin: { cssClass: 'geo-marker--origin', i18nKey: 'map-origin', shortLabel: 'A' },
  target: { cssClass: 'geo-marker--target', i18nKey: 'map-target', shortLabel: 'B' },
};

// ── Marker colour scheme configurations ───────────────────────────────────
// Each entry provides originColor and targetColor for the two map roles.
// All marker CSS rules use var(--mc-origin) / var(--mc-target).
// To add a new scheme: add a key here + a matching i18n key.
const MARKER_COLOR_SCHEME_CONFIGS = {
  'ocean': { originColor: '#0891b2', targetColor: '#f59e0b', i18nKey: 'marker-color-ocean' },
};

// ── Marker style configurations ───────────────────────────────────────────
// buildHtml(roleCfg) returns an HTML string used as the Leaflet divIcon inner HTML.
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

// ── Connector line style configurations ──────────────────────────────────
// To add a new style: add a key here + matching translations in i18n.js.
const CONNECTOR_LINE_CONFIGS = {
  'tick-xs': {
    i18nKey: 'connector-tick-xs',
    arcFactor: 0.25, weight: 1, opacity: 0.85, dashArray: null, segments: 120,
    type: 'arrows', arrowShape: 'open', arrowSize: 4, arrowSpacing: 6,
    spineWeight: 0, glowEnabled: true, glowConfig: 'default',
  },
};

// ── Connector glow animation configurations ───────────────────────────────
// Controls the "meteor" light-pulse animation along the connector arc.
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

// Ordered list of map tile variant identifiers (left→right button layout).
const MAP_TILE_VARIANTS = ['light', 'osm', 'dark'];

// Maps each application theme to its default map tile variant.
const MAP_THEME_TO_TILE_VARIANT = {
  'default':       'light',
  'light-green':   'light',
  'deep-blue':     'dark',
  'forest-green':  'dark',
  'dark':          'dark',
};

// Theme IDs that use the dark tile variant.
const MAP_DARK_THEMES = new Set(['dark', 'deep-blue', 'forest-green']);

// Tile layer configurations keyed by variant ('light' | 'osm' | 'dark').
const TILE_LAYER_CONFIGS = {
  light: {
    url:         'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png',
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    i18nKey:     'map-tile-light',
    bgColor:     '#f5f5f0',
  },
  osm: {
    url:         'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
    i18nKey:     'map-tile-osm',
    bgColor:     '#f2efe9',
  },
  dark: {
    url:         'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png',
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    i18nKey:     'map-tile-dark',
    bgColor:     '#1a1a1a',
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

// ── Theme identifiers ─────────────────────────────────────────────────────
const THEMES = ['default', 'deep-blue', 'light-green', 'forest-green', 'dark'];
const DEFAULT_THEME = 'default';

// The year this project was first published (used for copyright range).
const COPYRIGHT_START_YEAR = 2026;

// ── Public API ────────────────────────────────────────────────────────────
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
  TARGET_PORTS,
  TARGET_MODE_PANELS,
  WEB_MODES_WITH_PORTS,
  TARGET_PLACEHOLDER_KEYS,
  THEMES,
  DEFAULT_THEME,
  COPYRIGHT_START_YEAR,
};
