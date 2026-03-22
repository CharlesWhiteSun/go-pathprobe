package server_test

import (
	"strings"
	"testing"
)

// ── hop-classifier.js — static content tests ─────────────────────────────
// These tests verify the structure and content of hop-classifier.js without
// executing JavaScript in a real browser.  They confirm that the module
// exports the expected API, uses correct RFC-1918/loopback/link-local ranges,
// and is free of IP-range literals that belong in the HopClassifier module
// rather than in renderer.js.

// TestStaticJS_HopClassifierExports verifies that hop-classifier.js assigns
// a HopClassifier object to window.PathProbe and exposes a classifyIP function.
func TestStaticJS_HopClassifierExports(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "PathProbe.HopClassifier") {
		t.Error("hop-classifier.js: must assign result to window.PathProbe.HopClassifier")
	}
	if !strings.Contains(body, "classifyIP") {
		t.Error("hop-classifier.js: must define and export classifyIP")
	}
}

// TestStaticJS_HopClassifierPrivateRanges verifies that hop-classifier.js
// covers all three RFC-1918 private address ranges.
func TestStaticJS_HopClassifierPrivateRanges(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	ranges := []struct {
		desc string
		want string
	}{
		{"10.0.0.0/8 — single octet check", "a === 10"},
		{"172.16.0.0/12 — second octet range", "b >= 16 && b <= 31"},
		{"192.168.0.0/16 — two-octet check", "b === 168"},
	}
	for _, r := range ranges {
		if !strings.Contains(body, r.want) {
			t.Errorf("hop-classifier.js: missing %s range check (expected %q)", r.desc, r.want)
		}
	}
	// The word 'private' must appear as the return value for these ranges.
	if !strings.Contains(body, "'private'") {
		t.Error("hop-classifier.js: must return 'private' for RFC-1918 addresses")
	}
}

// TestStaticJS_HopClassifierSpecialRanges verifies that hop-classifier.js
// recognises loopback (127.0.0.0/8) and link-local (169.254.0.0/16) ranges.
func TestStaticJS_HopClassifierSpecialRanges(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	checks := []struct{ needle, label string }{
		{"a === 127", "loopback (127.0.0.0/8)"},
		{"'loopback'", "loopback return value"},
		{"a === 169", "link-local first octet (169.254.0.0/16)"},
		{"b === 254", "link-local second octet"},
		{"'link-local'", "link-local return value"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.needle) {
			t.Errorf("hop-classifier.js: missing %s check (expected %q)", c.label, c.needle)
		}
	}
}

// TestStaticJS_HopClassifierPublicReturnsNull verifies that hop-classifier.js
// returns null for public (or unparseable) addresses — no badge needed.
func TestStaticJS_HopClassifierPublicReturnsNull(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "return null") {
		t.Error("hop-classifier.js: must return null for public / unrecognised addresses")
	}
}

// TestStaticJS_HopClassifierGuardsInput verifies that classifyIP guards
// against non-string / null / IPv6 inputs without throwing.
func TestStaticJS_HopClassifierGuardsInput(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	// Must check that the input is a string.
	if !strings.Contains(body, "typeof ip") {
		t.Error("hop-classifier.js: classifyIP must check typeof ip before parsing")
	}
	// Must check part count to reject IPv6 or malformed strings.
	if !strings.Contains(body, "parts.length !== 4") {
		t.Error("hop-classifier.js: classifyIP must reject non-IPv4 strings (parts.length !== 4)")
	}
}

// TestStaticJS_HopClassifierServedByHandler verifies that the static file
// handler serves hop-classifier.js with HTTP 200 and a JS content type.
func TestStaticJS_HopClassifierServedByHandler(t *testing.T) {
	srv := newStaticHandler(t)
	body := fetchBody(t, srv, "/hop-classifier.js")
	if len(body) == 0 {
		t.Error("hop-classifier.js: handler returned empty body")
	}
	// Must declare 'use strict' for consistent JS mode.
	if !strings.Contains(body, "'use strict'") {
		t.Error("hop-classifier.js: should start with 'use strict'")
	}
}

