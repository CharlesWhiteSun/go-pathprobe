'use strict';
// ── map-connector.js — gradient arc & animated glow connector module ─────────
// Extracted from app.js (sub-task 3.7).
// Exposes: PathProbe.MapConnector = { buildConnectorLayer }
// All internal helpers (lerpHex, hexToRgba, buildArcLatLngs, buildArrowSVG,
// isMapLoaded, ConnectorArcLayer, ConnectorGlowLayer, buildArrowConnectorLayer)
// are private to this IIFE and do not leak into the global scope.
(() => {

  // ── Private colour helpers ─────────────────────────────────────────────────

  /** Linearly interpolate between two hex colours.
   *  t is a value in [0, 1]; 0 returns hex1, 1 returns hex2.
   */
  function lerpHex(hex1, hex2, t) {
    const parse = h => [parseInt(h.slice(1, 3), 16), parseInt(h.slice(3, 5), 16), parseInt(h.slice(5, 7), 16)];
    const toHex = n => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0');
    const [r1, g1, b1] = parse(hex1);
    const [r2, g2, b2] = parse(hex2);
    return '#' + toHex(r1 + (r2 - r1) * t) + toHex(g1 + (g2 - g1) * t) + toHex(b1 + (b2 - b1) * t);
  }

  /** Convert a '#rrggbb' hex colour and an alpha value [0,1] into an rgba() CSS
   *  string suitable for use as a canvas strokeStyle or fillStyle.
   */
  function hexToRgba(hex, alpha) {
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    return 'rgba(' + r + ',' + g + ',' + b + ',' + alpha + ')';
  }

  // ── Private arc geometry ───────────────────────────────────────────────────

  /** Generate lat/lon waypoints along a northward quadratic-bezier arc.
   *  The Bézier control point is computed in Web-Mercator (EPSG:3857) space so
   *  the rendered curve appears geometrically smooth on the Leaflet Mercator map
   *  regardless of geographic scale or latitude.  arcFactor scales the control-
   *  point offset as a fraction of the Mercator straight-line distance.
   *  Returns an array of [lat, lon] pairs.
   */
  function buildArcLatLngs(lat1, lon1, lat2, lon2, arcFactor, numSegments) {
    // Helpers: geographic ↔ Web-Mercator (metres, EPSG:3857).
    const R        = 6378137;
    const toMerc   = (lat, lon) => ({
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
    const dist = Math.sqrt((m2.x - m1.x) ** 2 + (m2.y - m1.y) ** 2);
    // Control point bowed northward (positive Y in Mercator = north on the map).
    const ctlX = midX;
    const ctlY = midY + arcFactor * dist;
    const pts  = [];
    for (let i = 0; i <= numSegments; i++) {
      const t = i / numSegments;
      const u = 1 - t;
      pts.push(fromMerc(
        u * u * m1.x + 2 * u * t * ctlX + t * t * m2.x,
        u * u * m1.y + 2 * u * t * ctlY + t * t * m2.y,
      ));
    }
    return pts;
  }

  // ── Private arrow SVG ──────────────────────────────────────────────────────

  /** Render one directional arrowhead as an inline SVG element for use inside a
   *  Leaflet divIcon.  All shapes are defined on a normalised 10×10 viewBox and
   *  scaled to sz×sz screen pixels.  The SVG's own transform attribute rotates
   *  around the viewBox centre (5,5) so the anchor stays pixel-perfect.
   *
   *  shape values: 'triangle' | 'fat' | 'chevron' | 'double' | 'open' | 'pointer'
   */
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
    return '<svg width="'  + sz + '" height="' + sz +
           '" viewBox="0 0 10 10" xmlns="http://www.w3.org/2000/svg">' +
           '<g transform="rotate(' + rotateDeg + ',5,5)">' + inner + '</g>' +
           '</svg>';
  }

  // ── Private map guard ──────────────────────────────────────────────────────

  /** Returns true when the given Leaflet map instance has been both created and
   *  initialised (setView / fitBounds has been called).  Operations that require
   *  a loaded map — such as latLngToLayerPoint() — must guard with isMapLoaded()
   *  to avoid Leaflet throwing "Set map center and zoom first."
   *  Accepts mapInstance as a parameter so this module has no dependency on any
   *  global state variable.
   */
  function isMapLoaded(mapInstance) {
    return Boolean(mapInstance && mapInstance._loaded);
  }

  // ── ConnectorArcLayer ──────────────────────────────────────────────────────

  /** ConnectorArcLayer — a Leaflet custom layer that draws the gradient arc
   *  connector on a dedicated HTML5 canvas in ONE drawing pass.
   *
   *  This is the correct architecture for seamless dot/dash patterns with a
   *  gradient colour.  Drawing the full arc as a single canvas path with
   *  createLinearGradient and setLineDash completely eliminates the three
   *  failure modes of the old N-polyline approach:
   *
   *    1. Sub-pixel gaps between adjacent SVG/canvas polyline segments.
   *    2. Doubled round end-caps where two segments meet (doubled dot at
   *       every junction point).
   *    3. Floating-point drift in the per-segment dashOffset accumulation.
   *
   *  Lifecycle
   *  ---------
   *  onAdd    Creates a <canvas> element inside the map container, sized to
   *           the visible viewport, and binds map event listeners.
   *  _redraw  Projects arc waypoints via latLngToContainerPoint(), builds one
   *           canvas path, applies a linear gradient (createLinearGradient)
   *           and an optional dot/dash rhythm (setLineDash), then strokes.
   *           Runs on every 'move', 'zoom', 'zoomend', and 'resize' event
   *           so the arc always tracks the live viewport.
   *  onRemove Unregisters listeners and removes the <canvas> element.
   */
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
      // Place the canvas directly in the map container (not inside a pane)
      // so it is never affected by the CSS transforms Leaflet applies to
      // panes during animated pan / zoom.  z-index 450 puts it above the
      // overlayPane (400) but below the markerPane (600).
      canvas.style.cssText =
        'position:absolute;left:0;top:0;pointer-events:none;z-index:450;';
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
      this._canvas   = null;
      this._onRedraw = null;
      this._map      = null;
    },

    _redraw: function() {
      if (!this._map || !this._canvas || !this._map._loaded) return;
      const map    = this._map;
      const size   = map.getSize();
      const canvas = this._canvas;
      const cfg    = this._styleCfg;
      const pts    = this._pts;
      if (pts.length < 2) return;

      // Resize canvas to cover the current viewport exactly.
      canvas.width  = size.x;
      canvas.height = size.y;

      // Project geographic arc waypoints to container-relative pixel coords.
      const sp  = pts.map(p => map.latLngToContainerPoint(L.latLng(p[0], p[1])));
      const ctx = canvas.getContext('2d');

      // Single continuous arc path — no segment boundaries, no repeated end-caps.
      ctx.beginPath();
      ctx.moveTo(sp[0].x, sp[0].y);
      for (let i = 1; i < sp.length; i++) ctx.lineTo(sp[i].x, sp[i].y);

      // Linear gradient from arc start to arc end gives a smooth colour flow
      // that closely follows the arc direction without per-segment complexity.
      const grad = ctx.createLinearGradient(
        sp[0].x, sp[0].y, sp[sp.length - 1].x, sp[sp.length - 1].y,
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

  // ── ConnectorGlowLayer ─────────────────────────────────────────────────────

  /** ConnectorGlowLayer — a Leaflet custom layer that animates a glowing "meteor"
   *  orb travelling from origin to target along the arc and looping with a brief
   *  pause at the destination.  Runs entirely on a dedicated HTML5 canvas using
   *  requestAnimationFrame so it never blocks the UI thread.
   *
   *  Visual anatomy
   *  --------------
   *  • Comet tail  — a series of fading radial-gradient circles receding behind
   *                  the head, shrinking and becoming transparent toward the back.
   *  • Outer glow  — a large radial-gradient circle centered on the current head
   *                  position, providing the "light source" halo effect.
   *  • Bright core — a small, near-white radial-gradient circle at the very front
   *                  that gives the impression of intense illumination.
   *
   *  Colour interpolates from originColor to targetColor as the orb travels,
   *  matching the arc's own gradient palette for visual coherence.
   *
   *  Lifecycle (same contract as ConnectorArcLayer)
   *  -----------------------------------------------
   *  onAdd    Creates a canvas above the arc layer, starts the rAF loop.
   *  onRemove Cancels the loop, removes the canvas, resets all state.
   */
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
      // z-index 452: above ConnectorArcLayer (450) but below markers (600).
      canvas.style.cssText =
        'position:absolute;left:0;top:0;pointer-events:none;z-index:452;';
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
      this._canvas     = null;
      this._scrPts     = null;
      this._cumDist    = null;
      this._map        = null;
      this._onMapEvent = null;
      this._boundTick  = null;
    },

    _tick: function(ts) {
      if (!this._map || !this._canvas || !this._map._loaded) {
        this._rafId = requestAnimationFrame(this._boundTick);
        return;
      }
      if (this._startTs === null) this._startTs = ts;
      // Resize the canvas lazily whenever the viewport changes.
      if (this._needsResize) {
        const sz          = this._map.getSize();
        this._canvas.width  = sz.x;
        this._canvas.height = sz.y;
        this._needsResize   = false;
      }
      const cfg      = this._glowCfg;
      const fadeMs   = cfg.fadeMs   || 0;
      const cycleDur = cfg.travelMs + fadeMs + cfg.pauseMs;
      const elapsed  = (ts - this._startTs) % cycleDur;

      // Three-phase animation cycle:
      //   Phase 1 — travel  : head moves 0 → 1 along the arc at full brightness.
      //   Phase 2 — fade-out: head stays at destination, tail converges, all light
      //                        dissolves to 0 ("extinguish / breathing-lamp" effect).
      //   Phase 3 — dark    : canvas is blank until the next cycle begins.
      let progress;    // head position along arc [0, 1]
      let masterAlpha; // global brightness/opacity multiplier [0, 1]

      const ctx = this._canvas.getContext('2d');
      ctx.clearRect(0, 0, this._canvas.width, this._canvas.height);

      if (elapsed < cfg.travelMs) {
        // Phase 1: travelling.
        progress    = elapsed / cfg.travelMs;
        masterAlpha = 1.0;
      } else if (fadeMs > 0 && elapsed < cfg.travelMs + fadeMs) {
        // Phase 2: fade-out — head fixed at destination, linear brightness ramp.
        progress    = 1.0;
        masterAlpha = 1.0 - (elapsed - cfg.travelMs) / fadeMs;
      } else {
        // Phase 3: dark — canvas already cleared above, nothing to paint.
        this._rafId = requestAnimationFrame(this._boundTick);
        return;
      }

      this._drawGlow(ctx, progress, masterAlpha);
      this._rafId = requestAnimationFrame(this._boundTick);
    },

    /** Return cached screen-pixel projection; recomputes when invalidated. */
    _getScreenPts: function() {
      if (this._scrPts) return this._scrPts;
      const map = this._map;
      this._scrPts = this._pts.map(p =>
        map.latLngToContainerPoint(L.latLng(p[0], p[1])));
      const sp  = this._scrPts;
      const cum = [0];
      for (let i = 1; i < sp.length; i++) {
        const dx = sp[i].x - sp[i - 1].x;
        const dy = sp[i].y - sp[i - 1].y;
        cum.push(cum[i - 1] + Math.sqrt(dx * dx + dy * dy));
      }
      this._cumDist = cum;
      return sp;
    },

    /** Interpolate a screen-space {x,y} position at targetPx along the arc. */
    _posAtPx: function(sp, cum, targetPx) {
      const maxPx = cum[cum.length - 1];
      targetPx    = Math.max(0, Math.min(targetPx, maxPx));
      let j = 0;
      while (j < sp.length - 2 && cum[j + 1] < targetPx) j++;
      const segLen = cum[j + 1] - cum[j];
      const t      = segLen > 0 ? (targetPx - cum[j]) / segLen : 0;
      return {
        x: sp[j].x + t * (sp[j + 1].x - sp[j].x),
        y: sp[j].y + t * (sp[j + 1].y - sp[j].y),
      };
    },

    /** Draw the comet tail, outer glow halo, and bright core at `progress`.
     *  masterAlpha [0, 1] is a global brightness multiplier applied to every
     *  opacity value.  When it ramps from 1 → 0 during the fade-out phase:
     *    • tailPx shrinks proportionally so the tail converges into the head.
     *    • All radial-gradient alphas fade to 0, creating the extinguish effect.
     */
    _drawGlow: function(ctx, progress, masterAlpha) {
      const sp  = this._getScreenPts();
      const cum = this._cumDist;
      if (!sp || sp.length < 2 || !cum) return;
      const totalPx = cum[cum.length - 1];
      if (totalPx < 1) return;

      const cfg    = this._glowCfg;
      const headPx = progress * totalPx;
      const radius = cfg.glowRadius;
      // Scale base opacity by masterAlpha so all elements fade in unison.
      const alpha  = cfg.glowOpacity * masterAlpha;

      // ── Comet tail: shrinks with masterAlpha → converges into the head on fade-out ──
      const tailPx   = (cfg.tailLength || 0.18) * totalPx * masterAlpha;
      const TAIL_SAMP = 18;
      for (let i = TAIL_SAMP; i > 0; i--) {
        const ratio    = i / TAIL_SAMP;          // 1 = farthest back, 0 ≈ head
        const samplePx = Math.max(0, headPx - tailPx * ratio);
        const frac     = samplePx / totalPx;
        const pos      = this._posAtPx(sp, cum, samplePx);
        const color    = lerpHex(this._originColor, this._targetColor, frac);
        const a        = (1 - ratio) * alpha * 0.55;
        const r        = radius * (1 - ratio * 0.75);
        const grd      = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, r);
        grd.addColorStop(0, hexToRgba(color, a));
        grd.addColorStop(1, hexToRgba(color, 0));
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, r, 0, Math.PI * 2);
        ctx.fillStyle = grd;
        ctx.fill();
      }

      // ── Outer glow halo ──────────────────────────────────────────────────────
      const headPos   = this._posAtPx(sp, cum, headPx);
      const headColor = lerpHex(this._originColor, this._targetColor, progress);
      const outerGrd  = ctx.createRadialGradient(
        headPos.x, headPos.y, 0, headPos.x, headPos.y, radius,
      );
      outerGrd.addColorStop(0,    hexToRgba(headColor, alpha));
      outerGrd.addColorStop(0.45, hexToRgba(headColor, alpha * 0.45));
      outerGrd.addColorStop(1,    hexToRgba(headColor, 0));
      ctx.beginPath();
      ctx.arc(headPos.x, headPos.y, radius, 0, Math.PI * 2);
      ctx.fillStyle = outerGrd;
      ctx.fill();

      // ── Bright white core ────────────────────────────────────────────────────
      const coreR   = Math.max(2, radius * 0.28);
      const coreGrd = ctx.createRadialGradient(
        headPos.x, headPos.y, 0, headPos.x, headPos.y, coreR,
      );
      coreGrd.addColorStop(0,   hexToRgba('#ffffff', 0.96 * masterAlpha));
      coreGrd.addColorStop(0.6, hexToRgba(headColor, 0.85 * masterAlpha));
      coreGrd.addColorStop(1,   hexToRgba(headColor, 0));
      ctx.beginPath();
      ctx.arc(headPos.x, headPos.y, coreR, 0, Math.PI * 2);
      ctx.fillStyle = coreGrd;
      ctx.fill();
    },
  });

  // ── Private: arrow-icon connector ─────────────────────────────────────────

  /** Build a Leaflet LayerGroup containing gradient-coloured divIcon arrow symbols
   *  distributed along the arc at a fixed screen-pixel spacing.  By working in
   *  pixel space the visual density is consistent at every zoom level and
   *  geographic scale, avoiding the sparse "   >    >    >   " appearance that
   *  arises when symbols are placed at equal geographic intervals.  Each symbol
   *  is rotated to align with the local arc tangent direction.
   *  Used by styles with type === 'arrows'.
   *
   *  mapInstance is passed explicitly so this function has no dependency on any
   *  global variable and can be unit-tested without a global _map reference.
   */
  function buildArrowConnectorLayer(pub, tgt, styleCfg, originColor, targetColor, mapInstance) {
    const pts = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                 styleCfg.arcFactor, styleCfg.segments);
    const group = L.layerGroup();
    if (!isMapLoaded(mapInstance) || pts.length < 2) return group;

    // ── Optional spine ── a thin gradient polyline rendered behind the arrow icons.
    // When spineWeight > 0 the full arc path is always visible even at wide icon
    // spacing, preventing a "floating arrows in empty space" appearance.
    const spineW = styleCfg.spineWeight || 0;
    if (spineW > 0) {
      // Spine uses ConnectorArcLayer (single canvas path) for the same seamless
      // gradient rendering quality as the dot-family connector styles.
      const spineCfg = { weight: spineW, opacity: styleCfg.opacity, dashArray: null };
      group.addLayer(new ConnectorArcLayer(pts, spineCfg, originColor, targetColor));
    }

    // Convert all arc waypoints to screen pixels (consistent with current zoom/pan).
    const scrPts = pts.map(p => mapInstance.latLngToLayerPoint(L.latLng(p[0], p[1])));
    const n      = scrPts.length;

    // Build a cumulative pixel-distance table so each symbol can be placed at a
    // precise fixed screen-pixel interval regardless of segment length variation.
    const cum = [0];
    for (let i = 1; i < n; i++) {
      const dx = scrPts[i].x - scrPts[i - 1].x;
      const dy = scrPts[i].y - scrPts[i - 1].y;
      cum.push(cum[i - 1] + Math.sqrt(dx * dx + dy * dy));
    }
    const totalPx = cum[n - 1];
    if (totalPx < 1) return group;

    const spacing = styleCfg.arrowSpacing || 40; // screen-px between successive symbols
    const sz      = styleCfg.arrowSize    || 14;
    let   nextPx  = spacing / 2;                 // start centred in first interval
    let   j       = 0;

    while (nextPx < totalPx) {
      // Advance segment pointer until the current segment straddles nextPx.
      while (j < n - 2 && cum[j + 1] < nextPx) j++;
      const segLen = cum[j + 1] - cum[j];
      const t      = segLen > 0 ? (nextPx - cum[j]) / segLen : 0;
      // Interpolated geographic position for the marker anchor.
      const lat = pts[j][0] + t * (pts[j + 1][0] - pts[j][0]);
      const lon = pts[j][1] + t * (pts[j + 1][1] - pts[j][1]);
      // Screen-space tangent for accurate per-symbol rotation (origin → target).
      const dx        = scrPts[j + 1].x - scrPts[j].x;
      const dy        = scrPts[j + 1].y - scrPts[j].y;
      const rotateDeg = Math.round(Math.atan2(dy, dx) * 180 / Math.PI);
      // Gradient colour proportional to pixel distance along the arc.
      const color = lerpHex(originColor, targetColor, nextPx / totalPx);
      L.marker([lat, lon], {
        icon: L.divIcon({
          className: 'connector-arrow-icon',
          html: buildArrowSVG(
            styleCfg.arrowShape || 'triangle', sz, color, styleCfg.opacity, rotateDeg),
          iconSize:   [sz, sz],
          iconAnchor: [sz / 2, sz / 2],
        }),
        interactive: false,
        keyboard:    false,
      }).addTo(group);
      nextPx += spacing;
    }
    return group;
  }

  // ── Public: build connector layer ─────────────────────────────────────────

  /** Build a Leaflet LayerGroup that draws the origin→target connector arc.
   *  Dispatches to buildArrowConnectorLayer() for styles with type === 'arrows'.
   *  For polyline styles, a single ConnectorArcLayer is used: the whole arc is
   *  drawn on one HTML5 canvas path with createLinearGradient + setLineDash,
   *  giving a seamless gradient dot/dash pattern with no inter-segment gaps.
   *  When styleCfg.glowEnabled is true a ConnectorGlowLayer is added on top,
   *  rendering the meteor light-pulse animation without coupling to the base arc.
   *
   *  mapInstance is passed explicitly so this module has no dependency on any
   *  global variable and can be called safely before a global _map is set.
   */
  function buildConnectorLayer(pub, tgt, styleCfg, originColor, targetColor, mapInstance) {
    let group;
    if ((styleCfg.type || 'polyline') === 'arrows') {
      group = buildArrowConnectorLayer(pub, tgt, styleCfg, originColor, targetColor, mapInstance);
    } else {
      const pts = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                   styleCfg.arcFactor, styleCfg.segments);
      group = L.layerGroup();
      group.addLayer(new ConnectorArcLayer(pts, styleCfg, originColor, targetColor));
    }
    // Overlay the glowing meteor animation when the style opts in.
    if (styleCfg.glowEnabled) {
      const pts = buildArcLatLngs(pub.Lat, pub.Lon, tgt.Lat, tgt.Lon,
                                  styleCfg.arcFactor, styleCfg.segments);
      // Resolve CONNECTOR_GLOW_CONFIGS at call time from PathProbe.Config so
      // that map-connector.js has no hard dependency on config.js load order.
      const glowConfigs = (window.PathProbe && window.PathProbe.Config &&
                           window.PathProbe.Config.CONNECTOR_GLOW_CONFIGS) || {};
      const glowCfg = glowConfigs[styleCfg.glowConfig || 'default']
                    || glowConfigs['default'] || {};
      group.addLayer(new ConnectorGlowLayer(pts, originColor, targetColor, glowCfg));
    }
    return group;
  }

  // ── Export ─────────────────────────────────────────────────────────────────
  const _ns = window.PathProbe || {};
  _ns.MapConnector = { buildConnectorLayer };
  window.PathProbe = _ns;
})();
