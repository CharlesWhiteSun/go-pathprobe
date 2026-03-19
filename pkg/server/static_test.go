package server_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// TestStaticJS_DefaultThemeConstant verifies that app.js references DEFAULT_THEME
// (destructured from PathProbe.Config, declared in config.js) and that
// initTheme() reads the HTML data-default-theme attribute as its authoritative
// fallback source rather than relying on a hard-coded string literal.
func TestStaticJS_DefaultThemeConstant(t *testing.T) {
	h := newStaticHandler(t)

	// app.js must still reference DEFAULT_THEME (destructured from PathProbe.Config).
	// The constant declaration lives in config.js; TestStaticJS_ConfigNamespace
	// verifies the declaration there.
	appRec := httptest.NewRecorder()
	h.ServeHTTP(appRec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if appRec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", appRec.Code)
	}
	if !strings.Contains(appRec.Body.String(), "DEFAULT_THEME") {
		t.Error("app.js: DEFAULT_THEME must be referenced (expected via PathProbe.Config destructuring)")
	}

	// theme.js owns initTheme(); verify it reads the HTML attribute for the
	// server-declared default and validates against THEMES.
	themeRec := httptest.NewRecorder()
	h.ServeHTTP(themeRec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if themeRec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", themeRec.Code)
	}
	themeBody := themeRec.Body.String()

	if !strings.Contains(themeBody, "dataset.defaultTheme") {
		t.Error("theme.js: initTheme() must read document.documentElement.dataset.defaultTheme")
	}
	if !strings.Contains(themeBody, "themes.includes(htmlDefault)") {
		t.Error("theme.js: initTheme() must validate htmlDefault against the themes list before use")
	}
}

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

// TestStaticJS_CustomSelectFunctions verifies that app.js defines
// initCustomSelect(), selectItem() logic, and the _initTargetDone guard that
// prevents the entrance animation from firing on the cold page load.
func TestStaticJS_CustomSelectFunctions(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	if !strings.Contains(body, "initCustomSelect") {
		t.Error("form.js: initCustomSelect function must be defined")
	}
	if !strings.Contains(body, "_initTargetDone") {
		t.Error("form.js: _initTargetDone guard must be present to skip animation on cold page load")
	}
	if !strings.Contains(body, "panel-entering") {
		t.Error("form.js: onTargetChange must manage the panel-entering CSS class for the entrance animation")
	}
	// Custom select must sync the hidden native select so val('target') stays valid.
	if !strings.Contains(body, "select.value") {
		t.Error("form.js: initCustomSelect must sync the hidden native select .value")
	}
	// Keyboard navigation arrows must be wired.
	if !strings.Contains(body, "ArrowDown") || !strings.Contains(body, "ArrowUp") {
		t.Error("form.js: initCustomSelect must handle ArrowDown and ArrowUp keyboard navigation")
	}
	// has-selection class must be managed to give persistent primary-border indicator.
	if !strings.Contains(body, "has-selection") {
		t.Error("form.js: initCustomSelect must add 'has-selection' class to .cs-wrap for persistent selection indicator")
	}
	// close() must accept a restoreFocus parameter so outside clicks don't steal focus.
	if !strings.Contains(body, "restoreFocus") {
		t.Error("form.js: close() in initCustomSelect must accept a restoreFocus parameter")
	}
	// Document click handler must call close(false) to avoid stealing focus on outside click.
	if !strings.Contains(body, "close(false)") {
		t.Error("form.js: outside-click document listener must call close(false) to avoid focus theft")
	}
}

// ── Footer tests ─────────────────────────────────────────────────────────

// ── Select option theming tests ───────────────────────────────────────────

// ── Animation control tests ───────────────────────────────────────────────

// TestStaticJS_PanelLeaveAnimation verifies that app.js manages the
// panel-leaving class in onTargetChange() so the departing panel animates out
// before being hidden.  Also checks that the _pendingReveal cleanup guard
// exists for safe rapid target-switching.
func TestStaticJS_PanelLeaveAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	if !strings.Contains(body, "panel-leaving") {
		t.Error("form.js: onTargetChange must add 'panel-leaving' class to the departing panel for the exit animation")
	}
	// animationend fires after the CSS animation completes; hiding then keeps layout clean.
	if !strings.Contains(body, "animationend") {
		t.Error("form.js: onTargetChange must listen for animationend to hide the departing panel after its exit animation")
	}
	// Rapid-switch guard: _pendingReveal cleanup cancels in-flight transitions.
	if !strings.Contains(body, "_pendingReveal") {
		t.Error("form.js: _pendingReveal must be defined to cancel in-flight transitions on rapid target switching")
	}
	// Toggle functions must be absent — vivid mode is now the HTML-level default.
	for _, sym := range []string{"cycleAnim", "initAnim", "applyAnim", "ANIM_MODES"} {
		if strings.Contains(body, sym) {
			t.Errorf("form.js: %s must be removed; animation mode is now a static HTML attribute, not a runtime toggle", sym)
		}
	}
}

// TestStaticJS_PanelSequentialTransition verifies that app.js implements a
// strictly sequential panel transition in onTargetChange(): the incoming panel
// is kept hidden (incoming.hidden = true) while the departing panel is still
// animating, ensuring the two panels never coexist in the layout flow.  The
// test also confirms the revealIncoming helper function exists to decouple the
// "show new panel" step from the departure listener, and that _pendingReveal
// stores a cleanup callback that can be invoked by a subsequent call to cancel
// the in-flight transition and prevent a stale reveal from running.
func TestStaticJS_PanelSequentialTransition(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// The incoming panel must be explicitly hidden while the outgoing panel
	// is animating so both panels never occupy layout space at the same time.
	if !strings.Contains(body, "incoming.hidden = true") {
		t.Error("form.js: incoming.hidden must be set to true during the departure phase to prevent simultaneous layout overlap")
	}
	// revealIncoming encapsulates the deferred show+animate step and is the
	// sole entry point for making the incoming panel visible.
	if !strings.Contains(body, "revealIncoming") {
		t.Error("form.js: revealIncoming helper must be defined to decouple the reveal step from the animationend listener")
	}
	// _pendingReveal stores the listener cleanup for the in-flight transition
	// so that a rapid switch can cancel the previous departure and reveal.
	if !strings.Contains(body, "_pendingReveal") {
		t.Error("form.js: _pendingReveal cleanup variable must store the cancel function for the active transition")
	}
	// removeEventListener must be called inside the cleanup to stop stale
	// animationend handlers from triggering an outdated revealIncoming.
	if !strings.Contains(body, "removeEventListener") {
		t.Error("form.js: cleanup must call removeEventListener to prevent stale animationend handlers from triggering on rapid switch")
	}
	// Height animation: measurePanelHeight must exist to off-screen-measure the
	// incoming panel before revealing it.
	if !strings.Contains(body, "measurePanelHeight") {
		t.Error("form.js: measurePanelHeight function must be defined to measure the incoming panel height off-screen")
	}
	// measurePanelHeight must include CSS margins in the returned value so the
	// stage height transition target matches the panel's true occupied layout
	// space and does not jump when height: auto is restored afterwards.
	if !strings.Contains(body, "getComputedStyle") || !strings.Contains(body, "marginBottom") {
		t.Error("form.js: measurePanelHeight must use getComputedStyle to include marginTop/marginBottom in the height total")
	}
	// measurePanelHeight must use clone.offsetHeight (not clone.scrollHeight).
	// offsetHeight includes the element's border, while scrollHeight does not;
	// the parent stage's scrollHeight accounts for the child's full offsetHeight,
	// so using scrollHeight would leave the stage 2 px short (border top+bottom),
	// causing a visible snap when height:auto is restored at animation end.
	if !strings.Contains(body, "clone.offsetHeight") {
		t.Error("form.js: measurePanelHeight must use clone.offsetHeight (includes border) not clone.scrollHeight to avoid a 2px snap at animation end")
	}
	// stage.scrollHeight captures the current panel height before locking it.
	if !strings.Contains(body, "stage.scrollHeight") {
		t.Error("form.js: stage.scrollHeight must be read to capture current height before locking for the transition")
	}
	// stage.offsetWidth is passed to measurePanelHeight to simulate the correct layout width.
	if !strings.Contains(body, "stage.offsetWidth") {
		t.Error("form.js: stage.offsetWidth must be passed to measurePanelHeight to simulate the correct layout width")
	}
	// stage.style.height must be set and then cleared after the transition.
	if !strings.Contains(body, "stage.style.height = ") {
		t.Error("form.js: stage.style.height must be set during the height transition")
	}
	if !strings.Contains(body, "stage.style.height = ''") {
		t.Error("form.js: stage.style.height must be cleared to auto after the panel transition completes")
	}
}

// TestStaticJS_EmptyPanelHandling verifies that app.js honours the
// data-panel-empty attribute: when the target resolves to a content-free
// fieldset (imap, pop) all departing panels are still hidden and the stage
// height collapses smoothly, but the blank fieldset is never made visible so
// the user is never presented with an empty bordered box.
func TestStaticJS_EmptyPanelHandling(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// JS reads dataset.panelEmpty to decide whether to reveal the incoming panel.
	if !strings.Contains(body, "dataset.panelEmpty") {
		t.Error("form.js: onTargetChange must read dataset.panelEmpty to detect content-free panels")
	}
	// isEmptyPanel is the local flag derived from the attribute.
	if !strings.Contains(body, "isEmptyPanel") {
		t.Error("form.js: onTargetChange must define isEmptyPanel flag to branch the reveal path")
	}
	// When the incoming panel is empty the stage height target must be 0 so the
	// stage collapses smoothly rather than leaving residual whitespace.
	if !strings.Contains(body, "isEmptyPanel ? 0") {
		t.Error("form.js: empty panel transition must use incomingH=0 to collapse the stage smoothly")
	}
	// revealIncoming must guard on isEmptyPanel and return early without
	// unhiding the blank fieldset.
	if !strings.Contains(body, "if (isEmptyPanel)") {
		t.Error("form.js: revealIncoming must check isEmptyPanel and return early without showing the blank fieldset")
	}
}

// TestStaticJS_EmptyToContentTransition verifies that app.js smoothly
// animates the stage height from 0 to the incoming panel height when
// switching from a content-free panel (e.g. pop → ftp).  Without this
// branch the stage jumps directly from height:0 to height:auto, causing
// the card border to appear instantly instead of growing in.
func TestStaticJS_EmptyToContentTransition(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// The grow-from-empty branch must lock the stage at '0px' before
	// triggering the transition, so CSS has an explicit start value to
	// animate from (auto→auto never animates).
	if !strings.Contains(body, "stage.style.height = '0px'") {
		t.Error("form.js: grow-from-empty branch must set stage.style.height = '0px' to give CSS transition an explicit start value")
	}
	// The branch must measure the incoming panel so the stage knows its
	// target height before the transition starts.
	if !strings.Contains(body, "!isEmptyPanel && stage") {
		t.Error("form.js: grow-from-empty branch must guard on !isEmptyPanel && stage to ensure it only runs for content panels")
	}
}

// TestStaticJS_AdvancedOptsAnimation verifies that app.js defines
// initAdvancedOpts() and implements the expected animated expand/collapse
// behaviour: intercepts summary clicks, drives height transition and
// adv-entering / adv-leaving CSS classes, and calls transitionend cleanup.
func TestStaticJS_AdvancedOptsAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// Core function must be defined and wired up from form.js init().
	if !strings.Contains(body, "initAdvancedOpts") {
		t.Error("form.js: initAdvancedOpts function must be defined to wire up the Advanced Options animation")
	}
	// The function must look up the details element by id.
	if !strings.Contains(body, `getElementById('advanced-opts')`) {
		t.Error("form.js: initAdvancedOpts must find the details element via getElementById('advanced-opts')")
	}
	// CSS classes adv-entering and adv-leaving drive the animations.
	if !strings.Contains(body, "adv-entering") {
		t.Error("form.js: initAdvancedOpts must apply adv-entering class on expand")
	}
	if !strings.Contains(body, "adv-leaving") {
		t.Error("form.js: initAdvancedOpts must apply adv-leaving class on collapse")
	}
	// details.open must be managed manually so the browser does not instantly
	// show/hide content before the animation can run.
	if !strings.Contains(body, "details.open") {
		t.Error("form.js: initAdvancedOpts must manage details.open manually to prevent instant browser toggle")
	}
	// transitionend cleanup ensures height:auto is restored after the animation
	// so the panel can resize naturally (e.g. if the viewport width changes).
	if !strings.Contains(body, "transitionend") {
		t.Error("form.js: initAdvancedOpts must listen for transitionend to restore height:auto after animation")
	}
	// e.preventDefault() prevents the browser from toggling open/closed natively.
	if !strings.Contains(body, "e.preventDefault") {
		t.Error("form.js: initAdvancedOpts click handler must call e.preventDefault() to suppress native toggle")
	}
	// adv-is-open class controls the chevron rotation and must be added at
	// expand-start and removed at collapse-start (not at transitionend) so
	// the chevron rotation is always in sync with the height animation.
	if !strings.Contains(body, "adv-is-open") {
		t.Error("form.js: initAdvancedOpts must manage adv-is-open class to drive the chevron rotation in sync with height animation")
	}
}

