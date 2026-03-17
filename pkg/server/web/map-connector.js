'use strict';
// ── map-connector.js — connector arc layer builders (PathProbe.MapConnector)
// Depends on: config.js (PathProbe.Config)
// All functions are pure builders that receive map/state as parameters.
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

// ── Colour / geometry utilities ───────────────────────────────────────────

/** Linearly interpolate between two hex colours.  t in [0, 1]. */
function lerpHex(hex1, hex2, t) {
  const parse = h => [parseInt(h.slice(1,3),16), parseInt(h.slice(3,5),16), parseInt(h.slice(5,7),16)];
  const toHex = n => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0');
  const [r1,g1,b1] = parse(hex1);
  const [r2,g2,b2] = parse(hex2);
  return '#' + toHex(r1+(r2-r1)*t) + toHex(g1+(g2-g1)*t) + toHex(b1+(b2-b1)*t);
}

/** Convert a '#rrggbb' hex colour and alpha [0,1] into an rgba() CSS string. */
function hexToRgba(hex, alpha) {
  const r = parseInt(hex.slice(1,3),16);
  const g = parseInt(hex.slice(3,5),16);
  const b = parseInt(hex.slice(5,7),16);
  return 'rgba(' + r + ',' + g + ',' + b + ',' + alpha + ')';
}

/** Generate lat/lon waypoints along a northward quadratic-bezier arc.
 *  Computed in Web-Mercator space so the curve is geometrically smooth.
 *  Returns an array of [lat, lon] pairs.
 */
function buildArcLatLngs(lat1, lon1, lat2, lon2, arcFactor, numSegments) {
  const R      = 6378137;
  const toMerc = (lat, lon) => ({
    x: lon * Math.PI / 180 * R,
    y: Math.log(Math.tan(Math.PI / 4 + lat * Math.PI / 360)) * R,
  });
  const fromMerc = (x, y) => [
    (2 * Math.atan(Math.exp(y / R)) - Math.PI / 2) * 180 / Math.PI,
    x / R * 180 / Math.PI,
  ];
  const m1   = toMerc(lat1, lon1);
  const m2   = toMerc(lat2, lon2);
  const midX = (m1.x + m2.x) / 2;
  const midY = (m1.y + m2.y) / 2;
  const dist = Math.sqrt((m2.x-m1.x)**2 + (m2.y-m1.y)**2);
  const ctlY = midY + arcFactor * dist;
  const pts  = [];
  for (let i = 0; i <= numSegments; i++) {
    const t = i / numSegments;
    const u = 1 - t;
    pts.push(fromMerc(
      u*u*m1.x + 2*u*t*midX + t*t*m2.x,
      u*u*m1.y + 2*u*t*ctlY + t*t*m2.y,
    ));
  }
  return pts;
}

/** Render one directional arrowhead as an inline SVG for a Leaflet divIcon. */
function buildArrowSVG(shape, sz, color, opacity, rotateDeg) {
  const f  = ' fill="'   + color + '" opacity="' + opacity + '"';
  const sw = ' stroke="' + color + '" opacity="' + opacity +
             '" stroke-linecap="round" fill="none" stroke-width="1.8"';
  let inner;
  switch (shape) {
    case 'fat':     inner = '<polygon points="0,0.5 10,5 0,9.5"'    + f  + '/>'; break;
    case 'chevron': inner = '<polygon points="0,0 8,5 0,10 2,5"'    + f  + '/>'; break;
    case 'double':  inner = '<polygon points="0,0 5,5 0,10 1,5"'    + f  + '/>' +
                            '<polygon points="4,0 10,5 4,10 5.5,5"' + f  + '/>'; break;
    case 'open':    inner = '<polyline points="0.5,1 9,5 0.5,9"'    + sw + '/>'; break;
    case 'pointer': inner = '<polygon points="0,1.5 8,5 0,8.5 3,5"' + f  + '/>'; break;
    default:        inner = '<polygon points="0,1 10,5 0,9"'         + f  + '/>'; break;
  }
  return '<svg width="' + sz + '" height="' + sz +
         '" viewBox="0 0 10 10" xmlns="http://www.w3.org/2000/svg">' +
         '<g transform="rotate(' + rotateDeg + ',5,5)">' + inner + '</g>' +
         '</svg>';
}

