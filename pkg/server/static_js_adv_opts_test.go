package server_test

import (
	"strings"
	"testing"
)

// ── Advanced Options visibility feature ───────────────────────────────────────
//
// These tests verify that:
//   - config.js declares ADV_OPT_SUPPORT with entries for every target/mode.
//   - form.js defines updateAdvancedOpts(), wires it on target-change and
//     mode-change, and exports it via PathProbe.Form.
//   - index.html carries the wrapper element IDs required to show/hide each
//     option at runtime.
//
// None of these tests require a browser or DOM; they scan the served static-
// asset text to confirm the structural contracts are in place.

// TestStaticJS_AdvOpt_ConfigHasMatrix verifies that config.js declares the
// ADV_OPT_SUPPORT constant and exports it through PathProbe.Config.
func TestStaticJS_AdvOpt_ConfigHasMatrix(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	if !strings.Contains(body, "ADV_OPT_SUPPORT") {
		t.Error("config.js: ADV_OPT_SUPPORT constant must be declared")
	}
	// Must be exported so form.js can read it via PathProbe.Config.
	if !strings.Contains(body, "ADV_OPT_SUPPORT,") {
		t.Error("config.js: ADV_OPT_SUPPORT must be included in the PathProbe.Config export object")
	}
}

// TestStaticJS_AdvOpt_WebModeEntries verifies that ADV_OPT_SUPPORT covers all
// five web sub-modes and references the expected option keys.
func TestStaticJS_AdvOpt_WebModeEntries(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	for _, mode := range []string{"'public-ip'", "'dns'", "'http'", "'port'", "'traceroute'"} {
		if !strings.Contains(body, mode) {
			t.Errorf("config.js: ADV_OPT_SUPPORT.web must contain an entry for mode %s", mode)
		}
	}
	// 'mtr-count' must appear — used by port and traceroute modes.
	if !strings.Contains(body, "'mtr-count'") {
		t.Error("config.js: ADV_OPT_SUPPORT must reference 'mtr-count'")
	}
	// 'insecure' must appear — used by http and smtp/ftp modes.
	if !strings.Contains(body, "'insecure'") {
		t.Error("config.js: ADV_OPT_SUPPORT must reference 'insecure'")
	}
	// 'geo' must appear — several modes still support geo annotation.
	if !strings.Contains(body, "'geo'") {
		t.Error("config.js: ADV_OPT_SUPPORT must reference 'geo'")
	}
}

// TestStaticJS_AdvOpt_WebHttpNoGeo verifies that the web/http mode entry in
// ADV_OPT_SUPPORT does NOT include 'geo'.  HTTP probe results carry no
// server-IP geo data so the option is inapplicable and must be hidden.
func TestStaticJS_AdvOpt_WebHttpNoGeo(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	// Locate the ADV_OPT_SUPPORT declaration block.
	blockStart := strings.Index(body, "const ADV_OPT_SUPPORT")
	if blockStart == -1 {
		t.Fatal("config.js: ADV_OPT_SUPPORT not found")
	}
	// Locate the end of the web block (first closing '}, ' at the outer level
	// after the web: opening). A simpler proxy: find the 'smtp:' key — it always
	// follows the web block in the declared order.
	smtpIdx := strings.Index(body[blockStart:], "smtp:")
	if smtpIdx == -1 {
		t.Fatal("config.js: ADV_OPT_SUPPORT smtp: section not found")
	}
	webBlock := body[blockStart : blockStart+smtpIdx]

	// Within the web block, find the 'http': line.
	httpLineIdx := strings.Index(webBlock, "'http':")
	if httpLineIdx == -1 {
		t.Fatal("config.js: ADV_OPT_SUPPORT web block has no 'http' entry")
	}
	// Read just that line (up to the next newline).
	newline := strings.Index(webBlock[httpLineIdx:], "\n")
	var httpLine string
	if newline != -1 {
		httpLine = webBlock[httpLineIdx : httpLineIdx+newline]
	} else {
		httpLine = webBlock[httpLineIdx:]
	}
	if strings.Contains(httpLine, "'geo'") {
		t.Errorf("config.js: web/http entry must NOT include 'geo'; got: %s", strings.TrimSpace(httpLine))
	}
}

// TestStaticJS_AdvOpt_WebPortNoGeo verifies that the web/port mode entry in
// ADV_OPT_SUPPORT does NOT include 'geo'.  Port-scan results do not include
// geo annotation so the option is inapplicable and must be hidden.
func TestStaticJS_AdvOpt_WebPortNoGeo(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	blockStart := strings.Index(body, "const ADV_OPT_SUPPORT")
	if blockStart == -1 {
		t.Fatal("config.js: ADV_OPT_SUPPORT not found")
	}
	smtpIdx := strings.Index(body[blockStart:], "smtp:")
	if smtpIdx == -1 {
		t.Fatal("config.js: ADV_OPT_SUPPORT smtp: section not found")
	}
	webBlock := body[blockStart : blockStart+smtpIdx]

	portLineIdx := strings.Index(webBlock, "'port':")
	if portLineIdx == -1 {
		t.Fatal("config.js: ADV_OPT_SUPPORT web block has no 'port' entry")
	}
	newline := strings.Index(webBlock[portLineIdx:], "\n")
	var portLine string
	if newline != -1 {
		portLine = webBlock[portLineIdx : portLineIdx+newline]
	} else {
		portLine = webBlock[portLineIdx:]
	}
	if strings.Contains(portLine, "'geo'") {
		t.Errorf("config.js: web/port entry must NOT include 'geo'; got: %s", strings.TrimSpace(portLine))
	}
}