// ── Phase 4: traceroute API field assertions ──────────────────────────────

// TestStaticJS_WebModeTracerouteBuildOpts verifies that the embedded api-builder.js
// handles the "traceroute" mode in buildWebOpts() and forwards max_hops into
// the API request payload so the server's WebOptions.MaxHops is populated.
func TestStaticJS_WebModeTracerouteBuildOpts(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// The traceroute mode string constant must appear in buildWebOpts.
	if !strings.Contains(body, `'traceroute'`) {
		t.Error("api-builder.js: 'traceroute' mode string must appear in buildWebOpts")
	}
	// The max_hops JSON field must be written into the request opts.
	if !strings.Contains(body, "max_hops") {
		t.Error("api-builder.js: buildWebOpts must include max_hops in the traceroute mode branch")
	}
	// The traceroute sub-panel ID must exist in config.js TARGET_MODE_PANELS (data layer).
	cfgBody := fetchBody(t, newStaticHandler(t), "/config.js")
	if !strings.Contains(cfgBody, "web-fields-traceroute") {
		t.Error("config.js: TARGET_MODE_PANELS.web must include 'web-fields-traceroute' entry")
	}
}

// ── Phase 5: traceroute result rendering assertions ───────────────────────

// TestStaticJS_RenderRouteSection verifies that app.js defines a
// renderRouteSection function and wires it into renderReport so route hops
// are shown in the results pane when a traceroute diagnostic is returned.
func TestStaticJS_RenderRouteSection(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	// The render function must be defined.
	if !strings.Contains(body, "renderRouteSection") {
		t.Error("renderer.js: renderRouteSection function must be defined")
	}
	// It must be invoked from renderReport with the Route field.
	if !strings.Contains(body, "renderRouteSection(r.Route)") {
		t.Error("renderer.js: renderReport must call renderRouteSection(r.Route)")
	}
	// The route section heading i18n key must be referenced.
	if !strings.Contains(body, "'section-route'") {
		t.Error("renderer.js: renderRouteSection must reference i18n key 'section-route'")
	}
	// Timed-out hop indicator must be present.
	if !strings.Contains(body, "hop-timedout") {
		t.Error("renderer.js: renderRouteSection must apply 'hop-timedout' class to timed-out hops")
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

// TestStaticJS_ErrorClearsProgressLog verifies that app.js clears and hides
// the progress log both on SSE error events and on network-level failures, so
// partial traceroute output does not remain visible below the error banner.
func TestStaticJS_ErrorClearsProgressLog(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Both the catch block and the SSE error handler must clear innerHTML.
	// We verify by counting occurrences of the clear pattern.
	clearPattern := "progressEl.innerHTML = ''"
	count := strings.Count(body, clearPattern)
	if count < 2 {
		t.Errorf("app.js: progressEl.innerHTML='' must appear in both the catch block and the SSE error handler; found %d occurrence(s)", count)
	}
}

// TestStaticJS_TracerouteTimeoutAutoCompute verifies that app.js contains the
// logic to auto-compute a traceroute-appropriate timeout before sending the
// diagnostic request, preventing spurious deadline-exceeded errors.
func TestStaticJS_TracerouteTimeoutAutoCompute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// The traceroute-specific timeout guard must be present.
	if !strings.Contains(body, "traceroute") {
		t.Error("api-builder.js: must contain 'traceroute' reference for mode-specific timeout logic")
	}
	// parseTimeoutSec must be defined to compare user timeout vs worst-case minimum.
	if !strings.Contains(body, "parseTimeoutSec") {
		t.Error("api-builder.js: parseTimeoutSec helper must be defined for timeout comparison")
	}
	// The auto-compute formula (maxHops * mtrCount * 2 + 15) must be present.
	if !strings.Contains(body, "maxHops * mtrCount * 2 + 15") {
		t.Error("api-builder.js: traceroute timeout auto-compute must use formula maxHops * mtrCount * 2 + 15")
	}
}

// TestStaticJS_LocalizeError verifies that app.js defines the localizeError
// function to map raw server error strings to user-friendly i18n messages,
// replacing opaque Go internal strings like "context deadline exceeded".
func TestStaticJS_LocalizeError(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	if !strings.Contains(body, "localizeError") {
		t.Error("app.js: localizeError function must be defined")
	}
	// Must check for the deadline exceeded pattern.
	if !strings.Contains(body, "deadline exceeded") {
		t.Error("app.js: localizeError must handle 'deadline exceeded' error pattern")
	}
	// Must use the err-timeout i18n key for timeout errors.
	if !strings.Contains(body, "err-timeout") {
		t.Error("app.js: localizeError must reference 'err-timeout' i18n key for timeout errors")
	}
}

// TestStaticJS_WebPortModeReadsTextInput verifies that app.js handles the
// web/port mode using the shared text input (val('ports')) instead of the
// removed checkbox picker.  getWebPorts() must no longer exist in the codebase.
func TestStaticJS_WebPortModeReadsTextInput(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// getWebPorts() has been removed; buildRequest reads _val('ports') for web/port.
	if strings.Contains(body, "function getWebPorts(") {
		t.Error("api-builder.js: getWebPorts() must be removed; web/port mode now uses the shared text input")
	}
	// buildRequest must use _webModesWithPorts() (runtime-resolved) to decide whether to read ports.
	if !strings.Contains(body, "_webModesWithPorts().includes(mode)") {
		t.Error("api-builder.js: buildRequest must guard web port reading with _webModesWithPorts().includes(mode)")
	}
	// ports-text-group: the shared text input read path for web/port mode must be documented.
	if !strings.Contains(body, "ports-text-group") {
		t.Error("api-builder.js: comment must reference 'ports-text-group' to document the shared text input read path")
	}
	// The removed picker elements must not be referenced in JS logic.
	if strings.Contains(body, "getElementById('port-other-cb')") {
		t.Error("api-builder.js: port-other-cb has been removed and must not be referenced")
	}
	if strings.Contains(body, "getElementById('port-other-num')") {
		t.Error("api-builder.js: port-other-num has been removed and must not be referenced")
	}
}

// TestStaticJS_WebTargetPortDefaults verifies that TARGET_PORTS.web includes
// both port 80 (HTTP) and port 443 (HTTPS) as the auto-fill defaults shown
// when the user selects web target + port connectivity mode.
func TestStaticJS_WebTargetPortDefaults(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// TARGET_PORTS.web must include both HTTP (80) and HTTPS (443) defaults.
	if !strings.Contains(body, "web:  [80, 443]") {
		t.Error("config.js: TARGET_PORTS.web must be [80, 443] (HTTP + HTTPS defaults for port-connectivity mode)")
	}
}

// TestStaticJS_PortGroupModeAutoFill verifies that app.js auto-fills the ports
// text input when the user switches a web radio to the port-connectivity mode
// (mirrors the auto-fill onTargetChange() already does for target switches).
func TestStaticJS_PortGroupModeAutoFill(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// The radio change handler must auto-fill ports for web/port mode.
	if !strings.Contains(body, "_webModesWithPorts().includes(mode)") {
		t.Error("form.js: radio change handler must check _webModesWithPorts().includes(mode) to auto-fill ports")
	}
	// Must respect the userEdited guard so manual entries are preserved.
	if !strings.Contains(body, `dataset.userEdited !== 'true'`) {
		t.Error("form.js: radio change handler must respect dataset.userEdited guard before auto-filling")
	}
}

// TestStaticJS_PortGroupToggle verifies that app.js manages #port-group
// visibility via updatePortGroup(), which is driven by the WEB_MODES_WITH_PORTS
// constant so logic is data-driven rather than hardcoded per-mode.
func TestStaticJS_PortGroupToggle(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// The unified port-group ID must be referenced.
	if !strings.Contains(body, "port-group") {
		t.Error("form.js: must reference 'port-group' to toggle Ports column visibility")
	}
	// updatePortGroup must be callable from both onTargetChange and the radio handler.
	if !strings.Contains(body, "updatePortGroup(") {
		t.Error("form.js: updatePortGroup() must be called from onTargetChange and radio change handler")
	}
}

// TestStaticJS_WEB_MODES_WITH_PORTS verifies that config.js declares the
// WEB_MODES_WITH_PORTS constant used to drive port-group visibility in a
// data-driven, non-hardcoded manner.
func TestStaticJS_WEB_MODES_WITH_PORTS(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "WEB_MODES_WITH_PORTS") {
		t.Error("config.js: WEB_MODES_WITH_PORTS constant must be declared")
	}
	// Port connectivity mode must be listed as requiring port selection.
	if !strings.Contains(body, "'port'") {
		t.Error("config.js: WEB_MODES_WITH_PORTS must include 'port' mode")
	}
}

// TestStaticJS_UpdatePortGroup verifies that app.js declares the updatePortGroup()
// function which manages visibility of #port-group and its inner variants.
func TestStaticJS_UpdatePortGroup(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	if !strings.Contains(body, "function updatePortGroup(") {
		t.Error("form.js: updatePortGroup() function must be defined")
	}
	// Must reference all three DOM elements it manages.
	for _, id := range []string{"port-group", "ports-text-group"} {
		if !strings.Contains(body, id) {
			t.Errorf("form.js: updatePortGroup() must reference element #%s", id)
		}
	}
	// The removed checkbox picker must no longer be referenced in updatePortGroup.
	if strings.Contains(body, "getElementById('web-port-picker')") {
		t.Error("form.js: web-port-picker has been removed; updatePortGroup must not reference it")
	}
}

// TestStaticJS_RenderMapInvalidateSize verifies that renderMap() defers a call
// to _map.invalidateSize() via requestAnimationFrame so Leaflet re-projects all
// tiles after the #results section transitions from display:none to display:block.
// Without this, Leaflet sees a 0×0 container at init time and leaves large
// blank grey areas on the OpenStreetMap canvas.
func TestStaticJS_RenderMapInvalidateSize(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	// renderMap must call invalidateSize to correct blank-tile regression.
	if !strings.Contains(body, "invalidateSize") {
		t.Error("map.js: renderMap must call _map.invalidateSize() to fix blank tile regression when container was hidden")
	}
	// The call must be deferred via requestAnimationFrame so it runs after the
	// browser has re-laid-out the newly visible container.
	if !strings.Contains(body, "requestAnimationFrame") {
		t.Error("map.js: invalidateSize must be deferred via requestAnimationFrame so layout is complete before tiles repaint")
	}
}

// TestStaticJS_SSEResultRevealOrder verifies that in the SSE 'result' event
// handler, resultEl.hidden = false is set BEFORE renderMap() is called.
// Leaflet initialises by reading the container's layout dimensions; if the
// parent #results section is still hidden (display:none) at that point, the
// map gets a 0×0 size and tiles are blank.
func TestStaticJS_SSEResultRevealOrder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Locate handleSSEMessage and the evtName==='result' branch within it.
	fnStart := strings.Index(body, "function handleSSEMessage(")
	if fnStart == -1 {
		t.Fatal("app.js: handleSSEMessage function not found")
	}
	resultBranchIdx := strings.Index(body[fnStart:], "evtName === 'result'")
	if resultBranchIdx == -1 {
		t.Fatal("app.js: evtName === 'result' branch not found in handleSSEMessage")
	}
	// Inspect a window large enough to cover the result branch body.
	windowStart := fnStart + resultBranchIdx
	window := body[windowStart : windowStart+600]

	hiddenIdx := strings.Index(window, "resultEl.hidden = false")
	renderMapIdx := strings.Index(window, "renderMap(")
	if hiddenIdx == -1 {
		t.Fatal("app.js: resultEl.hidden = false not found in SSE result branch")
	}
	if renderMapIdx == -1 {
		t.Fatal("app.js: renderMap( not found in SSE result branch")
	}
	if hiddenIdx > renderMapIdx {
		t.Error("app.js: resultEl.hidden = false must appear BEFORE renderMap() in the SSE result handler — " +
			"#results must be visible so the Leaflet container has layout dimensions")
	}
}

// TestStaticJS_HistoryEntryRevealOrder verifies that in loadHistoryEntry(),
// resultEl.hidden = false is set BEFORE renderMap() for the same reason as
// in the SSE handler.
func TestStaticJS_HistoryEntryRevealOrder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	fnStart := strings.Index(body, "function loadHistoryEntry(")
	if fnStart == -1 {
		t.Fatal("app.js: loadHistoryEntry function not found")
	}
	// Bound the search to the function body (next top-level function boundary).
	// nextFn is relative to body[fnStart:], so add fnStart to get absolute end.
	nextFn := strings.Index(body[fnStart+1:], "\nasync function ")
	var fnBody string
	if nextFn != -1 && (fnStart+1+nextFn) <= len(body) {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		// Fallback: take up to 1200 chars, capped at body length.
		end := fnStart + 1200
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	hiddenIdx := strings.Index(fnBody, "resultEl.hidden = false")
	renderMapIdx := strings.Index(fnBody, "renderMap(")
	if hiddenIdx == -1 {
		t.Fatal("app.js: resultEl.hidden = false not found in loadHistoryEntry")
	}
	if renderMapIdx == -1 {
		t.Fatal("app.js: renderMap( not found in loadHistoryEntry")
	}
	if hiddenIdx > renderMapIdx {
		t.Error("app.js: resultEl.hidden = false must appear BEFORE renderMap() in loadHistoryEntry — " +
			"#results must be visible so the Leaflet container has layout dimensions")
	}
}

