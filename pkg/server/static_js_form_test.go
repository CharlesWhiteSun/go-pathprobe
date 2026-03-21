package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

// TestStaticJS_FormOnTargetChangeUsesUnifiedPlaceholder 驗證 form.js 的
// onTargetChange() 使用統一的 ph-host i18n 鍵設定 #host 佔位提示文字，
// 不再依賴 TARGET_PLACEHOLDER_KEYS 逐模式查詢，確保所有偵測模式的
// 主機輸入框皆顯示相同的範例提示，維持低耦合設計。
func TestStaticJS_FormOnTargetChangeUsesUnifiedPlaceholder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// 必須直接使用 'ph-host' 統一鍵設定 #host placeholder。
	if !strings.Contains(body, "'ph-host'") {
		t.Error("form.js: onTargetChange must use 'ph-host' key for #host placeholder — not a per-mode lookup")
	}
	// 不得再宣告或呼叫 _targetPlaceholderKeys 輔助函式。
	if strings.Contains(body, "_targetPlaceholderKeys") {
		t.Error("form.js: _targetPlaceholderKeys helper must be removed — use 'ph-host' directly")
	}
	// 不得再參照 TARGET_PLACEHOLDER_KEYS。
	if strings.Contains(body, "TARGET_PLACEHOLDER_KEYS") {
		t.Error("form.js: form.js must NOT reference TARGET_PLACEHOLDER_KEYS — use 'ph-host' directly")
	}
	// 不得再使用 ph-host-default 的 fallback 邏輯。
	if strings.Contains(body, "ph-host-default") {
		t.Error("form.js: ph-host-default must be removed — obsolete fallback key no longer exists")
	}
}

// TestStaticJS_EnterKeyTriggersRunDiag verifies that form.js wires up a
// document-level keydown listener via initEnterKey() so pressing Enter in any
// text or number input submits the diagnostic without clicking the run button.
//
// The function must:
//   - be a named, isolated function (initEnterKey) for readability and testability
//   - guard against mid-IME composition (e.isComposing) to avoid double-fire with CJK
//   - guard against double-submit when the run button is already disabled
//   - use document-level event delegation so future input fields are covered automatically
//   - call window.runDiag to trigger the diagnostic flow
//   - be called from init() so it activates on page load
func TestStaticJS_EnterKeyTriggersRunDiag(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	// initEnterKey must be a named function for isolated readability and testing.
	if !strings.Contains(body, "initEnterKey") {
		t.Fatal("form.js: initEnterKey function must be defined to wire Enter-key submit")
	}
	// Must listen for the 'Enter' key specifically.
	if !strings.Contains(body, "'Enter'") && !strings.Contains(body, `"Enter"`) {
		t.Error("form.js: initEnterKey must check for e.key === 'Enter'")
	}
	// Must guard against mid-IME input (CJK, etc.) to avoid double-fire.
	if !strings.Contains(body, "isComposing") {
		t.Error("form.js: initEnterKey must check e.isComposing to skip input mid-IME composition")
	}
	// Must guard against submitting while a diagnostic is already running.
	if !strings.Contains(body, "btn.disabled") {
		t.Error("form.js: initEnterKey must check run-btn.disabled to prevent double-submit")
	}
	// Must use event delegation on document (not per-input binding) so future
	// input fields are automatically covered without additional wiring.
	if !strings.Contains(body, "document.addEventListener") {
		t.Error("form.js: initEnterKey must use event delegation on document, not per-element binding")
	}
	// Must forward to window.runDiag to trigger the diagnostic flow.
	if !strings.Contains(body, "runDiag") {
		t.Error("form.js: initEnterKey must invoke runDiag to submit the diagnostic")
	}
	// init() must call initEnterKey() to activate it on page load.
	initIdx := strings.Index(body, "function init(")
	if initIdx == -1 {
		t.Fatal("form.js: init() function not found")
	}
	if !strings.Contains(body[initIdx:], "initEnterKey()") {
		t.Error("form.js: init() must call initEnterKey() to activate Enter-key submit on page load")
	}
}