// ── ConnectorArcLayer ─────────────────────────────────────────────────────
// A Leaflet custom layer that draws the gradient arc connector on a dedicated
// HTML5 canvas in one drawing pass, eliminating sub-pixel gaps and doubled
// end-caps that appear with the N-polyline approach.
const ConnectorArcLayer = L.Layer.extend({
  initialize: function(pts, styleCfg, originColor, targetColor) {
    this._pts         = pts;
    this._styleCfg    = styleCfg;
    this._originColor = originColor;
    this._targetColor = targetColor;
    this._canvas      = null;
    this._onRedraw    = null;
  },

  onAdd: function(map) {
    this._map = map;
    const canvas = document.createElement('canvas');
    canvas.style.cssText = 'position:absolute;left:0;top:0;pointer-events:none;z-index:450;';
    map.getContainer().appendChild(canvas);
    this._canvas   = canvas;
    this._onRedraw = this._redraw.bind(this);
    map.on('move zoom zoomend resize', this._onRedraw);
    this._redraw();
    return this;
  },

  onRemove: function(map) {
    map.off('move zoom zoomend resize', this._onRedraw);
    if (this._canvas && this._canvas.parentNode) {
      this._canvas.parentNode.removeChild(this._canvas);
    }
    this._canvas = null; this._onRedraw = null; this._map = null;
  },

  _redraw: function() {
    if (!this._map || !this._canvas || !this._map._loaded) return;
    const map    = this._map;
    const size   = map.getSize();
    const canvas = this._canvas;
    const cfg    = this._styleCfg;
    const pts    = this._pts;
    if (pts.length < 2) return;
    canvas.width  = size.x;
    canvas.height = size.y;
    const sp  = pts.map(p => map.latLngToContainerPoint(L.latLng(p[0], p[1])));
    const ctx = canvas.getContext('2d');
    ctx.beginPath();
    ctx.moveTo(sp[0].x, sp[0].y);
    for (let i = 1; i < sp.length; i++) ctx.lineTo(sp[i].x, sp[i].y);
    const grad = ctx.createLinearGradient(
      sp[0].x, sp[0].y, sp[sp.length-1].x, sp[sp.length-1].y,
    );
    grad.addColorStop(0, hexToRgba(this._originColor, cfg.opacity));
    grad.addColorStop(1, hexToRgba(this._targetColor, cfg.opacity));
    ctx.strokeStyle = grad;
    ctx.lineWidth   = cfg.weight;
    ctx.lineCap     = 'round';
    ctx.lineJoin    = 'round';
    if (cfg.dashArray) {
      ctx.setLineDash(cfg.dashArray.split(/\s+/).map(Number));
    } else {
      ctx.setLineDash([]);
    }
    ctx.stroke();
  },
});