// TestStaticJS_MapPointConfigs verifies that app.js declares MAP_POINT_CONFIGS
// with 'origin' and 'target' keys, forming a data-driven foundation for all
// map marker styling.  Callers derive visual behaviour from this object rather
// than hardcoding logic inside renderMap().
func TestStaticJS_MapPointConfigs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_POINT_CONFIGS") {
		t.Error("config.js: MAP_POINT_CONFIGS constant not found — map marker config must be data-driven")
	}
	if !strings.Contains(body, "'origin'") {
		t.Error("config.js: MAP_POINT_CONFIGS must include an 'origin' key for the public-IP marker")
	}
	if !strings.Contains(body, "'target'") {
		t.Error("config.js: MAP_POINT_CONFIGS must include a 'target' key for the destination marker")
	}
}

// TestStaticJS_HaversineKm verifies that app.js defines a haversineKm()
// helper for computing the great-circle distance.  This powers the distance
// badge displayed below the map between origin and target markers.
func TestStaticJS_HaversineKm(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function haversineKm(") {
		t.Error("map.js: haversineKm function not found — distance calculation must be a named helper")
	}
	// Earth radius constant must appear to confirm correct formula.
	if !strings.Contains(body, "6371") {
		t.Error("map.js: haversineKm must use Earth radius constant 6371 km")
	}
}

// TestStaticJS_BuildMarkerIcon verifies that app.js defines buildMarkerIcon()
// which creates L.divIcon instances driven by MAP_POINT_CONFIGS, replacing the
// default Leaflet marker pin with a role-coloured dot.
func TestStaticJS_BuildMarkerIcon(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function buildMarkerIcon(") {
		t.Error("map.js: buildMarkerIcon function not found — marker icon creation must be a named helper")
	}
	if !strings.Contains(body, "L.divIcon(") {
		t.Error("map.js: buildMarkerIcon must use L.divIcon for custom marker styling")
	}
}

// TestStaticJS_BuildPopupHtml verifies that app.js defines buildPopupHtml()
// which constructs a rich HTML popup from a GeoAnnotation, using the
// geo-popup__role badge to clearly identify origin vs target.
func TestStaticJS_BuildPopupHtml(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function buildPopupHtml(") {
		t.Error("map.js: buildPopupHtml function not found")
	}
	if !strings.Contains(body, "geo-popup__role") {
		t.Error("map.js: buildPopupHtml must emit .geo-popup__role element for visual role identification")
	}
	if !strings.Contains(body, "geo-popup__ip") {
		t.Error("map.js: buildPopupHtml must emit .geo-popup__ip element for the IP address")
	}
}

// TestStaticJS_RenderMapPolyline verifies that renderMap() draws a connector
// between origin and target to give users a clear visual probe direction.
// The connector is now rendered by ConnectorArcLayer (HTML5 canvas) rather
// than a raw L.polyline, so the test checks for buildConnectorLayer() and
// that dot/dash rhythms are configured via dashArray in CONNECTOR_LINE_CONFIGS.
func TestStaticJS_RenderMapPolyline(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "buildConnectorLayer(") {
		t.Error("map.js: renderMap must call buildConnectorLayer() to connect origin and target markers")
	}
	// dashArray configuration lives in config.js (CONNECTOR_LINE_CONFIGS presets).
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "dashArray") {
		t.Error("config.js: CONNECTOR_LINE_CONFIGS must include dashArray entries for dot/dash rhythm styles")
	}
}

// TestStaticJS_TileLayerConfigs verifies that app.js declares TILE_LAYER_CONFIGS
// with both 'light' and 'dark' variants pointing to the CARTO basemap service.
// Tile URLs must not use the raw OSM URL so theme-aware switching works correctly.
func TestStaticJS_TileLayerConfigs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "TILE_LAYER_CONFIGS") {
		t.Error("config.js: TILE_LAYER_CONFIGS constant not found — tile URLs must be data-driven")
	}
	if !strings.Contains(body, "'light'") {
		t.Error("config.js: TILE_LAYER_CONFIGS must include a 'light' variant")
	}
	if !strings.Contains(body, "'dark'") {
		t.Error("config.js: TILE_LAYER_CONFIGS must include a 'dark' variant")
	}
	// CARTO attribution must be present to satisfy the tile provider's terms.
	if !strings.Contains(body, "carto.com/attributions") {
		t.Error("config.js: CARTO attribution URL must be present in TILE_LAYER_CONFIGS")
	}
	// OSM is now a supported variant inside TILE_LAYER_CONFIGS; its URL is data-driven
	// and must appear inside that config block, not hardcoded in renderMap.
	if !strings.Contains(body, "tile.openstreetmap.org") {
		t.Error("config.js: tile.openstreetmap.org URL must appear in TILE_LAYER_CONFIGS as the osm variant")
	}
}

// TestStaticJS_MapDarkThemes verifies that config.js declares MAP_DARK_THEMES as
// the authoritative set of theme IDs that map to the dark tile variant.
func TestStaticJS_MapDarkThemes(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_DARK_THEMES") {
		t.Error("config.js: MAP_DARK_THEMES constant not found — dark/light tile selection must be data-driven")
	}
	// The known dark themes must be listed.
	for _, id := range []string{"'dark'", "'deep-blue'", "'forest-green'"} {
		cfg := strings.Index(body, "MAP_DARK_THEMES")
		if cfg == -1 {
			break
		}
		// look for the id somewhere after MAP_DARK_THEMES declaration
		if !strings.Contains(body[cfg:cfg+200], id) {
			t.Errorf("config.js: MAP_DARK_THEMES must include theme id %s", id)
		}
	}
}

// TestStaticJS_GetMapTileVariant verifies that app.js exposes a named
// getMapTileVariant() function which is the single decision point for
// mapping the active application theme to a tile-layer variant string.
func TestStaticJS_GetMapTileVariant(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function getMapTileVariant(") {
		t.Error("map.js: getMapTileVariant function not found")
	}
}

// TestStaticJS_RefreshMapTiles verifies that map.js exposes a named
// refreshMapTiles() function that swaps the tile layer on the live map
// with a fade-out/fade-in animation.  It is called only from
// setMapTileVariant() (user-driven tile changes).  Theme-triggered tile swaps
// are handled silently by syncMapTileVariantToTheme().
func TestStaticJS_RefreshMapTiles(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function refreshMapTiles(") {
		t.Error("map.js: refreshMapTiles function not found")
	}
}