// TestStaticJS_I18nHopClassificationKeys verifies that i18n.js defines the
// translation keys referenced by _hopIpBadge in renderer.js.
func TestStaticJS_I18nHopClassificationKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	keys := []string{
		"'hop-type-private'",
		"'hop-type-loopback'",
		"'hop-type-link-local'",
		"'hop-timeout-tip'",
	}
	for _, k := range keys {
		if !strings.Contains(body, k) {
			t.Errorf("i18n.js: missing hop-classification i18n key %s", k)
		}
	}
}

// TestStaticJS_I18nRouteStatsKeys verifies that both i18n locales define all
// translation keys required by the route statistics summary card.
func TestStaticJS_I18nRouteStatsKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	statsKeys := []string{
		"'route-stats-title'",
		"'route-stats-total'",
		"'route-stats-responsive'",
		"'route-stats-timeout'",
		"'route-stats-avg-loss'",
		"'route-stats-max-rtt'",
		"'route-stats-countries'",
		"'route-stats-reached'",
		"'route-stats-not-reached'",
	}
	for _, k := range statsKeys {
		if !strings.Contains(body, k) {
			t.Errorf("i18n.js: missing route-stats i18n key %s", k)
		}
	}
}

// TestStaticJS_I18nSeparateIPHostnameKeys verifies that i18n.js has separate
// 'th-ip' and 'th-hostname' keys (split from the old combined 'th-ip-host').
func TestStaticJS_I18nSeparateIPHostnameKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	for _, k := range []string{"'th-ip'", "'th-hostname'"} {
		if !strings.Contains(body, k) {
			t.Errorf("i18n.js: missing separate column key %s", k)
		}
	}
}

// TestStaticHTML_HopClassifierLoaded verifies that index.html loads
// hop-classifier.js before renderer.js so classifyIP is available when
// _hopIpBadge is first called by appendLiveHop.
func TestStaticHTML_HopClassifierLoaded(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, "hop-classifier.js") {
		t.Error("index.html: must load hop-classifier.js")
	}
	// hop-classifier.js must appear before renderer.js in the script order.
	classifierIdx := strings.Index(body, "hop-classifier.js")
	rendererIdx := strings.Index(body, "renderer.js")
	if classifierIdx == -1 || rendererIdx == -1 {
		t.Fatal("index.html: both hop-classifier.js and renderer.js must be present")
	}
	if classifierIdx > rendererIdx {
		t.Error("index.html: hop-classifier.js must be loaded BEFORE renderer.js")
	}
}

// TestStaticHTML_LiveTableSplitColumns verifies that the live progress table
// in index.html uses 'th-ip' and 'th-hostname' columns rather than the old
// combined 'th-ip-host'.
func TestStaticHTML_LiveTableSplitColumns(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `data-i18n="th-ip"`) {
		t.Error("index.html: live-progress table must include th-ip column")
	}
	if !strings.Contains(body, `data-i18n="th-hostname"`) {
		t.Error("index.html: live-progress table must include th-hostname column")
	}
}

// TestStaticHTML_LiveTableTypeColumn verifies that the live progress table
// in index.html includes the th-type column header for IP classification tags.
func TestStaticHTML_LiveTableTypeColumn(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `data-i18n="th-type"`) {
		t.Error("index.html: live-progress table must include th-type column for IP classification")
	}
}

// TestStaticJS_HopClassifierClassifyIPTagsExported verifies that
// classifyIPTags is defined and exported alongside classifyIP.
func TestStaticJS_HopClassifierClassifyIPTagsExported(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "classifyIPTags") {
		t.Error("hop-classifier.js: classifyIPTags must be defined and exported")
	}
	// Both functions must appear in the exported HopClassifier object.
	if !strings.Contains(body, "classifyIPTags }") {
		t.Error("hop-classifier.js: both classifyIP and classifyIPTags must appear in the HopClassifier export")
	}
}

