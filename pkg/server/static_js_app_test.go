package server_test

import (
	"strings"
	"testing"
)

// ── Web mode radio-button tests ───────────────────────────────────────────

// TestStaticJS_BrandSystemRemoved verifies that the brand style management
// system has been removed from app.js now that the logo style is fixed.
func TestStaticJS_BrandSystemRemoved(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	for _, absent := range []string{
		"BRAND_STYLES",
		"toggleBrandPicker",
		"initBrandStyle",
	} {
		if strings.Contains(body, absent) {
			t.Errorf("app.js: brand system symbol %q must not be present", absent)
		}
	}
}

// ── animation & error-message tests ──────────────────────────────────────

// TestStaticJS_DotsRunAnimation verifies that app.js always injects the
// dots animation markup into #run-btn and that the picker system has been
// removed in favour of the fixed dots choice.
func TestStaticJS_DotsRunAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// getRunningHTML must exist and emit the dots markup.
	if !strings.Contains(body, "getRunningHTML") {
		t.Error("app.js: getRunningHTML must be defined")
	}
	if !strings.Contains(body, "anim-dots") {
		t.Error("app.js: getRunningHTML must return anim-dots markup")
	}
	// Picker management functions must have been removed.
	for _, removed := range []string{"RUN_ANIMATIONS", "initRunAnimation", "setRunAnimation", "_syncAnimPicker"} {
		if strings.Contains(body, removed) {
			t.Errorf("app.js: removed animation picker symbol %q must not be present", removed)
		}
	}
	// picker HTML must not be present.
	if strings.Contains(body, "id=\"anim-picker\"") {
		t.Error("index.html: #anim-picker must have been removed")
	}
}

// TestStaticJS_AppConfigDefensiveAccess verifies that app.js accesses
// PathProbe.Config through the explicit window.PathProbe property rather than
// a bare PathProbe identifier.
//
// Background: bare identifier lookup in a classic browser script throws
// ReferenceError when window.PathProbe was never set (e.g. when the browser
// cache serves an old index.html that lacks the config.js <script> tag).
// window.PathProbe property access safely returns undefined instead of
// throwing, preventing a catastrophic script failure that would leave
// setTheme() and setLocale() uncallable from inline onclick attributes.
func TestStaticJS_AppConfigDefensiveAccess(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Must access config through window.PathProbe, not via a bare PathProbe
	// identifier that throws ReferenceError when config.js did not run.
	if !strings.Contains(body, "window.PathProbe") {
		t.Error("app.js: must access config through window.PathProbe to avoid " +
			"ReferenceError when config.js fails to execute")
	}

	// Must use a defensive fallback (|| {} or ?? {}) so the destructuring
	// never throws even when window.PathProbe.Config is unavailable.
	if !strings.Contains(body, "|| {}") && !strings.Contains(body, "?? {}") {
		t.Error("app.js: config alias block must use a defensive fallback (|| {} or ?? {}) " +
			"to prevent crashing when config.js is unavailable")
	}

	// THEMES must have an explicit fallback default inside the destructuring
	// so that applyTheme() / initTheme() can safely call THEMES.includes()
	// even when config.js failed to load.
	if !strings.Contains(body, "THEMES") || !strings.Contains(body, "'default'") {
		t.Error("app.js: THEMES must carry a fallback default value in the config alias block")
	}

	// setTheme and setLocale must be defined in app.js as top-level function
	// declarations so they are accessible from inline onclick attributes.
	for _, fn := range []string{"function setTheme(", "function setLocale("} {
		if !strings.Contains(body, fn) {
			t.Errorf("app.js: %q must be a top-level function declaration", fn)
		}
	}
}
