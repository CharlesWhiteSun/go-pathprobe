'use strict';
// ── hop-classifier.js — IP address scope and class classification ─────────
// Classifies IPv4 / IPv6 addresses into scope categories and IPv4 classes
// so the route-trace table can annotate hops with informative type badges.
//
// Exported as: window.PathProbe.HopClassifier = { classifyIP, classifyIPTags }
//
// classifyIP(ip) — backward-compat single-scope classifier, returns one of:
//   'private'    — RFC-1918 address (LAN, home router, VPN inner, etc.)
//   'loopback'   — 127.0.0.0/8  (should not normally appear in a trace)
//   'link-local' — 169.254.0.0/16 (APIPA / auto-assigned)
//   null         — public (or unparseable) – no badge needed
//
// classifyIPTags(ip) — multi-tag classifier, returns an ordered array of keys:
//   IPv4: [classTag, scopeTag]  e.g. ['class-c', 'private'] or ['class-a', 'public']
//     classTag : 'class-a' | 'class-b' | 'class-c' | 'class-d' | 'class-e'
//     scopeTag : 'private' | 'loopback' | 'link-local' | 'cgnat' | 'multicast'
//                'reserved' | 'broadcast' | 'documentation' | '6to4-relay' | 'public'
//   IPv6: ['ipv6']
//   Unparseable / empty: []
//
//   Each key maps to i18n key 'hop-type-{key}' and CSS class 'hop-ip-badge--{key}'.
//
// The functions are pure (no I/O, no side-effects).
(() => {
  /**
   * Parse a dotted-decimal IPv4 string into four octets, or null if invalid.
   * @param {string} ip
   * @returns {number[]|null}
   */
  function _parseIPv4(ip) {
    if (!ip || typeof ip !== 'string') return null;
    const parts = ip.split('.');
    if (parts.length !== 4) return null;
    const octets = parts.map(Number);
    if (octets.some(n => !Number.isInteger(n) || n < 0 || n > 255)) return null;
    return octets;
  }

  /**
   * Classify an IPv4 address string into a named scope, or null for public IPs.
   * Only IPv4 is handled; IPv6 addresses always return null.
   *
   * @param {string} ip - dotted-decimal IPv4 address (e.g. "192.168.1.1")
   * @returns {'private'|'loopback'|'link-local'|null}
   */
  function classifyIP(ip) {
    const octs = _parseIPv4(ip);
    if (!octs) return null;
    const [a, b] = octs;

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

  /**
   * Classify an IP address into an ordered array of tag keys for display.
   *
   * @param {string} ip
   * @returns {string[]}
   */
  function classifyIPTags(ip) {
    if (!ip || typeof ip !== 'string') return [];

    // IPv6: detected by presence of a colon (full or compressed notation)
    if (ip.indexOf(':') !== -1) return ['ipv6'];

    const octs = _parseIPv4(ip);
    if (!octs) return [];

    const [a, b, c, d] = octs;

    // ── IPv4 Class (A–E) based on leading-bit boundaries ─────────────────
    let ipClass;
    if (a < 128)      ipClass = 'class-a';  // 0–127   (0xxxxxxx)
    else if (a < 192) ipClass = 'class-b';  // 128–191 (10xxxxxx)
    else if (a < 224) ipClass = 'class-c';  // 192–223 (110xxxxx)
    else if (a < 240) ipClass = 'class-d';  // 224–239 (1110xxxx) — Multicast
    else              ipClass = 'class-e';  // 240–255 (1111xxxx) — Reserved

    // ── Scope / purpose tag ───────────────────────────────────────────────
    let scopeTag;

    if (ipClass === 'class-d') {
      // Class D is entirely multicast (224.0.0.0/4)
      scopeTag = 'multicast';

    } else if (ipClass === 'class-e') {
      // 255.255.255.255 = limited broadcast; all other Class E = reserved
      scopeTag = (a === 255 && b === 255 && c === 255 && d === 255)
        ? 'broadcast' : 'reserved';

    } else if (a === 127) {
      scopeTag = 'loopback';                          // 127.0.0.0/8

    } else if (a === 169 && b === 254) {
      scopeTag = 'link-local';                        // 169.254.0.0/16 (APIPA)

    } else if (a === 100 && b >= 64 && b <= 127) {
      scopeTag = 'cgnat';                             // 100.64.0.0/10 (Shared / CGNAT)

    } else if (a === 10) {
      scopeTag = 'private';                           // 10.0.0.0/8

    } else if (a === 172 && b >= 16 && b <= 31) {
      scopeTag = 'private';                           // 172.16.0.0/12

    } else if (a === 192 && b === 168) {
      scopeTag = 'private';                           // 192.168.0.0/16

    } else if ((a === 192 && b === 0   && c === 2  ) ||
               (a === 198 && b === 51  && c === 100) ||
               (a === 203 && b === 0   && c === 113)) {
      scopeTag = 'documentation';                     // TEST-NET-1/2/3 (RFC 5737)

    } else if (a === 192 && b === 88 && c === 99) {
      scopeTag = '6to4-relay';                        // 192.88.99.0/24 (RFC 3068)

    } else {
      scopeTag = 'public';                            // routable public address
    }

    return [ipClass, scopeTag];
  }

  const _ns = window.PathProbe || {};
  _ns.HopClassifier = { classifyIP, classifyIPTags };
  window.PathProbe = _ns;
})();