// ── ConnectorGlowLayer ────────────────────────────────────────────────────
// A Leaflet custom layer that animates a glowing "meteor" orb from origin
// to target along the arc, looping with a fade-out and brief dark pause.
const ConnectorGlowLayer = L.Layer.extend({
  initialize: function(pts, originColor, targetColor, glowCfg) {
    this._pts         = pts;
    this._originColor = originColor;
    this._targetColor = targetColor;
    this._glowCfg     = glowCfg;
    this._canvas      = null;
    this._rafId       = null;
    this._startTs     = null;
    this._scrPts      = null;
    this._cumDist     = null;
    this._needsResize = true;
    this._map         = null;
    this._onMapEvent  = null;
    this._boundTick   = null;
  },

  onAdd: function(map) {
    this._map = map;
    const canvas = document.createElement('canvas');
    canvas.style.cssText = 'position:absolute;left:0;top:0;pointer-events:none;z-index:452;';
    map.getContainer().appendChild(canvas);
    this._canvas      = canvas;
    this._needsResize = true;
    this._onMapEvent  = () => { this._scrPts = null; this._needsResize = true; };
    map.on('move zoom zoomend resize', this._onMapEvent);
    this._boundTick = this._tick.bind(this);
    this._rafId     = requestAnimationFrame(this._boundTick);
    return this;
  },

  onRemove: function(map) {
    if (this._onMapEvent) map.off('move zoom zoomend resize', this._onMapEvent);
    if (this._rafId !== null) { cancelAnimationFrame(this._rafId); this._rafId = null; }
    if (this._canvas && this._canvas.parentNode) {
      this._canvas.parentNode.removeChild(this._canvas);
    }
    this._canvas = null; this._scrPts = null; this._cumDist = null;
    this._map = null; this._onMapEvent = null; this._boundTick = null;
  },

  _tick: function(ts) {
    if (!this._map || !this._canvas || !this._map._loaded) {
      this._rafId = requestAnimationFrame(this._boundTick);
      return;
    }
    if (this._startTs === null) this._startTs = ts;
    if (this._needsResize) {
      const sz = this._map.getSize();
      this._canvas.width  = sz.x;
      this._canvas.height = sz.y;
      this._needsResize   = false;
    }
    const cfg      = this._glowCfg;
    const fadeMs   = cfg.fadeMs || 0;
    const cycleDur = cfg.travelMs + fadeMs + cfg.pauseMs;
    const elapsed  = (ts - this._startTs) % cycleDur;
    let progress, masterAlpha;
    const ctx = this._canvas.getContext('2d');
    ctx.clearRect(0, 0, this._canvas.width, this._canvas.height);
    if (elapsed < cfg.travelMs) {
      progress    = elapsed / cfg.travelMs;
      masterAlpha = 1.0;
    } else if (fadeMs > 0 && elapsed < cfg.travelMs + fadeMs) {
      progress    = 1.0;
      masterAlpha = 1.0 - (elapsed - cfg.travelMs) / fadeMs;
    } else {
      this._rafId = requestAnimationFrame(this._boundTick);
      return;
    }
    this._drawGlow(ctx, progress, masterAlpha);
    this._rafId = requestAnimationFrame(this._boundTick);
  },

  _getScreenPts: function() {
    if (this._scrPts) return this._scrPts;
    const map = this._map;
    this._scrPts = this._pts.map(p => map.latLngToContainerPoint(L.latLng(p[0], p[1])));
    const sp  = this._scrPts;
    const cum = [0];
    for (let i = 1; i < sp.length; i++) {
      const dx = sp[i].x - sp[i-1].x;
      const dy = sp[i].y - sp[i-1].y;
      cum.push(cum[i-1] + Math.sqrt(dx*dx + dy*dy));
    }
    this._cumDist = cum;
    return sp;
  },

  _posAtPx: function(sp, cum, targetPx) {
    const maxPx = cum[cum.length - 1];
    targetPx = Math.max(0, Math.min(targetPx, maxPx));
    let j = 0;
    while (j < sp.length - 2 && cum[j+1] < targetPx) j++;
    const segLen = cum[j+1] - cum[j];
    const t      = segLen > 0 ? (targetPx - cum[j]) / segLen : 0;
    return { x: sp[j].x + t*(sp[j+1].x - sp[j].x), y: sp[j].y + t*(sp[j+1].y - sp[j].y) };
  },

  _drawGlow: function(ctx, progress, masterAlpha) {
    const sp  = this._getScreenPts();
    const cum = this._cumDist;
    if (!sp || sp.length < 2 || !cum) return;
    const totalPx = cum[cum.length - 1];
    if (totalPx < 1) return;
    const cfg    = this._glowCfg;
    const headPx = progress * totalPx;
    const radius = cfg.glowRadius;
    const alpha  = cfg.glowOpacity * masterAlpha;
    // Comet tail — shrinks with masterAlpha, converges into head on fade-out.
    const tailPx    = (cfg.tailLength || 0.18) * totalPx * masterAlpha;
    const TAIL_SAMP = 18;
    for (let i = TAIL_SAMP; i > 0; i--) {
      const ratio    = i / TAIL_SAMP;
      const samplePx = Math.max(0, headPx - tailPx * ratio);
      const frac     = samplePx / totalPx;
      const pos      = this._posAtPx(sp, cum, samplePx);
      const color    = lerpHex(this._originColor, this._targetColor, frac);
      const a        = (1 - ratio) * alpha * 0.55;
      const r        = radius * (1 - ratio * 0.75);
      const grd      = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, r);
      grd.addColorStop(0, hexToRgba(color, a));
      grd.addColorStop(1, hexToRgba(color, 0));
      ctx.beginPath(); ctx.arc(pos.x, pos.y, r, 0, Math.PI*2);
      ctx.fillStyle = grd; ctx.fill();
    }
    // Outer glow halo.
    const headPos   = this._posAtPx(sp, cum, headPx);
    const headColor = lerpHex(this._originColor, this._targetColor, progress);
    const outerGrd  = ctx.createRadialGradient(headPos.x, headPos.y, 0, headPos.x, headPos.y, radius);
    outerGrd.addColorStop(0,    hexToRgba(headColor, alpha));
    outerGrd.addColorStop(0.45, hexToRgba(headColor, alpha * 0.45));
    outerGrd.addColorStop(1,    hexToRgba(headColor, 0));
    ctx.beginPath(); ctx.arc(headPos.x, headPos.y, radius, 0, Math.PI*2);
    ctx.fillStyle = outerGrd; ctx.fill();
    // Bright white core.
    const coreR   = Math.max(2, radius * 0.28);
    const coreGrd = ctx.createRadialGradient(headPos.x, headPos.y, 0, headPos.x, headPos.y, coreR);
    coreGrd.addColorStop(0,   hexToRgba('#ffffff', 0.96 * masterAlpha));
    coreGrd.addColorStop(0.6, hexToRgba(headColor, 0.85 * masterAlpha));
    coreGrd.addColorStop(1,   hexToRgba(headColor, 0));
    ctx.beginPath(); ctx.arc(headPos.x, headPos.y, coreR, 0, Math.PI*2);
    ctx.fillStyle = coreGrd; ctx.fill();
  },
});