// TestStaticJS_AdvOpt_FormUnchecksOnHide verifies that updateAdvancedOpts()
// not only hides and disables inputs inside hidden wrappers but also unchecks
// checkboxes, so a previously-ticked option never silently affects the
// submitted request payload when its wrapper becomes hidden.
func TestStaticJS_AdvOpt_FormUnchecksOnHide(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	fnStart := strings.Index(body, "function updateAdvancedOpts(")
	if fnStart == -1 {
		t.Fatal("form.js: updateAdvancedOpts function not found")
	}
	// Capture the function body up to the next sibling function.
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
	// Must branch on input type to target only checkboxes.
	if !strings.Contains(fnBody, "inp.type === 'checkbox'") {
		t.Error("form.js: updateAdvancedOpts must check inp.type === 'checkbox' before unchecking")
	}
	// Must explicitly clear the checked state.
	if !strings.Contains(fnBody, "inp.checked = false") {
		t.Error("form.js: updateAdvancedOpts must set inp.checked = false when hiding a checkbox option")
	}
}

// TestStaticJS_AdvOpt_NonWebTargetEntries verifies that ADV_OPT_SUPPORT
// includes entries for all non-web diagnostic targets.
func TestStaticJS_AdvOpt_NonWebTargetEntries(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/config.js")

	for _, target := range []string{"smtp:", "imap:", "pop:", "ftp:", "sftp:"} {
		if !strings.Contains(body, target) {
			t.Errorf("config.js: ADV_OPT_SUPPORT must have an entry for target %q", target)
		}
	}
}

// TestStaticJS_AdvOpt_FormHasUpdateFunction verifies that form.js defines the
// updateAdvancedOpts() function and uses the ADV_OPT_SUPPORT matrix and the
// _ADV_OPT_KEYS constant to drive visibility.
func TestStaticJS_AdvOpt_FormHasUpdateFunction(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	if !strings.Contains(body, "function updateAdvancedOpts(") {
		t.Error("form.js: updateAdvancedOpts function must be defined")
	}
	if !strings.Contains(body, "ADV_OPT_SUPPORT") {
		t.Error("form.js: updateAdvancedOpts must read ADV_OPT_SUPPORT from config via _advOptSupport()")
	}
	if !strings.Contains(body, "_ADV_OPT_KEYS") {
		t.Error("form.js: _ADV_OPT_KEYS constant must be declared and used to iterate option wrappers")
	}
}

// TestStaticJS_AdvOpt_FormWiredOnTargetChange verifies that updateAdvancedOpts
// is called inside onTargetChange so switching targets updates option visibility.
func TestStaticJS_AdvOpt_FormWiredOnTargetChange(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	fnStart := strings.Index(body, "function onTargetChange(")
	if fnStart == -1 {
		t.Fatal("form.js: onTargetChange not found")
	}
	// Capture a generous window covering the entire function body.  The next
	// top-level inner function acts as a reliable end-marker.
	endMarker := "\n  function "
	nextFn := strings.Index(body[fnStart+1:], endMarker)
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
	if !strings.Contains(fnBody, "updateAdvancedOpts(") {
		t.Error("form.js: onTargetChange must call updateAdvancedOpts() to refresh advanced option visibility on target switch")
	}
}

// TestStaticJS_AdvOpt_FormWiredOnModeChange verifies that updateAdvancedOpts
// is called inside the mode radio change listener so switching sub-modes also
// updates visibility.
func TestStaticJS_AdvOpt_FormWiredOnModeChange(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	listenerStr := "radio.addEventListener('change'"
	idx := strings.Index(body, listenerStr)
	if idx == -1 {
		t.Fatal("form.js: mode radio change listener (radio.addEventListener('change'...)) not found")
	}
	// Capture a window large enough to cover the entire listener callback.
	end := idx + 800
	if end > len(body) {
		end = len(body)
	}
	listenerBody := body[idx:end]
	if !strings.Contains(listenerBody, "updateAdvancedOpts(") {
		t.Error("form.js: mode radio change listener must call updateAdvancedOpts() to refresh advanced option visibility on mode switch")
	}
}

// TestStaticJS_AdvOpt_FormExported verifies that updateAdvancedOpts is included
// in the PathProbe.Form namespace export for testability and external use.
func TestStaticJS_AdvOpt_FormExported(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/form.js")

	exportLine := "PathProbe.Form = {"
	exportIdx := strings.Index(body, exportLine)
	if exportIdx == -1 {
		t.Fatal("form.js: PathProbe.Form export not found")
	}
	// The export object is a single line; 200 bytes is ample.
	end := exportIdx + 200
	if end > len(body) {
		end = len(body)
	}
	exportBlock := body[exportIdx:end]
	if !strings.Contains(exportBlock, "updateAdvancedOpts") {
		t.Error("form.js: updateAdvancedOpts must be exported in PathProbe.Form (for external and test access)")
	}
}

// TestStaticHTML_AdvOpt_WrapperIDs verifies that index.html contains the three
// wrapper element IDs that updateAdvancedOpts() targets to show/hide options.
func TestStaticHTML_AdvOpt_WrapperIDs(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, id := range []string{"adv-opt-mtr-count", "adv-opt-insecure", "adv-opt-geo"} {
		if !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("index.html: advanced option wrapper element with id=%q must exist for dynamic show/hide to work", id)
		}
	}
}