// TestStaticJS_ApplyThemeCallsRefreshMapTiles verifies that applyTheme() ensures
// map tiles are refreshed when the colour theme changes.  The function may do
// this directly (refreshMapTiles()) or via syncMapTileVariantToTheme(), which
// itself calls refreshMapTiles() internally.
func TestStaticJS_ApplyThemeCallsRefreshMapTiles(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")

	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	// Find the closing brace of applyTheme by scanning for the next top-level function.
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// applyTheme must trigger a tile refresh either directly or via syncMapTileVariantToTheme.
	if !strings.Contains(fnBody, "refreshMapTiles()") && !strings.Contains(fnBody, "syncMapTileVariantToTheme(") {
		t.Error("theme.js: applyTheme must call refreshMapTiles() or syncMapTileVariantToTheme() so tile layer updates on theme change")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 fix tests — theme fade / input colour / map-bar visibility / tile swap
// ---------------------------------------------------------------------------

// TestStaticJS_ApplyThemeFiltersOpacityEvent verifies that applyTheme() uses
// e.propertyName to guard the transitionend handler so only the body's own
// opacity transition — not background/color transitions or bubbling child
// events — triggers the theme swap.
func TestStaticJS_ApplyThemeFiltersOpacityEvent(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")

	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1200
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "propertyName") {
		t.Error("theme.js: applyTheme transitionend handler must check e.propertyName to filter the correct transition event")
	}
	if !strings.Contains(fnBody, "'opacity'") {
		t.Error("theme.js: applyTheme must guard transitionend with e.propertyName === 'opacity'")
	}
}

// TestStaticJS_MapBarHiddenToggled verifies that renderMap() removes the hidden
// attribute from #geo-map-outer when the map is shown and sets it when hidden,
// so the tile-variant selector bar (inside the outer wrapper) is visible exactly
// when the map is visible.
func TestStaticJS_MapBarHiddenToggled(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 3000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// Both show and hide paths must reference the outer wrapper element and toggle hidden.
	if !strings.Contains(fnBody, "geo-map-outer") {
		t.Error("map.js: renderMap must reference geo-map-outer to toggle its visibility")
	}
	if !strings.Contains(fnBody, "hidden = false") && !strings.Contains(fnBody, "removeAttribute('hidden')") {
		t.Error("map.js: renderMap must reveal #geo-map-outer (hidden = false) when map is shown")
	}
	if !strings.Contains(fnBody, "hidden = true") && !strings.Contains(fnBody, "setAttribute('hidden'") {
		t.Error("map.js: renderMap must hide #geo-map-outer (hidden = true) when map is hidden")
	}
}

// TestStaticJS_RefreshMapTilesRequestAnimationFrame verifies that the updated
// refreshMapTiles() uses requestAnimationFrame to remove the fading class after
// the tile swap, rather than registering a second transitionend listener that
// would never fire (since removing the class triggers the transition, not ends it).
func TestStaticJS_RefreshMapTilesRequestAnimationFrame(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function refreshMapTiles(")
	if fnStart == -1 {
		t.Fatal("map.js: refreshMapTiles function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "requestAnimationFrame") {
		t.Error("map.js: refreshMapTiles must use requestAnimationFrame to remove geo-map--fading after tile swap")
	}
	if !strings.Contains(fnBody, "propertyName") {
		t.Error("map.js: refreshMapTiles transitionend handler must filter by e.propertyName to avoid acting on bubbling child events")
	}
}

// TestStaticJS_SyncMapTileVariantNoFadeAnimation verifies that
// syncMapTileVariantToTheme() does NOT call refreshMapTiles(), ensuring the
// theme-driven tile swap is always silent (no map fade animation).  The fade
// would be redundant because the body is already invisible during a theme
// transition, and the second transitionend listener in the old refreshMapTiles
// would leave geo-map--fading stuck permanently.
func TestStaticJS_SyncMapTileVariantNoFadeAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function syncMapTileVariantToTheme(")
	if fnStart == -1 {
		t.Fatal("map.js: syncMapTileVariantToTheme function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 600
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// Must NOT delegate to animated refreshMapTiles — silent swap only.
	if strings.Contains(fnBody, "refreshMapTiles()") {
		t.Error("map.js: syncMapTileVariantToTheme must NOT call refreshMapTiles() — tile swap must be silent during theme transitions")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 — theme-fade / map-tile-bar tests
// ---------------------------------------------------------------------------

// TestStaticJS_MapThemeToTileVariant verifies that config.js declares
// MAP_THEME_TO_TILE_VARIANT mapping all five supported theme IDs to either
// 'light' or 'dark', providing the default tile variant for each theme.
func TestStaticJS_MapThemeToTileVariant(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_THEME_TO_TILE_VARIANT") {
		t.Fatal("config.js: MAP_THEME_TO_TILE_VARIANT constant not found")
	}
	for _, themeID := range []string{"'default'", "'light-green'", "'deep-blue'", "'forest-green'", "'dark'"} {
		if !strings.Contains(body, themeID) {
			t.Errorf("config.js: MAP_THEME_TO_TILE_VARIANT must include theme %s", themeID)
		}
	}
}

// TestStaticJS_MapTileVariants verifies that MAP_TILE_VARIANTS is declared in
// config.js as an ordered array containing all three supported tile variants.
func TestStaticJS_MapTileVariants(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MAP_TILE_VARIANTS") {
		t.Fatal("config.js: MAP_TILE_VARIANTS constant not found")
	}
	// All three variants must be listed.
	for _, v := range []string{"'light'", "'osm'", "'dark'"} {
		if !strings.Contains(body, v) {
			t.Errorf("config.js: MAP_TILE_VARIANTS must contain variant %s", v)
		}
	}
}

// TestStaticJS_OsmTileInConfigs verifies that the osm tile variant entry in
// TILE_LAYER_CONFIGS points to tile.openstreetmap.org.
func TestStaticJS_OsmTileInConfigs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// 'osm' key must exist as a variant key.
	if !strings.Contains(body, "osm:") && !strings.Contains(body, "'osm'") {
		t.Error("config.js: TILE_LAYER_CONFIGS must declare an osm variant")
	}
	if !strings.Contains(body, "tile.openstreetmap.org") {
		t.Error("config.js: osm variant must use tile.openstreetmap.org URL")
	}
}

// TestStaticJS_SetMapTileVariant verifies that app.js exposes a named
// setMapTileVariant() function which is called from the map bar buttons.
func TestStaticJS_SetMapTileVariant(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function setMapTileVariant(") {
		t.Error("map.js: setMapTileVariant function not found")
	}
}

// TestStaticJS_RenderMapBar verifies that app.js exposes a named renderMapBar()
// function that builds the three tile-variant buttons above the map.
func TestStaticJS_RenderMapBar(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function renderMapBar(") {
		t.Error("map.js: renderMapBar function not found")
	}
}

// TestStaticJS_SyncMapTileVariantToTheme verifies that map.js exposes a named
// syncMapTileVariantToTheme() function which is called via PathProbe.Map by
// theme.js to align the tile variant with the active colour theme.
func TestStaticJS_SyncMapTileVariantToTheme(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function syncMapTileVariantToTheme(") {
		t.Error("map.js: syncMapTileVariantToTheme function not found")
	}
}

// TestStaticJS_ThemeTransitioning verifies that applyTheme() adds the
// 'theme-transitioning' class to body to drive the fade-out/in animation.
func TestStaticJS_ThemeTransitioning(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")

	if !strings.Contains(body, "theme-transitioning") {
		t.Error("theme.js: 'theme-transitioning' class not found — theme fade animation requires it")
	}
	// The class must be both added and removed within applyTheme.
	fnStart := strings.Index(body, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	if !strings.Contains(fnBody, "theme-transitioning") {
		t.Error("theme.js: applyTheme must reference 'theme-transitioning' class")
	}
}

// ---------------------------------------------------------------------------
// Phase 6 fix-2 tests — color-scheme / dot buttons / overlay wrapper
// ---------------------------------------------------------------------------

// TestStaticJS_RenderMapUsesOuterWrapper verifies that renderMap() references
// geo-map-outer to toggle the entire map area (wrapper + bar + map) as one unit.
func TestStaticJS_RenderMapUsesOuterWrapper(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 3000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "geo-map-outer") {
		t.Error("map.js: renderMap must reference geo-map-outer to toggle map area visibility")
	}
}

// TestStaticJS_RenderMapBarNoTextContent verifies that renderMapBar() produces
// buttons without text content — dot-only style, accessible via aria-label.
func TestStaticJS_RenderMapBarNoTextContent(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMapBar(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMapBar function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// Must have aria-label for accessibility.
	if !strings.Contains(fnBody, "aria-label") {
		t.Error("map.js: renderMapBar buttons must include aria-label for accessibility")
	}
	// Must have title for native tooltip.
	if !strings.Contains(fnBody, "title=") {
		t.Error("map.js: renderMapBar buttons should include title attribute for tooltip")
	}
	// The button closing tag must immediately follow the opening tag (no text node).
	// Check that the inner text is NOT rendered (no i18nKey value as text content).
	if strings.Contains(fnBody, ">'"+"\n") || strings.Contains(fnBody, "> +\n      esc(t(") {
		t.Error("map.js: renderMapBar must not render i18n text inside the button element")
	}
}

// Phase 7 fix tests — map z-index isolation / header+footer fade / copyright year
// ---------------------------------------------------------------------------

// TestStaticJS_ApplyThemeUsesMainElement verifies that applyTheme() attaches
// the transitionend listener to the .main element (not document.body), so the
// theme variables are swapped after only the main content has faded out and
// header/footer remain fully visible throughout.
func TestStaticJS_ApplyThemeUsesMainElement(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", rec.Code)
	}
	themeJS := rec.Body.String()

	fnStart := strings.Index(themeJS, "function applyTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: applyTheme function not found")
	}
	nextFn := strings.Index(themeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = themeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(themeJS) {
			end = len(themeJS)
		}
		fnBody = themeJS[fnStart:end]
	}

	if !strings.Contains(fnBody, ".main") && !strings.Contains(fnBody, "querySelector('.main')") {
		t.Error("theme.js: applyTheme must use .main (querySelector('.main')) as the fade target, not body")
	}
	if !strings.Contains(fnBody, "addEventListener('transitionend'") && !strings.Contains(fnBody, `addEventListener("transitionend"`) {
		t.Error("theme.js: applyTheme must attach a transitionend listener to the fade target")
	}
}

// TestStaticJS_CopyrightStartYearConst verifies that app.js declares a
// COPYRIGHT_START_YEAR constant so the copyright year range is driven from a
// single, readable source-of-truth rather than scattered literal values.
func TestStaticJS_CopyrightStartYearConst(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "COPYRIGHT_START_YEAR") {
		t.Error("config.js: COPYRIGHT_START_YEAR constant not found — copyright year logic requires a single source-of-truth")
	}
	// The constant must be assigned a four-digit year value.
	if !strings.Contains(body, "COPYRIGHT_START_YEAR = 2026") {
		t.Error("config.js: COPYRIGHT_START_YEAR must be initialised to 2026")
	}
}

// TestStaticJS_UpdateCopyrightYearFunction verifies that locale.js defines an
// updateCopyrightYear() function that references the footer-copyright i18n key
// and builds an en-dash year range from COPYRIGHT_START_YEAR to the current year.
func TestStaticJS_UpdateCopyrightYearFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function updateCopyrightYear(")
	if fnStart == -1 {
		t.Fatal("locale.js: updateCopyrightYear function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "footer-copyright") {
		t.Error("locale.js: updateCopyrightYear must target [data-i18n='footer-copyright'] elements")
	}
	if !strings.Contains(fnBody, "COPYRIGHT_START_YEAR") {
		t.Error("locale.js: updateCopyrightYear must use COPYRIGHT_START_YEAR constant")
	}
	// En-dash (U+2013) separates the start and end years in the range string.
	if !strings.Contains(fnBody, `\u2013`) && !strings.Contains(fnBody, "–") {
		t.Error("locale.js: updateCopyrightYear must use an en-dash to separate the year range")
	}
}

// TestStaticJS_ApplyLocaleCallsCopyrightYear verifies that applyLocale() calls
// updateCopyrightYear() so the copyright year is refreshed every time the
// locale is applied (including on page load and when the user switches language).
func TestStaticJS_ApplyLocaleCallsCopyrightYear(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("locale.js: applyLocale function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "updateCopyrightYear") {
		t.Error("locale.js: applyLocale must call updateCopyrightYear() to keep the copyright year range current")
	}
}

// Phase 7 (Round 2) tests — spellcheck suppression / map tile bg-color flash fix
// ---------------------------------------------------------------------------

// TestStaticJS_SpellcheckDisabledInDOMContentLoaded verifies that app.js
// centrally disables browser spell-check, autocorrect and autocapitalize on
// all input[type="text"] elements.  Doing this in the initialisation block
// (rather than per-element HTML attributes) ensures every current and future
// text field is covered without per-field opt-out.
func TestStaticJS_SpellcheckDisabledInDOMContentLoaded(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/form.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /form.js: want 200, got %d", rec.Code)
	}
	formJS := rec.Body.String()

	if !strings.Contains(formJS, "spellcheck") {
		t.Error("form.js: must disable spellcheck on text inputs")
	}
	if !strings.Contains(formJS, "spellcheck = false") {
		t.Error("form.js: spellcheck must be set to false (el.spellcheck = false)")
	}
	if !strings.Contains(formJS, "autocorrect") {
		t.Error("form.js: must set autocorrect='off' on text inputs")
	}
	if !strings.Contains(formJS, "autocapitalize") {
		t.Error("form.js: must set autocapitalize='none' on text inputs")
	}
	// Must target input[type="text"] specifically.
	if !strings.Contains(formJS, `input[type="text"]`) {
		t.Error(`form.js: spellcheck suppression must target input[type="text"] elements`)
	}
}

// TestStaticJS_TileLayerConfigsBgColor verifies that every entry in
// TILE_LAYER_CONFIGS declares a bgColor property.  bgColor is the single
// source of truth for the map container background colour; without it the
// white-flash artefact cannot be fixed without hardcoding values elsewhere.
func TestStaticJS_TileLayerConfigsBgColor(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	cfgJS := rec.Body.String()

	cfgStart := strings.Index(cfgJS, "const TILE_LAYER_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("config.js: TILE_LAYER_CONFIGS not found")
	}
	// Extract to the closing brace of the object.
	endIdx := strings.Index(cfgJS[cfgStart:], "\n};")
	var cfgBlock string
	if endIdx != -1 {
		cfgBlock = cfgJS[cfgStart : cfgStart+endIdx+3]
	} else {
		cfgBlock = cfgJS[cfgStart : cfgStart+1500]
	}

	if !strings.Contains(cfgBlock, "bgColor") {
		t.Error("config.js: TILE_LAYER_CONFIGS must include a bgColor property on each entry")
	}
	// All three variants must carry the property.
	for _, variant := range []string{"light", "osm", "dark"} {
		vStart := strings.Index(cfgBlock, variant+":")
		if vStart == -1 {
			t.Errorf("config.js: TILE_LAYER_CONFIGS.%s entry not found", variant)
			continue
		}
		vEnd := strings.Index(cfgBlock[vStart:], "},")
		if vEnd == -1 {
			vEnd = len(cfgBlock) - vStart
		}
		vBlock := cfgBlock[vStart : vStart+vEnd]
		if !strings.Contains(vBlock, "bgColor") {
			t.Errorf("config.js: TILE_LAYER_CONFIGS.%s must have a bgColor property", variant)
		}
	}
}

// TestStaticJS_ApplyMapBgColorFunction verifies that app.js defines an
// applyMapBgColor() function that reads bgColor from TILE_LAYER_CONFIGS and
// applies it to the map container, acting as the single point responsible for
// the background-colour update.
func TestStaticJS_ApplyMapBgColorFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function applyMapBgColor(")
	if fnStart == -1 {
		t.Fatal("map.js: applyMapBgColor function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 400
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "bgColor") {
		t.Error("map.js: applyMapBgColor must read bgColor from TILE_LAYER_CONFIGS")
	}
	if !strings.Contains(fnBody, "background") {
		t.Error("map.js: applyMapBgColor must set container.style.background")
	}
}

// TestStaticJS_RefreshMapTilesCallsApplyMapBgColor verifies that the animated
// tile swap path in refreshMapTiles() calls applyMapBgColor() after the new
// tile layer is added, so the container background is correct before the map
// fades back in.
func TestStaticJS_RefreshMapTilesCallsApplyMapBgColor(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function refreshMapTiles(")
	if fnStart == -1 {
		t.Fatal("map.js: refreshMapTiles function not found")
	}
	// Functions inside the IIFE are indented, so look for the next indented function.
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "applyMapBgColor") {
		t.Error("map.js: refreshMapTiles must call applyMapBgColor() after swapping tiles to prevent white-flash on dark tile load")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 3) — marker icon redesign & temporary style picker
// ---------------------------------------------------------------------------

// TestStaticJS_MarkerStyleConfigsDefined verifies that config.js declares
// MARKER_STYLE_CONFIGS with the diamond-pulse style entry.
func TestStaticJS_MarkerStyleConfigsDefined(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MARKER_STYLE_CONFIGS") {
		t.Fatal("config.js: MARKER_STYLE_CONFIGS constant must be declared")
	}
	if !strings.Contains(body, "'marker-style-diamond-pulse'") {
		t.Error("config.js: MARKER_STYLE_CONFIGS must include i18nKey 'marker-style-diamond-pulse'")
	}
	// Entry must declare a buildHtml property.
	if !strings.Contains(body, "buildHtml") {
		t.Error("config.js: MARKER_STYLE_CONFIGS entries must declare a buildHtml function")
	}
}

// TestStaticJS_MapPointConfigsShortLabel verifies that MAP_POINT_CONFIGS
// carries a shortLabel property on both 'origin' and 'target' entries.
// shortLabel is passed to buildHtml() so the labeled style can render the
// marker letter without hardcoding it inside the style config.
func TestStaticJS_MapPointConfigsShortLabel(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	cfgStart := strings.Index(body, "const MAP_POINT_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("config.js: MAP_POINT_CONFIGS not found")
	}
	endIdx := strings.Index(body[cfgStart:], "};")
	if endIdx == -1 {
		endIdx = 400
	}
	cfgBlock := body[cfgStart : cfgStart+endIdx+2]
	if !strings.Contains(cfgBlock, "shortLabel") {
		t.Error("config.js: MAP_POINT_CONFIGS must include a shortLabel property for the labeled marker style")
	}
}

// TestStaticJS_BuildMarkerIconUsesStyleConfig verifies that buildMarkerIcon()
// reads both MAP_POINT_CONFIGS (for role colour / class) and
// MARKER_STYLE_CONFIGS (for shape / size), combining them into a single
// L.divIcon — clean separation of role vs. visual style.
func TestStaticJS_BuildMarkerIconUsesStyleConfig(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildMarkerIcon(")
	if fnStart == -1 {
		t.Fatal("map.js: buildMarkerIcon function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 600
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "MARKER_STYLE_CONFIGS") {
		t.Error("map.js: buildMarkerIcon must read MARKER_STYLE_CONFIGS for the shape/size data")
	}
	if !strings.Contains(fnBody, "buildHtml") {
		t.Error("map.js: buildMarkerIcon must call styleCfg.buildHtml to produce the inner HTML")
	}
	if !strings.Contains(fnBody, "_markerStyleId") {
		t.Error("map.js: buildMarkerIcon must use _markerStyleId to select the active style config")
	}
}

// TestStaticJS_RefreshMapMarkersFunction verifies that app.js defines
// refreshMapMarkers() which replaces only the Marker layers on the live map
// without destroying the tile layer, polyline, or legend.
func TestStaticJS_RefreshMapMarkersFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function refreshMapMarkers(") {
		t.Fatal("map.js: refreshMapMarkers function not found")
	}
	fnStart := strings.Index(body, "function refreshMapMarkers(")
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}
	// Must iterate layers and remove Marker instances only.
	if !strings.Contains(fnBody, "eachLayer") {
		t.Error("map.js: refreshMapMarkers must use _map.eachLayer to locate existing markers")
	}
	if !strings.Contains(fnBody, "L.Marker") {
		t.Error("map.js: refreshMapMarkers must guard removal with 'instanceof L.Marker'")
	}
	if !strings.Contains(fnBody, "_lastPub") && !strings.Contains(fnBody, "_lastTgt") {
		t.Error("map.js: refreshMapMarkers must use _lastPub/_lastTgt to recreate markers")
	}
}

// TestStaticJS_RenderMapStoresLastGeo verifies that renderMap() stores the
// pub and tgt arguments into _lastPub and _lastTgt so that refreshMapMarkers()
// can recreate markers without requiring a full map rebuild.
func TestStaticJS_RenderMapStoresLastGeo(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 3000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "_lastPub") {
		t.Error("map.js: renderMap must store the pub argument into _lastPub")
	}
	if !strings.Contains(fnBody, "_lastTgt") {
		t.Error("map.js: renderMap must store the tgt argument into _lastTgt")
	}
	if !strings.Contains(fnBody, "applyMarkerColorScheme()") {
		t.Error("map.js: renderMap must call applyMarkerColorScheme() to apply the initial colour scheme")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 3) — diamond marker redesign & theme-adaptive tokens
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Phase 7 (Round 4) — marker colour scheme picker + legend sync
// ---------------------------------------------------------------------------

// TestStaticJS_MarkerColorSchemeConfigsDefined verifies that config.js declares
// MARKER_COLOR_SCHEME_CONFIGS with the ocean colour scheme entry.
func TestStaticJS_MarkerColorSchemeConfigsDefined(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "MARKER_COLOR_SCHEME_CONFIGS") {
		t.Fatal("config.js: MARKER_COLOR_SCHEME_CONFIGS constant must be declared")
	}
	if !strings.Contains(body, "'marker-color-ocean'") {
		t.Error("config.js: MARKER_COLOR_SCHEME_CONFIGS must include i18nKey 'marker-color-ocean'")
	}
	if !strings.Contains(body, "originColor") || !strings.Contains(body, "targetColor") {
		t.Error("config.js: MARKER_COLOR_SCHEME_CONFIGS entries must declare originColor and targetColor fields")
	}
}

// TestStaticJS_MarkerColorSchemeStateVars verifies that app.js declares the
// _markerColorSchemeId and _legendControl module-level state variables.
func TestStaticJS_MarkerColorSchemeStateVars(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "_markerColorSchemeId") {
		t.Error("map.js: _markerColorSchemeId state variable not declared")
	}
	if !strings.Contains(body, "_legendControl") {
		t.Error("map.js: _legendControl state variable not declared")
	}
}

// TestStaticJS_ColorSchemeFunctions verifies that app.js defines
// applyMarkerColorScheme() which applies the active colour scheme.
func TestStaticJS_ColorSchemeFunctions(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	if !strings.Contains(body, "function applyMarkerColorScheme") {
		t.Error("map.js: function applyMarkerColorScheme not found")
	}

	// applyMarkerColorScheme must set --mc-origin and --mc-target on the root element.
	fnStart := strings.Index(body, "function applyMarkerColorScheme")
	if fnStart != -1 {
		nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
		var fnBody string
		if nextFn != -1 {
			fnBody = body[fnStart : fnStart+1+nextFn]
		} else {
			end := fnStart + 800
			if end > len(body) {
				end = len(body)
			}
			fnBody = body[fnStart:end]
		}
		if !strings.Contains(fnBody, "--mc-origin") {
			t.Error("map.js: applyMarkerColorScheme must set the --mc-origin CSS custom property")
		}
		if !strings.Contains(fnBody, "--mc-target") {
			t.Error("map.js: applyMarkerColorScheme must set the --mc-target CSS custom property")
		}
	}
}

// TestStaticJS_BuildMapLegendUsesBuildHtml verifies that buildMapLegend()
// uses MARKER_STYLE_CONFIGS[_markerStyleId].buildHtml(roleCfg) to produce
// the legend icon so it mirrors the active marker style (legend sync fix).
func TestStaticJS_BuildMapLegendUsesBuildHtml(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildMapLegend(")
	if fnStart == -1 {
		t.Fatal("map.js: buildMapLegend function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "MARKER_STYLE_CONFIGS") {
		t.Error("map.js: buildMapLegend must use MARKER_STYLE_CONFIGS to look up the active style")
	}
	if !strings.Contains(fnBody, "buildHtml") {
		t.Error("map.js: buildMapLegend must call buildHtml() to produce the legend icon (legend sync)")
	}
	if !strings.Contains(fnBody, "geo-legend__marker") {
		t.Error("map.js: buildMapLegend must apply the .geo-legend__marker CSS class to the icon wrapper")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 5) — fixed marker appearance + colour scheme + legend i18n
// ---------------------------------------------------------------------------

// TestStaticJS_BuildMapLegendDataI18nAttribute verifies that buildMapLegend()
// adds a data-i18n attribute to the label span in each legend item so that
// applyLocale() can update the text reactively when the user switches language.
// Without this attribute the legend text would be frozen at creation time and
// never reflect a locale change.
func TestStaticJS_BuildMapLegendDataI18nAttribute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function buildMapLegend(")
	if fnStart == -1 {
		t.Fatal("map.js: buildMapLegend function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 1500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "data-i18n") {
		t.Error("map.js: buildMapLegend must add a data-i18n attribute to the legend label span so applyLocale() can update it on language change")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 6) tests — results section i18n re-render on locale switch
// ---------------------------------------------------------------------------

// TestStaticJS_LastReportStateVar verifies that app.js declares a module-level
// _lastReport variable used to cache the most recently rendered diagnostic
// report for re-rendering when the user switches locale.
func TestStaticJS_LastReportStateVar(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/renderer.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /renderer.js: want 200, got %d", rec.Code)
	}
	rendererJS := rec.Body.String()

	if !strings.Contains(rendererJS, "let _lastReport = null") {
		t.Error("renderer.js: module-level variable '_lastReport' not found — required to cache the report for locale-switch re-render")
	}
}

// TestStaticJS_RenderReportStoresLastReport verifies that renderReport() saves
// the report object into _lastReport so applyLocale() can re-render it later.
func TestStaticJS_RenderReportStoresLastReport(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/renderer.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /renderer.js: want 200, got %d", rec.Code)
	}
	rendererJS := rec.Body.String()

	fnStart := strings.Index(rendererJS, "function renderReport(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderReport function not found")
	}
	nextFn := strings.Index(rendererJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = rendererJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 500
		if end > len(rendererJS) {
			end = len(rendererJS)
		}
		fnBody = rendererJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "_lastReport = r") {
		t.Error("renderer.js: renderReport must assign '_lastReport = r' so the report can be replayed when the locale changes")
	}
}

// TestStaticJS_RenderRouteSectionColumns verifies that renderer.js references
// all six i18n column-header keys used in the route-trace hop table.
func TestStaticJS_RenderRouteSectionColumns(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	keys := []string{"th-ttl", "th-ip-host", "th-asn", "th-country", "th-loss", "th-avg-rtt"}
	for _, k := range keys {
		if !strings.Contains(body, "'"+k+"'") {
			t.Errorf("renderer.js: renderRouteSection must reference i18n key %q", k)
		}
	}
}

// TestStaticJS_AppendProgressNoInnerHTML verifies that appendProgress in
// app.js builds its DOM nodes with textContent (not innerHTML) so that
// untrusted progress-event strings cannot inject HTML/JS (XSS protection).
func TestStaticJS_AppendProgressNoInnerHTML(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/app.js")

	// Locate the function body.
	fnStart := strings.Index(body, "function appendProgress(")
	if fnStart == -1 {
		t.Fatal("app.js: appendProgress function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 600
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if strings.Contains(fnBody, "innerHTML") {
		t.Error("app.js: appendProgress must not use innerHTML — use textContent for XSS safety")
	}
	if !strings.Contains(fnBody, "textContent") {
		t.Error("app.js: appendProgress must use textContent to set stage/message text")
	}
}

// TestStaticJS_ApplyLocaleReRendersReport verifies that locale.js / applyLocale()
// triggers results-section re-render via the runtime-resolved
// PathProbe.Renderer.rerenderLast() callback when a cached report is present.
// This keeps all dynamically generated label text in sync with the active
// locale without requiring data-i18n attributes in the generated HTML.
func TestStaticJS_ApplyLocaleReRendersReport(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("locale.js: applyLocale function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	// Must call through the runtime-resolved callback, not directly.
	if !strings.Contains(fnBody, "PathProbe.Renderer") {
		t.Error("locale.js: applyLocale must trigger report re-render via PathProbe.Renderer.rerenderLast()")
	}
	if !strings.Contains(fnBody, "rerenderLast") {
		t.Error("locale.js: applyLocale must call rerenderLast() to re-render the results section on locale change")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 8) tests — locale-aware history timestamps
// ---------------------------------------------------------------------------

// TestStaticJS_LastHistoryItemsStateVar verifies that app.js declares a
// module-level _lastHistoryItems variable used to cache the fetched history
// items for re-rendering when the user switches locale.
func TestStaticJS_LastHistoryItemsStateVar(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "let _lastHistoryItems = null") {
		t.Error("app.js: module-level variable '_lastHistoryItems' not found — required to cache history list for locale-switch re-render")
	}
}

// TestStaticJS_FormatHistoryTimeFunction verifies that app.js defines a
// formatHistoryTime() function and that it reads the active locale from
// PathProbe.Locale.getLocale() so timestamps reflect the active language.
func TestStaticJS_FormatHistoryTimeFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function formatHistoryTime(")
	if fnStart == -1 {
		t.Fatal("app.js: formatHistoryTime function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 400
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	// Must delegate locale lookup to PathProbe.Locale.getLocale() since _locale
	// is now encapsulated inside locale.js.
	if !strings.Contains(fnBody, "getLocale()") {
		t.Error("app.js: formatHistoryTime must obtain the active locale via PathProbe.Locale.getLocale() — _locale is private to locale.js")
	}
	if !strings.Contains(fnBody, "toLocaleString(") {
		t.Error("app.js: formatHistoryTime must call toLocaleString() to format timestamps using the active locale")
	}
}

// TestStaticJS_RenderHistoryListCachesItems verifies that renderHistoryList()
// assigns items to _lastHistoryItems so applyLocale() can re-render the list
// when the user switches language.
func TestStaticJS_RenderHistoryListCachesItems(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js: want 200, got %d", rec.Code)
	}
	appJS := rec.Body.String()

	fnStart := strings.Index(appJS, "function renderHistoryList(")
	if fnStart == -1 {
		t.Fatal("app.js: renderHistoryList function not found")
	}
	nextFn := strings.Index(appJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = appJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 800
		if end > len(appJS) {
			end = len(appJS)
		}
		fnBody = appJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "_lastHistoryItems = items") {
		t.Error("app.js: renderHistoryList must assign '_lastHistoryItems = items' so the list can be replayed on locale change")
	}
	if !strings.Contains(fnBody, "formatHistoryTime(") {
		t.Error("app.js: renderHistoryList must call formatHistoryTime() to produce locale-aware timestamps")
	}
}

// TestStaticJS_ApplyLocaleReRendersHistory verifies that locale.js / applyLocale()
// triggers history-list re-render via the runtime-resolved
// PathProbe.History.rerenderLast() callback so locale-aware timestamps are
// updated immediately when the user switches language.
func TestStaticJS_ApplyLocaleReRendersHistory(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/locale.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /locale.js: want 200, got %d", rec.Code)
	}
	localeJS := rec.Body.String()

	fnStart := strings.Index(localeJS, "function applyLocale(")
	if fnStart == -1 {
		t.Fatal("locale.js: applyLocale function not found")
	}
	nextFn := strings.Index(localeJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = localeJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(localeJS) {
			end = len(localeJS)
		}
		fnBody = localeJS[fnStart:end]
	}

	// Must call through the runtime-resolved callback, not directly.
	if !strings.Contains(fnBody, "PathProbe.History") {
		t.Error("locale.js: applyLocale must trigger history re-render via PathProbe.History.rerenderLast()")
	}
	if !strings.Contains(fnBody, "rerenderLast") {
		t.Error("locale.js: applyLocale must call rerenderLast() to re-render the history list on locale change")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 9) tests — gradient arc connector between origin and target
// ---------------------------------------------------------------------------

// TestStaticJS_ConnectorLineStateVars verifies that app.js declares the
// module-level state variables used by the connector arc system.
func TestStaticJS_ConnectorLineStateVars(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	for _, decl := range []string{"let _connectorStyleId", "let _connectorLayer"} {
		if !strings.Contains(body, decl) {
			t.Errorf("map.js: module-level declaration %q not found — required by the connector arc system", decl)
		}
	}
}

// TestStaticJS_ConnectorLineConfigsDefined verifies that CONNECTOR_LINE_CONFIGS
// contains the single connector style ('tick-xs').
func TestStaticJS_ConnectorLineConfigsDefined(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "CONNECTOR_LINE_CONFIGS") {
		t.Fatal("config.js: CONNECTOR_LINE_CONFIGS not found")
	}
	for _, id := range []string{
		"'tick-xs'",
	} {
		if !strings.Contains(body, id) {
			t.Errorf("config.js: CONNECTOR_LINE_CONFIGS is missing entry %s", id)
		}
	}
}

// TestStaticJS_ConnectorLineFunctions verifies that map-connector.js defines
// the arc-rendering helpers and that app.js retains refreshConnectorLayer.
func TestStaticJS_ConnectorLineFunctions(t *testing.T) {
	h := newStaticHandler(t)
	// map-connector.js must expose the extracted arc-rendering helpers.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	for _, fn := range []string{
		"function lerpHex(",
		"function buildArcLatLngs(",
		"function buildArrowConnectorLayer(",
		"function buildConnectorLayer(",
	} {
		if !strings.Contains(mcJS, fn) {
			t.Errorf("map-connector.js: function %q not found — required by the connector arc feature", fn)
		}
	}

	// map.js contains refreshConnectorLayer (private, injecting _map).
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "function refreshConnectorLayer(") {
		t.Error("map.js: function refreshConnectorLayer not found — required by the connector arc system")
	}
}

// TestStaticJS_RenderMapUsesConnectorLayer verifies that renderMap() calls
// buildConnectorLayer() to draw the gradient arc instead of a plain polyline.
func TestStaticJS_RenderMapUsesConnectorLayer(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 4000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "buildConnectorLayer(") {
		t.Error("map.js: renderMap must call buildConnectorLayer() to draw the gradient arc connector")
	}
	if strings.Contains(fnBody, "color: '#5b8dee'") {
		t.Error("map.js: renderMap must not use the hardcoded '#5b8dee' polyline — use buildConnectorLayer() instead")
	}
}

// TestStaticJS_ConnectorDefaultIsTickXs verifies that the initial connector
// style identifier is set to 'tick-xs'.
func TestStaticJS_ConnectorDefaultIsTickXs(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "_connectorStyleId = 'tick-xs'") {
		t.Error("map.js: _connectorStyleId must default to 'tick-xs'")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 10) tests — 10 line-pattern styles + temporary picker
// ---------------------------------------------------------------------------

// TestStaticJS_BuildArrowConnectorLayerFunction verifies that map-connector.js defines
// buildArrowConnectorLayer() and that it renders directional symbols using
// pixel-distance-based placement (consistent density at every zoom level).
func TestStaticJS_BuildArrowConnectorLayerFunction(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArrowConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArrowConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 6000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	// Delegates to the SVG helper.
	if !strings.Contains(fnBody, "buildArrowSVG(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must call buildArrowSVG() to render arrow icons")
	}
	// Shape is read from config, not a hardcoded Unicode glyph.
	if !strings.Contains(fnBody, "arrowShape") {
		t.Error("map-connector.js: buildArrowConnectorLayer must read styleCfg.arrowShape to select the SVG shape")
	}
	// Pixel-distance-based placement: cumulative distance table + arrowSpacing.
	if !strings.Contains(fnBody, "cum") {
		t.Error("map-connector.js: buildArrowConnectorLayer must build a cumulative pixel-distance table ('cum') for even spacing")
	}
	if !strings.Contains(fnBody, "arrowSpacing") {
		t.Error("map-connector.js: buildArrowConnectorLayer must read styleCfg.arrowSpacing to control symbol density")
	}
	// Rotation from screen-space tangent.
	if !strings.Contains(fnBody, "latLngToLayerPoint(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must call latLngToLayerPoint() to compute the arc tangent angle")
	}
	if !strings.Contains(fnBody, "atan2(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must use Math.atan2() to derive arrow rotation from arc direction")
	}
}