// ── Layer builders ────────────────────────────────────────────────────────

/**
 * Build a LayerGroup of divIcon arrow symbols spaced at fixed screen-pixel
 * intervals along the arc. map is passed explicitly to avoid state coupling.
 */
function buildArrowConnectorLayer(map, pub, tgt, styleCfg, originColor, targetColor) {
  const pts   = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                 styleCfg.arcFactor, styleCfg.segments);
  const group = L.layerGroup();
  if (!(map && map._loaded) || pts.length < 2) return group;

  // Optional spine — thin gradient arc behind the arrow icons.
  const spineW = styleCfg.spineWeight || 0;
  if (spineW > 0) {
    const spineCfg = { weight: spineW, opacity: styleCfg.opacity, dashArray: null };
    group.addLayer(new ConnectorArcLayer(pts, spineCfg, originColor, targetColor));
  }

  const scrPts = pts.map(p => map.latLngToLayerPoint(L.latLng(p[0], p[1])));
  const n      = scrPts.length;
  const cum    = [0];
  for (let i = 1; i < n; i++) {
    const dx = scrPts[i].x - scrPts[i-1].x;
    const dy = scrPts[i].y - scrPts[i-1].y;
    cum.push(cum[i-1] + Math.sqrt(dx*dx + dy*dy));
  }
  const totalPx = cum[n-1];
  if (totalPx < 1) return group;

  const spacing = styleCfg.arrowSpacing || 40;
  const sz      = styleCfg.arrowSize    || 14;
  let   nextPx  = spacing / 2;
  let   j       = 0;

  while (nextPx < totalPx) {
    while (j < n - 2 && cum[j+1] < nextPx) j++;
    const segLen = cum[j+1] - cum[j];
    const t      = segLen > 0 ? (nextPx - cum[j]) / segLen : 0;
    const lat    = pts[j][0] + t * (pts[j+1][0] - pts[j][0]);
    const lon    = pts[j][1] + t * (pts[j+1][1] - pts[j][1]);
    const dx     = scrPts[j+1].x - scrPts[j].x;
    const dy     = scrPts[j+1].y - scrPts[j].y;
    const rotateDeg = Math.round(Math.atan2(dy, dx) * 180 / Math.PI);
    const color  = lerpHex(originColor, targetColor, nextPx / totalPx);
    L.marker([lat, lon], {
      icon: L.divIcon({
        className: 'connector-arrow-icon',
        html:      buildArrowSVG(styleCfg.arrowShape || 'triangle', sz, color, styleCfg.opacity, rotateDeg),
        iconSize:   [sz, sz],
        iconAnchor: [sz/2, sz/2],
      }),
      interactive: false, keyboard: false,
    }).addTo(group);
    nextPx += spacing;
  }
  return group;
}

/**
 * Build a LayerGroup for the origin→target connector arc.
 * map is passed explicitly; dispatch to arrow or polyline style as needed.
 */
function buildConnectorLayer(pub, tgt, styleCfg, originColor, targetColor, map) {
  const { CONNECTOR_GLOW_CONFIGS } = PathProbe.Config;
  let group;
  if ((styleCfg.type || 'polyline') === 'arrows') {
    group = buildArrowConnectorLayer(map, pub, tgt, styleCfg, originColor, targetColor);
  } else {
    const pts = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                 styleCfg.arcFactor, styleCfg.segments);
    group = L.layerGroup();
    group.addLayer(new ConnectorArcLayer(pts, styleCfg, originColor, targetColor));
  }
  if (styleCfg.glowEnabled) {
    const pts     = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                    styleCfg.arcFactor, styleCfg.segments);
    const glowCfg = CONNECTOR_GLOW_CONFIGS[styleCfg.glowConfig || 'default']
                  || CONNECTOR_GLOW_CONFIGS['default'];
    group.addLayer(new ConnectorGlowLayer(pts, originColor, targetColor, glowCfg));
  }
  return group;
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.MapConnector = {
  ConnectorArcLayer,
  ConnectorGlowLayer,
  buildConnectorLayer,
  lerpHex,
  hexToRgba,
  buildArcLatLngs,
};
