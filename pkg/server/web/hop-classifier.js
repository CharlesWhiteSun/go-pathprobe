'use strict';
// ── hop-classifier.js — IP address scope classification ──────────────────
// Classifies IPv4 addresses into scope categories so the route-trace table
// can annotate private, loopback, and link-local hops with badges.
//
// Exported as: window.PathProbe.HopClassifier = { classifyIP }
//
// classifyIP(ip) returns one of:
//   'private'    — RFC-1918 address (LAN, home router, VPN inner, etc.)
//   'loopback'   — 127.0.0.0/8  (should not normally appear in a trace)
//   'link-local' — 169.254.0.0/16 (APIPA / auto-assigned)
//   null         — public (or unparseable) – no badge needed
//
// The function is pure (no I/O, no side-effects) and is safe to call from
// any goroutine / concurrently from multiple test fixtures.
(() => {
  /**
   * Classify an IPv4 address string into a named scope, or null for public IPs.
   * Only IPv4 is handled; IPv6 addresses always return null.
   *
   * @param {string} ip - dotted-decimal IPv4 address (e.g. "192.168.1.1")
   * @returns {'private'|'loopback'|'link-local'|null}
   */
  function classifyIP(ip) {
    if (!ip || typeof ip !== 'string') return null;

    const parts = ip.split('.');
    if (parts.length !== 4) return null;

    const octets = parts.map(Number);
    if (octets.some(n => !Number.isInteger(n) || n < 0 || n > 255)) return null;

    const [a, b] = octets;

    // 10.0.0.0/8
    if (a === 10) return 'private';

    // 172.16.0.0/12  (172.16 – 172.31)
    if (a === 172 && b >= 16 && b <= 31) return 'private';

    // 192.168.0.0/16
    if (a === 192 && b === 168) return 'private';

    // 127.0.0.0/8  — loopback
    if (a === 127) return 'loopback';

    // 169.254.0.0/16 — link-local (APIPA)
    if (a === 169 && b === 254) return 'link-local';

    return null; // public or unrecognised
  }

  const _ns = window.PathProbe || {};
  _ns.HopClassifier = { classifyIP };
  window.PathProbe = _ns;
})();