// TestStaticJS_BuildArcLatLngsMercatorSpace verifies that buildArcLatLngs()
// computes the Bézier arc in Web-Mercator (EPSG:3857) space so the rendered
// curve is geometrically smooth on the Leaflet Mercator map.
func TestStaticJS_BuildArcLatLngsMercatorSpace(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArcLatLngs(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArcLatLngs function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	// Must contain Mercator forward projection (toMerc) and inverse (fromMerc).
	if !strings.Contains(fnBody, "toMerc") {
		t.Error("map-connector.js: buildArcLatLngs must define a toMerc helper for forward Web-Mercator projection")
	}
	if !strings.Contains(fnBody, "fromMerc") {
		t.Error("map-connector.js: buildArcLatLngs must define a fromMerc helper for inverse Web-Mercator projection")
	}
	// Earth radius constant must be present for EPSG:3857 math.
	if !strings.Contains(fnBody, "6378137") {
		t.Error("map-connector.js: buildArcLatLngs must use the WGS-84 Earth radius (6378137) for Mercator conversion")
	}
}

// TestStaticJS_BuildConnectorLayerDispatchesByType verifies that
// buildConnectorLayer() delegates to buildArrowConnectorLayer() when the
// style config declares type === 'arrows', following the Open/Closed principle.
func TestStaticJS_BuildConnectorLayerDispatchesByType(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "buildArrowConnectorLayer(") {
		t.Error("map-connector.js: buildConnectorLayer must call buildArrowConnectorLayer() for 'arrows' type styles")
	}
	if !strings.Contains(fnBody, "'arrows'") {
		t.Error("map-connector.js: buildConnectorLayer must check for type === 'arrows' to dispatch correctly")
	}
}