// TestStaticJS_HopClassifierIPv4Classes verifies that classifyIPTags covers
// all five IPv4 classes (A–E) using first-octet boundary thresholds.
func TestStaticJS_HopClassifierIPv4Classes(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	boundaries := []struct{ needle, label string }{
		{"'class-a'", "class-a tag"},
		{"'class-b'", "class-b tag"},
		{"'class-c'", "class-c tag"},
		{"'class-d'", "class-d tag"},
		{"'class-e'", "class-e tag"},
		{"a < 128", "Class A upper boundary (< 128)"},
		{"a < 192", "Class B upper boundary (< 192)"},
		{"a < 224", "Class C upper boundary (< 224)"},
		{"a < 240", "Class D upper boundary (< 240)"},
	}
	for _, b := range boundaries {
		if !strings.Contains(body, b.needle) {
			t.Errorf("hop-classifier.js: missing %s (expected %q)", b.label, b.needle)
		}
	}
}

// TestStaticJS_HopClassifierCGNAT verifies that classifyIPTags recognises
// the CGNAT / Shared Address range 100.64.0.0/10 (RFC 6598).
func TestStaticJS_HopClassifierCGNAT(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "'cgnat'") {
		t.Error("hop-classifier.js: must return 'cgnat' for 100.64.0.0/10 range")
	}
	if !strings.Contains(body, "b >= 64 && b <= 127") {
		t.Error("hop-classifier.js: must check second-octet range (b >= 64 && b <= 127) for CGNAT")
	}
}

// TestStaticJS_HopClassifierDocumentation verifies that classifyIPTags handles
// the three TEST-NET documentation ranges defined in RFC 5737.
func TestStaticJS_HopClassifierDocumentation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "'documentation'") {
		t.Error("hop-classifier.js: must return 'documentation' for TEST-NET ranges")
	}
	testNets := []struct{ needle, label string }{
		{"b === 0   && c === 2", "TEST-NET-1 (192.0.2.0/24)"},
		{"b === 51  && c === 100", "TEST-NET-2 (198.51.100.0/24)"},
		{"b === 0   && c === 113", "TEST-NET-3 (203.0.113.0/24)"},
	}
	for _, tn := range testNets {
		if !strings.Contains(body, tn.needle) {
			t.Errorf("hop-classifier.js: missing %s check (expected %q)", tn.label, tn.needle)
		}
	}
}

// TestStaticJS_HopClassifierIPv6 verifies that classifyIPTags returns the
// 'ipv6' tag for IPv6 addresses by detecting a colon in the string.
func TestStaticJS_HopClassifierIPv6(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "'ipv6'") {
		t.Error("hop-classifier.js: must return 'ipv6' tag for IPv6 addresses")
	}
	if !strings.Contains(body, "indexOf(':') !== -1") {
		t.Error("hop-classifier.js: must detect IPv6 via indexOf(':') colon check")
	}
}

// TestStaticJS_HopClassifierPublicTag verifies that classifyIPTags explicitly
// returns a 'public' tag for ordinary routable addresses.
func TestStaticJS_HopClassifierPublicTag(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/hop-classifier.js")

	if !strings.Contains(body, "'public'") {
		t.Error("hop-classifier.js: must return 'public' for routable public addresses")
	}
}

// TestStaticJS_I18nHopTypeExtendedKeys verifies that i18n.js defines all
// extended hop classification keys added by classifyIPTags.
func TestStaticJS_I18nHopTypeExtendedKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	keys := []string{
		"'th-type'",
		"'hop-type-class-a'", "'hop-type-class-b'", "'hop-type-class-c'",
		"'hop-type-class-d'", "'hop-type-class-e'",
		"'hop-type-public'", "'hop-type-cgnat'", "'hop-type-multicast'",
		"'hop-type-reserved'", "'hop-type-broadcast'",
		"'hop-type-documentation'", "'hop-type-6to4-relay'", "'hop-type-ipv6'",
	}
	for _, k := range keys {
		if !strings.Contains(body, k) {
			t.Errorf("i18n.js: missing extended hop-type key %s", k)
		}
	}
}