// TestStaticJS_BuildArrowSVGHelper verifies that map-connector.js defines a buildArrowSVG()
// helper that renders all shape variants as inline SVG using a normalised viewBox.
// SVG-based arrows avoid Unicode glyph size/font variance and ensure
// pixel-accurate arrowheads at every zoom level.
func TestStaticJS_BuildArrowSVGHelper(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArrowSVG(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArrowSVG helper function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	// Must produce inline SVG output.
	if !strings.Contains(fnBody, "viewBox") {
		t.Error("map-connector.js: buildArrowSVG must use an SVG viewBox for normalised coordinate rendering")
	}
	// Must handle all defined shape variants via switch.
	for _, shape := range []string{"chevron", "double", "open", "pointer", "fat"} {
		if !strings.Contains(fnBody, "'"+shape+"'") {
			t.Errorf("map-connector.js: buildArrowSVG is missing a case for shape %q", shape)
		}
	}
	// Rotation must be applied via SVG transform (not CSS) for anchor consistency.
	if !strings.Contains(fnBody, "rotate(") {
		t.Error("map-connector.js: buildArrowSVG must apply rotation via SVG transform rotate()")
	}
}

// TestStaticJS_ConnectorArcLayerSinglePassRendering verifies that
// ConnectorArcLayer._redraw() draws the full arc in a single canvas drawing
// pass using setLineDash (for seamless dot/dash patterns) and
// createLinearGradient (for smooth colour gradient).  This replaces the old
// N-polyline + dashOffset approach which produced doubled end-caps and
// float-precision seams at every segment boundary.
func TestStaticJS_ConnectorArcLayerSinglePassRendering(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	// ConnectorArcLayer must be defined as a L.Layer extension.
	if !strings.Contains(mcJS, "ConnectorArcLayer") {
		t.Fatal("map-connector.js: ConnectorArcLayer not found")
	}
	// Find the _redraw method body.
	redrawStart := strings.Index(mcJS, "_redraw: function()")
	if redrawStart == -1 {
		t.Fatal("map-connector.js: ConnectorArcLayer._redraw method not found")
	}
	nextFn := strings.Index(mcJS[redrawStart+1:], "\n    },")
	var redrawBody string
	if nextFn != -1 {
		redrawBody = mcJS[redrawStart : redrawStart+1+nextFn]
	} else {
		end := redrawStart + 3000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		redrawBody = mcJS[redrawStart:end]
	}
	if !strings.Contains(redrawBody, "setLineDash(") {
		t.Error("map-connector.js: ConnectorArcLayer._redraw must call setLineDash() for seamless dot/dash patterns")
	}
	if !strings.Contains(redrawBody, "createLinearGradient(") {
		t.Error("map-connector.js: ConnectorArcLayer._redraw must call createLinearGradient() for smooth colour gradient")
	}
	if !strings.Contains(redrawBody, "ctx.stroke()") {
		t.Error("map-connector.js: ConnectorArcLayer._redraw must call ctx.stroke() to render the arc")
	}
}

// TestStaticJS_BuildConnectorLayerUsesArcLayer verifies that
// buildConnectorLayer() delegates dot/dash arc rendering to ConnectorArcLayer
// (a single-canvas-path Leaflet layer) instead of creating N gradient
// sub-polylines.  The single-pass architecture is the correct fix for
// the visual discontinuities of the old polyline-per-segment approach.
func TestStaticJS_BuildConnectorLayerUsesArcLayer(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "ConnectorArcLayer") {
		t.Error("map-connector.js: buildConnectorLayer must instantiate ConnectorArcLayer for polyline-type styles")
	}
	if !strings.Contains(fnBody, "group.addLayer(") {
		t.Error("map-connector.js: buildConnectorLayer must add ConnectorArcLayer to the LayerGroup via addLayer()")
	}
}

// TestStaticJS_BuildArrowConnectorLayerSpine verifies that
// buildArrowConnectorLayer() supports an optional spine drawn beneath the
// arrow icons when styleCfg.spineWeight > 0.  The spine must use
// ConnectorArcLayer so it benefits from the same single-canvas-pass
// seamless gradient rendering as the dot-family connector styles.
func TestStaticJS_BuildArrowConnectorLayerSpine(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildArrowConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildArrowConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 6000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "spineWeight") {
		t.Error("map-connector.js: buildArrowConnectorLayer must read styleCfg.spineWeight to conditionally draw a spine")
	}
	if !strings.Contains(fnBody, "ConnectorArcLayer") {
		t.Error("map-connector.js: buildArrowConnectorLayer spine must use ConnectorArcLayer for seamless single-pass rendering")
	}
}

// TestStaticJS_HexToRgbaHelper verifies that map-connector.js defines a hexToRgba()
// helper that converts a '#rrggbb' hex colour and an alpha value [0,1] to the
// rgba() CSS format required by ConnectorArcLayer for canvas strokeStyle.
func TestStaticJS_HexToRgbaHelper(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "function hexToRgba(") {
		t.Fatal("map-connector.js: hexToRgba() helper function not found")
	}
	fnStart := strings.Index(mcJS, "function hexToRgba(")
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}
	if !strings.Contains(fnBody, "rgba(") {
		t.Error("map-connector.js: hexToRgba() must produce an rgba() CSS string")
	}
	if !strings.Contains(fnBody, "parseInt(") {
		t.Error("map-connector.js: hexToRgba() must parse hex channel values with parseInt()")
	}
}

// TestStaticJS_ConnectorArcLayerDefined verifies that map-connector.js defines
// ConnectorArcLayer as a L.Layer extension with all required lifecycle methods,
// map event bindings, and canvas placement inside the map container.
func TestStaticJS_ConnectorArcLayerDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "ConnectorArcLayer = L.Layer.extend(") {
		t.Fatal("map-connector.js: ConnectorArcLayer must be defined as a L.Layer extension")
	}
	for _, method := range []string{"initialize: function(", "onAdd: function(", "onRemove: function(", "_redraw: function("} {
		if !strings.Contains(mcJS, method) {
			t.Errorf("map-connector.js: ConnectorArcLayer must define the %q method", method)
		}
	}
	if !strings.Contains(mcJS, "map.on('move zoom zoomend resize'") {
		t.Error("map-connector.js: ConnectorArcLayer.onAdd must bind 'move zoom zoomend resize' map events")
	}
	if !strings.Contains(mcJS, "map.off('move zoom zoomend resize'") {
		t.Error("map-connector.js: ConnectorArcLayer.onRemove must unbind 'move zoom zoomend resize' map events")
	}
	if !strings.Contains(mcJS, "map.getContainer().appendChild(") {
		t.Error("map-connector.js: ConnectorArcLayer.onAdd must append the canvas to map.getContainer()")
	}
}

// TestStaticJS_IsMapLoadedHelper verifies that app.js defines an isMapLoaded()
// shim and that map-connector.js guards buildArrowConnectorLayer with isMapLoaded().
func TestStaticJS_IsMapLoadedHelper(t *testing.T) {
	h := newStaticHandler(t)

	// ── map.js contains the private isMapLoaded() helper ────────────────────
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map.js: want 200, got %d", rec.Code)
	}
	mapJS := rec.Body.String()

	if !strings.Contains(mapJS, "function isMapLoaded()") {
		t.Fatal("map.js: isMapLoaded() helper not found")
	}
	fnStart := strings.Index(mapJS, "function isMapLoaded()")
	fnEnd := strings.Index(mapJS[fnStart+1:], "\nfunction ")
	var fnBody string
	if fnEnd != -1 {
		fnBody = mapJS[fnStart : fnStart+1+fnEnd]
	} else {
		end := fnStart + 200
		if end > len(mapJS) {
			end = len(mapJS)
		}
		fnBody = mapJS[fnStart:end]
	}
	if !strings.Contains(fnBody, "_map._loaded") {
		t.Error("map.js: isMapLoaded() must check _map._loaded")
	}

	// ── map-connector.js: buildArrowConnectorLayer must guard with isMapLoaded() ──
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec2.Code)
	}
	mcJS := rec2.Body.String()

	arrowStart := strings.Index(mcJS, "function buildArrowConnectorLayer(")
	if arrowStart == -1 {
		t.Fatal("map-connector.js: buildArrowConnectorLayer not found")
	}
	arrowEnd := strings.Index(mcJS[arrowStart+1:], "\n  function ")
	var arrowBody string
	if arrowEnd != -1 {
		arrowBody = mcJS[arrowStart : arrowStart+1+arrowEnd]
	} else {
		end := arrowStart + 6000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		arrowBody = mcJS[arrowStart:end]
	}
	if !strings.Contains(arrowBody, "isMapLoaded(") {
		t.Error("map-connector.js: buildArrowConnectorLayer must guard with isMapLoaded() before calling latLngToLayerPoint()")
	}

	// ── map.js: refreshConnectorLayer must guard with isMapLoaded() ──────────
	if !strings.Contains(mapJS, "function refreshConnectorLayer(") {
		t.Fatal("map.js: refreshConnectorLayer not found")
	}
	rlStart := strings.Index(mapJS, "function refreshConnectorLayer(")
	rlEnd := strings.Index(mapJS[rlStart+1:], "\nfunction ")
	var rlBody string
	if rlEnd != -1 {
		rlBody = mapJS[rlStart : rlStart+1+rlEnd]
	} else {
		end := rlStart + 400
		if end > len(mapJS) {
			end = len(mapJS)
		}
		rlBody = mapJS[rlStart:end]
	}
	if !strings.Contains(rlBody, "isMapLoaded()") {
		t.Error("map.js: refreshConnectorLayer must guard with isMapLoaded() to prevent premature map operations")
	}
}

// TestStaticJS_RenderMapSetsViewBeforeConnector verifies that renderMap()
// calls setView / fitBounds before buildConnectorLayer() so the Leaflet map
// is fully initialised when latLngToLayerPoint() is first invoked.
func TestStaticJS_RenderMapSetsViewBeforeConnector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/map.js")

	fnStart := strings.Index(body, "function renderMap(")
	if fnStart == -1 {
		t.Fatal("map.js: renderMap function not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\nfunction ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 4000
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	// Find relative positions: setView/fitBounds must appear before buildConnectorLayer.
	setViewIdx := strings.Index(fnBody, ".setView(")
	fitBoundsIdx := strings.Index(fnBody, ".fitBounds(")
	connectorIdx := strings.Index(fnBody, "buildConnectorLayer(")

	if setViewIdx == -1 && fitBoundsIdx == -1 {
		t.Fatal("map.js: renderMap must call setView() or fitBounds() to initialise the map")
	}
	if connectorIdx == -1 {
		t.Fatal("map.js: renderMap must call buildConnectorLayer()")
	}

	// At least one viewport-setting call must precede buildConnectorLayer.
	viewportIdx := setViewIdx
	if fitBoundsIdx != -1 && (viewportIdx == -1 || fitBoundsIdx < viewportIdx) {
		viewportIdx = fitBoundsIdx
	}
	if viewportIdx >= connectorIdx {
		t.Error("map.js: renderMap must call setView/fitBounds BEFORE buildConnectorLayer() " +
			"so the Leaflet map is loaded before latLngToLayerPoint() is invoked")
	}
}

// ---------------------------------------------------------------------------
// Phase 7 (Round 11) tests — meteor glow animation on connector arc
// ---------------------------------------------------------------------------

// TestStaticJS_ConnectorGlowConfigsDefined verifies that map-connector.js references
// CONNECTOR_GLOW_CONFIGS and uses all required timing and visual parameters
// for the meteor animation.
func TestStaticJS_ConnectorGlowConfigsDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "CONNECTOR_GLOW_CONFIGS") {
		t.Fatal("map-connector.js: CONNECTOR_GLOW_CONFIGS must be referenced")
	}
	// 'default' preset must define all required animation parameters.
	for _, param := range []string{"travelMs", "pauseMs", "glowRadius", "glowOpacity", "tailLength"} {
		if !strings.Contains(mcJS, param) {
			t.Errorf("map-connector.js: CONNECTOR_GLOW_CONFIGS must include parameter %q", param)
		}
	}
}

// TestStaticJS_ConnectorTickXsGlowEnabled verifies that the 'tick-xs' connector
// style config opts into the meteor animation via glowEnabled: true and
// references the 'default' glow preset via glowConfig.
func TestStaticJS_ConnectorTickXsGlowEnabled(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	cfgJS := rec.Body.String()

	if !strings.Contains(cfgJS, "glowEnabled: true") {
		t.Error("config.js: CONNECTOR_LINE_CONFIGS 'tick-xs' must set glowEnabled: true to enable the meteor animation")
	}
	if !strings.Contains(cfgJS, "glowConfig: 'default'") {
		t.Error("config.js: CONNECTOR_LINE_CONFIGS 'tick-xs' must set glowConfig: 'default' to reference the glow preset")
	}
}

// TestStaticJS_ConnectorGlowLayerDefined verifies that map-connector.js defines
// ConnectorGlowLayer as a L.Layer extension with the required lifecycle
// methods and animation helpers for the meteor light-pulse effect.
func TestStaticJS_ConnectorGlowLayerDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	if !strings.Contains(mcJS, "ConnectorGlowLayer = L.Layer.extend(") {
		t.Fatal("map-connector.js: ConnectorGlowLayer must be defined as a L.Layer extension")
	}
	for _, method := range []string{
		"initialize: function(",
		"onAdd: function(",
		"onRemove: function(",
		"_tick: function(",
		"_drawGlow: function(",
		"_posAtPx: function(",
		"_getScreenPts: function(",
	} {
		if !strings.Contains(mcJS, method) {
			t.Errorf("map-connector.js: ConnectorGlowLayer must define the %q method", method)
		}
	}
}

// TestStaticJS_ConnectorGlowLayerAnimation verifies that ConnectorGlowLayer
// uses requestAnimationFrame for the animation loop, cancels it in onRemove,
// and binds map move/zoom events to invalidate the cached screen projection.
func TestStaticJS_ConnectorGlowLayerAnimation(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	// Find the body of ConnectorGlowLayer.
	start := strings.Index(mcJS, "ConnectorGlowLayer = L.Layer.extend(")
	if start == -1 {
		t.Fatal("map-connector.js: ConnectorGlowLayer not found")
	}
	// Locate the end of the const declaration (next IIFE-level function).
	rest := mcJS[start:]
	endIdx := strings.Index(rest, "\n  function ")
	var layerBody string
	if endIdx != -1 {
		layerBody = rest[:endIdx]
	} else {
		end := start + 15000
		if end > len(mcJS) {
			end = len(mcJS)
		}
		layerBody = mcJS[start:end]
	}

	if !strings.Contains(layerBody, "requestAnimationFrame(") {
		t.Error("map-connector.js: ConnectorGlowLayer must use requestAnimationFrame for the animation loop")
	}
	if !strings.Contains(layerBody, "cancelAnimationFrame(") {
		t.Error("map-connector.js: ConnectorGlowLayer.onRemove must call cancelAnimationFrame to stop the loop")
	}
	if !strings.Contains(layerBody, "clearRect(") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must call clearRect to erase the previous frame")
	}
	if !strings.Contains(layerBody, "createRadialGradient(") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must use createRadialGradient for the glow halo")
	}
	if !strings.Contains(layerBody, "lerpHex(") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must call lerpHex to interpolate head colour along the arc")
	}
}

// TestStaticJS_BuildConnectorLayerAddsGlowLayer verifies that
// buildConnectorLayer() instantiates ConnectorGlowLayer when the style config
// carries glowEnabled === true, adding the meteor animation on top of the base
// arc without coupling the two layers.
func TestStaticJS_BuildConnectorLayerAddsGlowLayer(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	fnStart := strings.Index(mcJS, "function buildConnectorLayer(")
	if fnStart == -1 {
		t.Fatal("map-connector.js: buildConnectorLayer function not found")
	}
	nextFn := strings.Index(mcJS[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = mcJS[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 2500
		if end > len(mcJS) {
			end = len(mcJS)
		}
		fnBody = mcJS[fnStart:end]
	}

	if !strings.Contains(fnBody, "glowEnabled") {
		t.Error("map-connector.js: buildConnectorLayer must check styleCfg.glowEnabled to decide whether to add the glow layer")
	}
	if !strings.Contains(fnBody, "ConnectorGlowLayer") {
		t.Error("map-connector.js: buildConnectorLayer must instantiate ConnectorGlowLayer when glowEnabled is true")
	}
	if !strings.Contains(fnBody, "CONNECTOR_GLOW_CONFIGS") {
		t.Error("map-connector.js: buildConnectorLayer must look up the glow config from CONNECTOR_GLOW_CONFIGS")
	}
	if !strings.Contains(fnBody, "group.addLayer(") {
		t.Error("map-connector.js: buildConnectorLayer must add ConnectorGlowLayer to the LayerGroup via addLayer()")
	}
}

// TestStaticJS_ConnectorGlowConfigsFadeMs verifies that CONNECTOR_GLOW_CONFIGS
// declares a fadeMs parameter in the 'default' preset.  fadeMs defines the
// duration of the extinguish phase after the head reaches the destination and
// is the structural requirement for the three-phase animation cycle.
func TestStaticJS_ConnectorGlowConfigsFadeMs(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /config.js: want 200, got %d", rec.Code)
	}
	cfgJS := rec.Body.String()

	if !strings.Contains(cfgJS, "fadeMs") {
		t.Fatal("config.js: CONNECTOR_GLOW_CONFIGS must declare a fadeMs parameter for the extinguish phase")
	}
	// fadeMs must appear inside the CONNECTOR_GLOW_CONFIGS block.
	cfgStart := strings.Index(cfgJS, "const CONNECTOR_GLOW_CONFIGS")
	if cfgStart == -1 {
		t.Fatal("config.js: CONNECTOR_GLOW_CONFIGS not found")
	}
	cfgEnd := strings.Index(cfgJS[cfgStart:], "};")
	if cfgEnd == -1 {
		cfgEnd = 300
	}
	cfgBlock := cfgJS[cfgStart : cfgStart+cfgEnd+2]
	if !strings.Contains(cfgBlock, "fadeMs") {
		t.Error("config.js: fadeMs must be declared inside CONNECTOR_GLOW_CONFIGS (not elsewhere)")
	}
}

// TestStaticJS_ConnectorGlowLayerExtinguish verifies that ConnectorGlowLayer
// implements the three-phase extinguish animation:
//   - Phase 1 (travel): masterAlpha = 1, progress ramps 0→1.
//   - Phase 2 (fade-out): progress fixed at 1, masterAlpha ramps 1→0, tail
//     converges back into the head (tailPx ∝ masterAlpha).
//   - Phase 3 (dark): canvas cleared, no drawing until next cycle.
//
// These invariants are verified by checking for the structural keywords that
// the three-phase _tick() and masterAlpha-aware _drawGlow() must contain.
func TestStaticJS_ConnectorGlowLayerExtinguish(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/map-connector.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /map-connector.js: want 200, got %d", rec.Code)
	}
	mcJS := rec.Body.String()

	layerStart := strings.Index(mcJS, "ConnectorGlowLayer = L.Layer.extend(")
	if layerStart == -1 {
		t.Fatal("map-connector.js: ConnectorGlowLayer not found")
	}
	// Capture enough of the layer body to cover all methods (~8 KB).
	end := layerStart + 15000
	if end > len(mcJS) {
		end = len(mcJS)
	}
	layerBody := mcJS[layerStart:end]

	// _tick must compute a three-phase cycle: travelMs + fadeMs + pauseMs.
	if !strings.Contains(layerBody, "fadeMs") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must read cfg.fadeMs to compute the three-phase cycle duration")
	}
	if !strings.Contains(layerBody, "masterAlpha") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must declare masterAlpha as a phase-dependent brightness multiplier")
	}
	// _drawGlow must accept and apply masterAlpha.
	if !strings.Contains(layerBody, "_drawGlow: function(ctx, progress, masterAlpha)") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must accept masterAlpha as its third parameter")
	}
	// Tail convergence: tailPx must be proportional to masterAlpha.
	if !strings.Contains(layerBody, "tailPx") || !strings.Contains(layerBody, "* masterAlpha") {
		t.Error("map-connector.js: ConnectorGlowLayer._drawGlow must multiply tailPx by masterAlpha to converge the tail on fade-out")
	}
	// Phase 3: the dark phase must return early without calling _drawGlow.
	if !strings.Contains(layerBody, "return;") {
		t.Error("map-connector.js: ConnectorGlowLayer._tick must return early in phase 3 (dark) without drawing")
	}
}

// ── config.js tests ────────────────────────────────────────────────────────────────────

// TestStaticJS_ConfigNamespace verifies that config.js exports PathProbe.Config
// and exposes all expected public constant keys.
func TestStaticJS_ConfigNamespace(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// Namespace assignment must be present.
	if !strings.Contains(body, "PathProbe.Config") {
		t.Error("config.js: must export PathProbe.Config namespace")
	}

	// All public constant keys must appear in the file.
	for _, key := range []string{
		"MAP_POINT_CONFIGS",
		"MARKER_COLOR_SCHEME_CONFIGS",
		"MARKER_STYLE_CONFIGS",
		"CONNECTOR_LINE_CONFIGS",
		"CONNECTOR_GLOW_CONFIGS",
		"MAP_TILE_VARIANTS",
		"MAP_THEME_TO_TILE_VARIANT",
		"MAP_DARK_THEMES",
		"TILE_LAYER_CONFIGS",
		"TARGET_PORTS",
		"TARGET_MODE_PANELS",
		"WEB_MODES_WITH_PORTS",
		"TARGET_PLACEHOLDER_KEYS",
		"COPYRIGHT_START_YEAR",
		"THEMES",
		"DEFAULT_THEME",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("config.js: missing constant key %q", key)
		}
	}

	// The constant declarations must be present.
	for _, decl := range []string{
		"const DEFAULT_THEME",
		"const THEMES",
		"const COPYRIGHT_START_YEAR",
	} {
		if !strings.Contains(body, decl) {
			t.Errorf("config.js: expected constant declaration %q", decl)
		}
	}
}

// TestStaticJS_ConfigNoFunctions verifies that config.js is a pure data layer
// and contains no function declarations (only arrow-function values in data
// properties such as MARKER_STYLE_CONFIGS.buildHtml are permitted).
func TestStaticJS_ConfigNoFunctions(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// The keyword 'function ' (with trailing space) identifies named function
	// declarations or function expressions.  Arrow functions (=>) do not match
	// and are permitted as inline data-property values (e.g. buildHtml).
	if strings.Contains(body, "function ") {
		t.Error("config.js: must not contain function declarations — pure data layer only")
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

// TestStaticJS_ConfigScopeIsolation verifies that config.js wraps all its
// const declarations inside an arrow IIFE (Immediately-Invoked Function
// Expression) to prevent the "redeclaration of const" SyntaxError that
// browsers report when multiple classic <script> elements declare const
// variables with the same name in the shared global script scope.
//
// Background: in a browser, all classic (non-module) <script> tags share one
// "script scope" for const/let.  config.js declares e.g.
//
//	const CONNECTOR_LINE_CONFIGS = {...}
//
// and app.js destructures:
//
//	const { CONNECTOR_LINE_CONFIGS, ... } = window.PathProbe.Config
//
// Both declarations use the same identifier and therefore collide.  The
// SyntaxError is a parse-time failure: the entire app.js script is rejected
// before a single function is defined, which is why setTheme / setLocale /
// runDiag are all unreachable from inline onclick attributes.
//
// The arrow-IIFE wrapper `(() => { ... })()` moves config.js constants into a
// function scope, preventing them from appearing in the shared script scope.
func TestStaticJS_ConfigScopeIsolation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// config.js must open an arrow IIFE so const declarations are not in the
	// shared script scope.  Both compact and spaced variants are accepted.
	hasArrowIIFE := strings.Contains(body, "(() => {") || strings.Contains(body, "(()=>{")
	if !hasArrowIIFE {
		t.Error("config.js: all constants must be wrapped in an arrow IIFE (() => { ... })() " +
			"to prevent 'redeclaration of const' SyntaxErrors when the browser " +
			"loads both config.js and app.js in the same script scope")
	}

	// The IIFE must be properly closed so the code executes immediately.
	if !strings.Contains(body, "})()") {
		t.Error("config.js: the arrow IIFE must be closed with })() or })(); " +
			"without the closing invocation the constants are never assigned to " +
			"PathProbe.Config and window.PathProbe")
	}
}

// ── Sub-task 3.2: locale.js tests ─────────────────────────────────────────

// TestStaticJS_SetLocaleGlobal verifies that locale.js exposes setLocale as a
// global (window.setLocale = setLocale) so that inline onclick attributes in
// index.html (e.g. onclick="setLocale('en')") can call it without requiring
// app.js to re-declare the function.
func TestStaticJS_SetLocaleGlobal(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/locale.js")

	// The explicit global assignment must be present so browsers can call
	// setLocale() from inline onclick attributes on language buttons.
	if !strings.Contains(body, "window.setLocale = setLocale") {
		t.Error("locale.js: must contain 'window.setLocale = setLocale' to expose " +
			"setLocale as a global callable from inline onclick attributes")
	}
}

// TestStaticJS_LocaleUsesConfigCopyrightYear verifies that locale.js reads the
// copyright start year from PathProbe.Config.COPYRIGHT_START_YEAR at runtime
// rather than hard-coding a numeric year literal.  Hard-coding the year would
// violate the single-source-of-truth principle: the year is already declared
// in config.js and must not be duplicated.
func TestStaticJS_LocaleUsesConfigCopyrightYear(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/locale.js")

	// locale.js must read the year from the config namespace, not define it.
	if !strings.Contains(body, "COPYRIGHT_START_YEAR") {
		t.Error("locale.js: must read COPYRIGHT_START_YEAR from PathProbe.Config, " +
			"not hard-code a numeric year value")
	}

	// There must be no standalone four-digit year literal in the file.
	// The regex matches a bare year number that is not part of a larger
	// identifier (e.g. "2026" as a standalone token).
	import_re := `\b20\d{2}\b`
	matched, _ := regexp.MatchString(import_re, body)
	if matched {
		t.Error("locale.js: must not contain a hard-coded year literal — " +
			"read COPYRIGHT_START_YEAR from PathProbe.Config instead")
	}
}

// TestStaticJS_LocaleRuntimeResolvedCrossModuleCalls verifies that locale.js
// triggers re-render of the results section and history list through
// runtime-resolved cross-module calls (PathProbe.Renderer.rerenderLast and
// PathProbe.History.rerenderLast) rather than calling renderReport() or
// renderHistoryList() directly.
//
// Direct calls would create a hard load-order dependency on app.js (or future
// renderer.js / history.js modules), making locale.js impossible to test in
// isolation and breaking the low-coupling principle.  Guard expressions
// (PathProbe.Renderer && …) ensure the calls degrade gracefully when the
// target module is not yet registered.
func TestStaticJS_LocaleRuntimeResolvedCrossModuleCalls(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/locale.js")

	// Guard for the Renderer module must be present.
	if !strings.Contains(body, "PathProbe.Renderer &&") {
		t.Error("locale.js: applyLocale() must guard PathProbe.Renderer with " +
			"'PathProbe.Renderer &&' before calling rerenderLast() so the call " +
			"degrades gracefully when renderer.js has not yet loaded")
	}

	// Guard for the History module must be present.
	if !strings.Contains(body, "PathProbe.History &&") {
		t.Error("locale.js: applyLocale() must guard PathProbe.History with " +
			"'PathProbe.History &&' before calling rerenderLast() so the call " +
			"degrades gracefully when history.js has not yet loaded")
	}

	// The re-render must go through rerenderLast(), not call renderReport()
	// or renderHistoryList() directly (which would create a hard dependency).
	if strings.Contains(body, "renderReport(") || strings.Contains(body, "renderHistoryList(") {
		t.Error("locale.js: must NOT call renderReport() or renderHistoryList() directly — " +
			"use PathProbe.Renderer.rerenderLast() and PathProbe.History.rerenderLast() " +
			"for runtime-resolved cross-module calls")
	}
}

// ---------------------------------------------------------------------------
// Subtask 3.3 — theme.js module registration tests
// ---------------------------------------------------------------------------

// TestStaticJS_SetThemeGlobal verifies that theme.js exposes setTheme as a
// window-level global so that HTML onclick="setTheme(...)" attributes work
// without requiring callers to reference the PathProbe namespace directly.
func TestStaticJS_SetThemeGlobal(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/theme.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /theme.js: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "window.setTheme = setTheme") {
		t.Error("theme.js: must assign window.setTheme = setTheme for HTML onclick compatibility")
	}
}

// TestStaticJS_ThemeJSRuntimeResolvedMapSync verifies that theme.js guards
// the syncMapTileVariantToTheme call with a runtime check for PathProbe.Map
// so theme.js has no hard load-order dependency on the map module.
func TestStaticJS_ThemeJSRuntimeResolvedMapSync(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")
	if !strings.Contains(body, "PathProbe.Map") {
		t.Error("theme.js: syncMapTileVariantToTheme call must be guarded by a PathProbe.Map runtime check")
	}
	if !strings.Contains(body, "syncMapTileVariantToTheme") {
		t.Error("theme.js: must call syncMapTileVariantToTheme to keep map tiles in sync with the active theme")
	}
}

// TestStaticJS_InitThemeReadsDataDefaultTheme verifies that initTheme() reads
// the server-declared fallback from dataset.defaultTheme rather than repeating
// a hard-coded string, so a server-side theme preference takes effect without
// modifying any JavaScript source.
func TestStaticJS_InitThemeReadsDataDefaultTheme(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/theme.js")
	fnStart := strings.Index(body, "function initTheme(")
	if fnStart == -1 {
		t.Fatal("theme.js: initTheme function not found")
	}
	end := fnStart + 600
	if end > len(body) {
		end = len(body)
	}
	if !strings.Contains(body[fnStart:end], "dataset.defaultTheme") {
		t.Error("theme.js: initTheme() must read document.documentElement.dataset.defaultTheme as the server-declared default")
	}
}

// TestStaticJS_FormPublicAPIComplete verifies that form.js exports every symbol
// that other modules and tests rely on through PathProbe.Form: val, checked,
// getModeFor, getRunningHTML, onTargetChange, and init.
func TestStaticJS_FormPublicAPIComplete(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	for _, sym := range []string{"val", "checked", "getModeFor", "getRunningHTML", "onTargetChange", "init"} {
		// Each symbol must appear inside the PathProbe.Form = { … } export block.
		needle := "PathProbe.Form = {"
		exportStart := strings.Index(body, needle)
		if exportStart == -1 {
			t.Fatalf("form.js: PathProbe.Form = { ... } export block not found")
		}
		exportEnd := strings.Index(body[exportStart:], "};")
		if exportEnd == -1 {
			t.Fatalf("form.js: closing }; not found after PathProbe.Form export block")
		}
		exportBlock := body[exportStart : exportStart+exportEnd+2]
		if !strings.Contains(exportBlock, sym) {
			t.Errorf("form.js: PathProbe.Form must export %q", sym)
		}
	}
}

// TestStaticJS_BuildRequestFunction verifies that api-builder.js defines
// buildRequest() and assembles a payload with the expected { target, options }
// top-level structure, and that it is the sole function exported via the public API.
func TestStaticJS_BuildRequestFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	// buildRequest must be defined as the main entry point.
	if !strings.Contains(body, "function buildRequest(") {
		t.Error("api-builder.js: buildRequest() function must be defined")
	}
	// The returned payload must contain both top-level fields.
	if !strings.Contains(body, "{ target, options: opts }") {
		t.Error("api-builder.js: buildRequest must return { target, options: opts } payload")
	}
	// Only buildRequest is exported — internal helpers remain private to the IIFE.
	needle := "PathProbe.ApiBuilder = {"
	exportStart := strings.Index(body, needle)
	if exportStart == -1 {
		t.Fatalf("api-builder.js: PathProbe.ApiBuilder = { ... } export block not found")
	}
	exportEnd := strings.Index(body[exportStart:], "};")
	if exportEnd == -1 {
		t.Fatalf("api-builder.js: closing }; not found after PathProbe.ApiBuilder export block")
	}
	exportBlock := body[exportStart : exportStart+exportEnd+2]
	if !strings.Contains(exportBlock, "buildRequest") {
		t.Error("api-builder.js: PathProbe.ApiBuilder must export buildRequest")
	}
	// Internal helpers must NOT be exported.
	for _, priv := range []string{"buildWebOpts", "buildSMTPOpts", "buildFTPOpts", "buildSFTPOpts", "parseTimeoutSec"} {
		if strings.Contains(exportBlock, priv) {
			t.Errorf("api-builder.js: %s must remain private (not exported in PathProbe.ApiBuilder)", priv)
		}
	}
}
