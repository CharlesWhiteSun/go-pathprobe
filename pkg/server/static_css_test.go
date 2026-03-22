package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticCSS_ButtonFixedDimensions verifies that the embedded style.css declares
// all fixed-dimension properties required to prevent layout shift on buttons.
func TestStaticCSS_ButtonFixedDimensions(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// .lang-btn: explicit fixed width prevents re-flow when EN/TW ↔ 英文/繁中.
	if !strings.Contains(body, "width: 2.8rem") {
		t.Error("style.css: .lang-btn must declare 'width: 2.8rem' to prevent locale-switch layout shift")
	}

	// #run-btn: square icon-only button — both width and height must be fixed.
	if !strings.Contains(body, "width: 2.75rem") {
		t.Error("style.css: #run-btn must declare 'width: 2.75rem' for icon-only square shape")
	}
	if !strings.Contains(body, "height: 2.75rem") {
		t.Error("style.css: #run-btn must declare 'height: 2.75rem' to prevent vertical layout shift")
	}
}

// TestStaticCSS_CancelBtnMatchesRunBtn 驗證 .btn-cancel 的尺寸與 #run-btn 一致
// (同為 2.75rem 方形圖示按鈕)，且透過 CSS ::before 偽元素繪製停止圖示。
func TestStaticCSS_CancelBtnMatchesRunBtn(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Locate .btn-cancel rule block.
	cancelStart := strings.Index(body, ".btn-cancel")
	if cancelStart == -1 {
		t.Fatal("style.css: .btn-cancel rule not found")
	}

	// The cancel button section must declare matching square dimensions.
	sectionEnd := strings.Index(body[cancelStart:], "/*  History Panel  */")
	if sectionEnd == -1 {
		sectionEnd = 800
	}
	section := body[cancelStart : cancelStart+sectionEnd]

	if !strings.Contains(section, "width: 2.75rem") {
		t.Error("style.css: .btn-cancel must declare 'width: 2.75rem' to match #run-btn")
	}
	if !strings.Contains(section, "height: 2.75rem") {
		t.Error("style.css: .btn-cancel must declare 'height: 2.75rem' to match #run-btn")
	}
	// Stop icon must be drawn via CSS ::before pseudo-element (no text/glyph dependency).
	if !strings.Contains(section, ".btn-cancel::before") {
		t.Error("style.css: .btn-cancel must use ::before pseudo-element for the stop icon")
	}
}

// TestStaticCSS_ThemeBarButtons verifies that the embedded style.css defines
// the circular dot-button styles for the .theme-bar switcher. It confirms:
//  1. .theme-btn uses border-radius: 50% to produce a circle.
//  2. Each of the five themes has a per-theme background rule targeting the
//     button element via .theme-btn[data-theme="..."], keeping button colours
//     independent of the active page theme.
//  3. Flat-design constraints: no linear-gradient (flashy half-split removed)
//     and no transform: scale in hover/active (no distracting zoom effect).
func TestStaticCSS_ThemeBarButtons(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Base circle shape.
	if !strings.Contains(body, "border-radius: 50%") {
		t.Error("style.css: .theme-btn must declare 'border-radius: 50%' for circular shape")
	}

	// Per-theme button colour rules (independent of page-level data-theme).
	for _, theme := range []string{"forest-green", "light-green", "default", "deep-blue", "dark"} {
		selector := `.theme-btn[data-theme="` + theme + `"]`
		if !strings.Contains(body, selector) {
			t.Errorf("style.css: missing per-button colour rule for theme %q (expected selector %s)", theme, selector)
		}
	}

	// Flat-design: buttons must use solid colour only (no gradient).
	// Scoped to the theme-bar section below (after themeBarSec is computed)
	// to avoid false positives from other components — e.g. the diamond-split
	// marker style intentionally uses linear-gradient for its two-tone effect.

	// Flat-design: no scale transform on hover/active (avoids flashy zoom).
	// Scope the check to only the theme-bar section to avoid false positives
	// from other components that legitimately use scale transforms (e.g.
	// the custom-select popup uses scaleY for its entrance animation).
	// The section comment may carry an inline description after the mark;
	// search only for the fixed prefix that will always be present.
	themeSectionMark := "/* \u2500\u2500 Theme bar"
	themeSecStart := strings.Index(body, themeSectionMark)
	if themeSecStart == -1 {
		t.Fatalf("style.css: '/* ── Theme bar …' section comment not found; snippet around 'theme-btn':\n%s",
			func() string {
				idx := strings.Index(body, ".theme-btn")
				if idx == -1 {
					return "(.theme-btn not found)"
				}
				start := idx - 120
				if start < 0 {
					start = 0
				}
				end := idx + 120
				if end > len(body) {
					end = len(body)
				}
				return body[start:end]
			}())
	}
	themeBarSec := body[themeSecStart:]
	if nextSec := strings.Index(themeBarSec[len(themeSectionMark):], "/* \u2500\u2500"); nextSec != -1 {
		themeBarSec = themeBarSec[:len(themeSectionMark)+nextSec]
	}
	if strings.Contains(themeBarSec, "transform: scale") {
		t.Error("style.css: .theme-btn must not use transform: scale — flat transition only")
	}
	if strings.Contains(themeBarSec, "linear-gradient") {
		t.Error("style.css: .theme-btn must not use linear-gradient — flat solid colour only")
	}

	// Old <select> style must be gone.
	if strings.Contains(body, "#theme-select") {
		t.Error("style.css: old #theme-select rule must be removed")
	}
}

// TestStaticCSS_ThemeBarFlat verifies that .header-brand exists in CSS to
// support the left-column flex layout of the 3-column header.
func TestStaticCSS_ThemeBarFlat(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// .header-brand must be defined to anchor the left column with flex: 1.
	if !strings.Contains(body, ".header-brand") {
		t.Error("style.css: .header-brand rule must be defined for 3-column header layout")
	}
	// .lang-switcher must use flex: 1 (not margin-left: auto) to mirror the brand column.
	if strings.Contains(body, "margin-left: auto") {
		t.Error("style.css: lang-switcher must not use margin-left: auto in the 3-column layout")
	}
}

// TestStaticCSS_ThemeVariables verifies that the embedded style.css contains
// CSS variable override blocks for all four non-default themes. Each block is
// identified by the [data-theme="..."] attribute selector; the presence of the
// selector proves the theme can be activated purely via a data attribute, with
// no additional JavaScript style manipulation required.
func TestStaticCSS_ThemeVariables(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Each non-default theme must have its own [data-theme] block.
	for _, theme := range []string{"deep-blue", "light-green", "forest-green", "dark"} {
		selector := `[data-theme="` + theme + `"]`
		if !strings.Contains(body, selector) {
			t.Errorf("style.css: missing theme block for %q (expected selector %s)", theme, selector)
		}
	}

	// The key component-level token variables must be tokenised in :root so
	// theme overrides propagate through to all component rules.
	for _, token := range []string{
		"--input-bg", "--error-bg", "--error-border",
		"--badge-ok-bg", "--badge-fail-bg", "--focus-ring", "--surface-alt",
	} {
		if !strings.Contains(body, token) {
			t.Errorf("style.css: :root must declare the %q CSS variable for theme overrides to work", token)
		}
	}
}

// TestStaticCSS_BrandTypography verifies that the embedded style.css contains
// the --brand-font token, individual brand-span rules, and the commented-out
// @font-face swap-point template so future custom fonts require only updating
// that one CSS variable.
func TestStaticCSS_BrandTypography(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	checks := []struct {
		needle string
		msg    string
	}{
		{"--brand-font", "style.css: --brand-font token must be declared in :root"},
		{".brand-path", "style.css: .brand-path rule must exist"},
		{".brand-probe", "style.css: .brand-probe rule must exist"},
		{"@font-face", "style.css: @font-face swap-point template must be present (as a comment)"},
		{"font-display: swap", "style.css: @font-face template must include font-display: swap"},
		{"brand.woff2", "style.css: @font-face template must reference brand.woff2"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.needle) {
			t.Error(c.msg)
		}
	}
}

// TestStaticCSS_HeaderPaddingToken verifies that the embedded style.css uses a
// --header-py CSS custom property for vertical header padding.  This makes
// header height adjustments a single-token change with no selector hunting.
func TestStaticCSS_HeaderPaddingToken(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "--header-py") {
		t.Error("style.css: --header-py token must be declared in :root")
	}
	if !strings.Contains(body, "var(--header-py)") {
		t.Error("style.css: .site-header must consume var(--header-py) for vertical padding")
	}
}

// TestStaticCSS_BrandLogoSizeTokens verifies that style.css declares a unified
// --brand-logo-size token in :root and that both .brand-path and .brand-probe
// consume it via var(), so both glyphs always share the same size.
func TestStaticCSS_BrandLogoSizeTokens(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "--brand-logo-size") {
		t.Error("style.css: --brand-logo-size token must be declared in :root")
	}
	// Both glyphs must reference the unified token — no separate size tokens.
	if strings.Contains(body, "--brand-path-size") {
		t.Error("style.css: --brand-path-size must not exist; use --brand-logo-size instead")
	}
	if strings.Contains(body, "--brand-probe-size") {
		t.Error("style.css: --brand-probe-size must not exist; use --brand-logo-size instead")
	}
	// Count occurrences of var(--brand-logo-size): must appear for .brand-path AND .brand-probe.
	count := strings.Count(body, "var(--brand-logo-size)")
	if count < 2 {
		t.Errorf("style.css: var(--brand-logo-size) must be used at least twice (brand-path + brand-probe), got %d", count)
	}
}

// TestStaticCSS_ModeSelector verifies the .mode-selector and .mode-option style rules exist.
func TestStaticCSS_ModeSelector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	for _, rule := range []string{".mode-selector", ".mode-option"} {
		if !strings.Contains(body, rule) {
			t.Errorf("style.css: %s rule must be defined", rule)
		}
	}
}

// TestStaticCSS_HeaderShadow verifies that the embedded style.css declares a
// --header-shadow CSS token in :root and that .site-header consumes it via
// var(--header-shadow), keeping the shadow value a single-token change.
func TestStaticCSS_HeaderShadow(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "--header-shadow") {
		t.Error("style.css: --header-shadow token must be declared in :root")
	}
	if !strings.Contains(body, "var(--header-shadow)") {
		t.Error("style.css: .site-header must consume var(--header-shadow)")
	}
}

// TestStaticCSS_StickyHeader verifies that the embedded style.css makes the
// site header stick to the top of the viewport while the page is scrolled.
// position: sticky + top: 0 achieves this without removing the header from
// normal document flow (unlike position: fixed), so .main requires no extra
// margin-top compensation.  z-index ensures the header layers above all cards.
func TestStaticCSS_StickyHeader(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "position: sticky") {
		t.Error("style.css: .site-header must declare 'position: sticky' to stay visible during scroll")
	}
	if !strings.Contains(body, "top: 0") {
		t.Error("style.css: .site-header must declare 'top: 0' to anchor at the viewport top")
	}
	if !strings.Contains(body, "z-index: 100") {
		t.Error("style.css: .site-header must declare 'z-index: 100' to layer above page content")
	}
}

// TestStaticCSS_SelectCustomChevron verifies that the embedded style.css
// removes the native OS dropdown arrow and replaces it with a custom chevron
// that follows the active theme's --primary colour via CSS mask-image.
// Both .select-wrap and .cs-wrap must carry this chevron so legacy native
// selects and the new custom-select widget look identical.
func TestStaticCSS_SelectCustomChevron(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Native arrow must be suppressed.
	if !strings.Contains(body, "appearance: none") {
		t.Error("style.css: select must declare 'appearance: none' to remove the native OS arrow")
	}
	if !strings.Contains(body, "-webkit-appearance: none") {
		t.Error("style.css: select must declare '-webkit-appearance: none' for Safari/Chrome compat")
	}

	// Both wrapper classes must be defined.
	for _, cls := range []string{".select-wrap", ".cs-wrap"} {
		if !strings.Contains(body, cls) {
			t.Errorf("style.css: %s rule must exist as a positioning context for the chevron", cls)
		}
	}

	// Custom chevron uses mask-image so background-color: var(--primary) provides
	// the colour — automatically correct for every theme.
	if !strings.Contains(body, "mask-image") {
		t.Error("style.css: chevron must use mask-image for the theme-aware colouring")
	}
	if !strings.Contains(body, "background-color: var(--primary)") {
		t.Error("style.css: chevron must use background-color: var(--primary) so colour tracks the active theme")
	}
	// Rotation signal: cs-wrap.open must rotate the chevron 180°.
	if !strings.Contains(body, "rotate(180deg)") {
		t.Error("style.css: .cs-wrap.open::after must rotate the chevron 180deg to indicate open state")
	}
}

// TestStaticCSS_CustomSelectPopup verifies that style.css defines the
// cs-* component rules with theme-aware tokens for the popup's visual style.
// Specifically: rounded corners (--select-popup-r), layered shadow
// (--select-popup-shadow), surface background, and a smooth opacity+scale
// entrance transition that is impossible with the OS-native dropdown.
func TestStaticCSS_CustomSelectPopup(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Token declarations in :root.
	for _, token := range []string{"--select-popup-shadow", "--select-popup-r"} {
		if !strings.Contains(body, token) {
			t.Errorf("style.css: %s token must be declared in :root", token)
		}
	}

	// Component rules.
	for _, rule := range []string{".cs-wrap", ".cs-trigger", ".cs-list", ".cs-item"} {
		if !strings.Contains(body, rule) {
			t.Errorf("style.css: %s rule must be defined for the custom-select component", rule)
		}
	}

	// Popup uses theme tokens for background and shadow.
	if !strings.Contains(body, "var(--select-popup-shadow)") {
		t.Error("style.css: .cs-list must consume var(--select-popup-shadow)")
	}
	if !strings.Contains(body, "var(--select-popup-r)") {
		t.Error("style.css: .cs-list must consume var(--select-popup-r) for themed corner radius")
	}

	// Popup entrance is driven by opacity + transform transitions.
	if !strings.Contains(body, "scaleY") {
		t.Error("style.css: .cs-list entrance animation must include a scaleY transform for a natural dropdown feel")
	}
	// Popup animation duration and scale are driven by CSS tokens.
	if !strings.Contains(body, "var(--cs-popup-anim-dur)") {
		t.Error("style.css: .cs-list transition must consume var(--cs-popup-anim-dur) instead of a hard-coded value")
	}
	if !strings.Contains(body, "var(--cs-popup-anim-scale)") {
		t.Error("style.css: .cs-list transform must consume var(--cs-popup-anim-scale) instead of a hard-coded value")
	}
	// .cs-wrap.open reveals the list.
	if !strings.Contains(body, ".cs-wrap.open .cs-list") {
		t.Error("style.css: .cs-wrap.open .cs-list selector must make the popup visible")
	}
	// Selected item uses primary colour.
	if !strings.Contains(body, `.cs-item[aria-selected="true"]`) {
		t.Error("style.css: cs-item[aria-selected=\"true\"] must be styled for the active selection")
	}
}

// TestStaticCSS_PanelTransition verifies that style.css declares the
// panel-appear @keyframes and the .target-fields.panel-entering rule so
// onTargetChange() can trigger the fade-in animation without extra CSS.
func TestStaticCSS_PanelTransition(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "@keyframes panel-appear") {
		t.Error("style.css: @keyframes panel-appear must be declared for the target fieldset entrance animation")
	}
	if !strings.Contains(body, ".target-fields.panel-entering") {
		t.Error("style.css: .target-fields.panel-entering must consume the panel-appear animation")
	}
	// Exit animation: departing panel must also animate out.
	if !strings.Contains(body, "@keyframes panel-leave") {
		t.Error("style.css: @keyframes panel-leave must be declared for the target fieldset exit animation")
	}
	if !strings.Contains(body, ".target-fields.panel-leaving") {
		t.Error("style.css: .target-fields.panel-leaving must consume the panel-leave animation")
	}
	// Animation must use opacity (fade) and a vertical transform (slide).
	if !strings.Contains(body, "translateY") {
		t.Error("style.css: panel animations must include translateY for the entrance/exit slide effect")
	}
	// Duration and distance must be driven by CSS tokens (not hard-coded values).
	if !strings.Contains(body, "var(--panel-anim-dur)") {
		t.Error("style.css: panel transition must consume var(--panel-anim-dur) instead of a hard-coded duration")
	}
	if !strings.Contains(body, "var(--panel-anim-dist)") {
		t.Error("style.css: panel-appear keyframe must consume var(--panel-anim-dist) instead of a hard-coded pixel offset")
	}
	// The panel-stage wrapper must clip exiting panels and animate its own
	// height smoothly so the card never jumps when switching between panels of
	// different heights.
	if !strings.Contains(body, ".panel-stage") {
		t.Error("style.css: .panel-stage rule must be declared to wrap all .target-fields fieldsets")
	}
	if !strings.Contains(body, "overflow: hidden") {
		t.Error("style.css: .panel-stage must set overflow: hidden to clip the exit animation")
	}
	if !strings.Contains(body, "transition: height var(--panel-anim-dur)") {
		t.Error("style.css: .panel-stage must animate height via transition: height var(--panel-anim-dur)")
	}
}

// TestStaticCSS_FooterStyles verifies that the embedded style.css defines the
// three footer component rules (.site-footer, .footer-inner, .footer-copy) and
// the --footer-shadow design token.  This ensures the footer can be restyled by
// changing a single token just like the header.
func TestStaticCSS_FooterStyles(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// All three footer component selectors must be defined.
	for _, rule := range []string{".site-footer", ".footer-inner", ".footer-copy"} {
		if !strings.Contains(body, rule) {
			t.Errorf("style.css: %s rule must be defined", rule)
		}
	}
	// Footer must reuse the same --header-py token for vertical rhythm parity.
	if !strings.Contains(body, "var(--header-py)") {
		t.Error("style.css: .site-footer must reuse var(--header-py) for vertically consistent rhythm with the header")
	}
	// The --footer-shadow token must be declared and consumed.
	if !strings.Contains(body, "--footer-shadow") {
		t.Error("style.css: --footer-shadow token must be declared in :root")
	}
	if !strings.Contains(body, "var(--footer-shadow)") {
		t.Error("style.css: .site-footer must consume var(--footer-shadow)")
	}
	// Footer must NOT be sticky or fixed — it should flow with the document.
	// We narrow the check to only the footer CSS section by using the section
	// comment marker "/* ── Footer" as the start boundary and the next "/* ──"
	// section marker as the end boundary.  This avoids false positives from
	// the header section which legitimately declares position: sticky.
	sectionMark := "/* \u2500\u2500 Footer"
	footerSecStart := strings.Index(body, sectionMark)
	if footerSecStart == -1 {
		t.Fatal("style.css: '/* ── Footer' section comment not found")
	}
	footerSec := body[footerSecStart+len(sectionMark):]
	if nextSec := strings.Index(footerSec, "/* \u2500\u2500"); nextSec != -1 {
		footerSec = footerSec[:nextSec]
	}
	if strings.Contains(footerSec, "position: sticky") || strings.Contains(footerSec, "position: fixed") {
		t.Error("style.css: .site-footer must NOT be sticky or fixed — it must flow with the document")
	}
}

// TestStaticCSS_BodyFlushBottom verifies that the embedded style.css configures
// the body as a flex-column container with min-height: 100vh, and that .main
// carries flex: 1 and width: 100%.  Together these rules guarantee:
//   - the footer is always pressed to the viewport bottom on short pages, and
//   - .main fills the full available width (up to max-width: 960px) instead of
//     shrinking to its intrinsic content width (flex cross-axis shrink-to-fit).
func TestStaticCSS_BodyFlushBottom(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "min-height: 100vh") {
		t.Error("style.css: body must declare 'min-height: 100vh' so the footer reaches the bottom on short pages")
	}
	if !strings.Contains(body, "flex-direction: column") {
		t.Error("style.css: body must declare 'flex-direction: column' for the header-main-footer stack")
	}
	if !strings.Contains(body, "flex: 1") {
		t.Error("style.css: .main must declare 'flex: 1' to fill remaining space above the footer")
	}
	// width: 100% is required so that margin: auto on the cross axis of the body
	// flex container does not trigger shrink-to-fit, which would squeeze the
	// diagnostic and history cards narrower than their intended 960px maximum.
	if !strings.Contains(body, "width: 100%") {
		t.Error("style.css: .main must declare 'width: 100%' to prevent shrink-to-fit inside the body flex container")
	}
}

// TestStaticCSS_ChromeHeightParity verifies that the embedded style.css
// declares a --chrome-inner-h design token and applies it as min-height to
// both .header-inner and .footer-inner.  This single token guarantees the
// visible chrome bars (header + footer) have identical height regardless of
// their text content size difference, producing a visually balanced bookend.
func TestStaticCSS_ChromeHeightParity(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Token must be declared in :root so themes can override it.
	if !strings.Contains(body, "--chrome-inner-h") {
		t.Error("style.css: --chrome-inner-h token must be declared in :root for header/footer height parity")
	}
	// Both inner containers must consume the token.
	count := strings.Count(body, "var(--chrome-inner-h)")
	if count < 2 {
		t.Errorf("style.css: var(--chrome-inner-h) must appear at least twice (header-inner + footer-inner), got %d", count)
	}
}

// TestStaticCSS_SelectOptionTheming verifies that style.css defines theme-aware
// option styling using only CSS custom-property tokens.  A single pair of rules
// (select option + option:checked) automatically covers every theme because
// each [data-theme] block overrides the tokens they reference — no per-theme
// CSS duplication is needed.
func TestStaticCSS_SelectOptionTheming(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Locate the option-theming section so assertions are scoped to it.
	sectionMark := "/* \u2500\u2500 Select option theming"
	secStart := strings.Index(body, sectionMark)
	if secStart == -1 {
		t.Fatal("style.css: '/* ── Select option theming' section comment not found")
	}
	sec := body[secStart+len(sectionMark):]
	if nextSec := strings.Index(sec, "/* \u2500\u2500"); nextSec != -1 {
		sec = sec[:nextSec]
	}

	// Base rule: options must display the theme's input-surface background and text colour.
	if !strings.Contains(sec, "select option") {
		t.Error("style.css: 'select option' selector must be present in the option-theming section")
	}
	if !strings.Contains(sec, "var(--input-bg)") {
		t.Error("style.css: select option background-color must reference var(--input-bg) to track the theme's input surface")
	}
	if !strings.Contains(sec, "var(--text)") {
		t.Error("style.css: select option color must reference var(--text) for legible text across all themes")
	}

	// Checked/selected state must highlight using the primary colour.
	if !strings.Contains(sec, "option:checked") {
		t.Error("style.css: 'option:checked' selector must be defined for the selected-option highlight")
	}
	if !strings.Contains(sec, "var(--primary)") {
		t.Error("style.css: option:checked background-color must reference var(--primary)")
	}
	// Foreground must use the --option-checked-fg token so themes with a light
	// primary colour can override it for adequate contrast without a new CSS block.
	if !strings.Contains(sec, "var(--option-checked-fg)") {
		t.Error("style.css: option:checked color must reference var(--option-checked-fg) for per-theme contrast control")
	}
}

// TestStaticCSS_OptionCheckedFgToken verifies that --option-checked-fg is
// declared in :root (defaulting to #fff) and that [data-theme="dark"] overrides
// it to a dark tint.  The dark theme's primary is #bb86fc (light purple), so
// white text would give only ~2.8:1 contrast; the surface override raises this
// to ~7.5:1, well above the WCAG AA threshold.
func TestStaticCSS_OptionCheckedFgToken(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Token must be declared somewhere in the stylesheet.
	if !strings.Contains(body, "--option-checked-fg") {
		t.Error("style.css: --option-checked-fg token must be declared in :root")
	}

	// Locate the dark-theme block and verify it overrides the token.
	// Search for the standalone block (prefixed with newline + selector + space)
	// to avoid accidentally matching .theme-btn[data-theme="dark"] which appears
	// earlier in the CSS for the header swatch buttons.
	darkMark := "\n[data-theme=\"dark\"] {"
	darkIdx := strings.Index(body, darkMark)
	if darkIdx == -1 {
		t.Fatalf("style.css: standalone [data-theme=\"dark\"] { block not found")
	}
	darkBlock := body[darkIdx:]
	// Trim to just this block (ends at the first bare closing brace on its own line).
	if closeIdx := strings.Index(darkBlock, "\n}"); closeIdx != -1 {
		darkBlock = darkBlock[:closeIdx+2]
	}
	if !strings.Contains(darkBlock, "--option-checked-fg") {
		t.Errorf("style.css: [data-theme=\"dark\"] must override --option-checked-fg for legible text on the light-purple primary (#bb86fc)")
	}
}

// TestStaticCSS_AnimationTokens verifies that style.css declares the four
// animation design tokens in :root and implements the [data-anim="vivid"] and
// [data-anim="off"] override blocks so JS can switch animation intensity by
// toggling a single HTML attribute.
func TestStaticCSS_AnimationTokens(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// All four tokens must be declared in :root.
	for _, token := range []string{
		"--panel-anim-dur",
		"--panel-anim-dist",
		"--cs-popup-anim-dur",
		"--cs-popup-anim-scale",
	} {
		if !strings.Contains(body, token) {
			t.Errorf("style.css: animation token %s must be declared in :root", token)
		}
	}

	// vivid and off mode blocks must exist.
	if !strings.Contains(body, `[data-anim="vivid"]`) {
		t.Error(`style.css: [data-anim="vivid"] override block must be present`)
	}
	if !strings.Contains(body, `[data-anim="off"]`) {
		t.Error(`style.css: [data-anim="off"] override block must be present`)
	}
}

// TestStaticCSS_CustomSelectHasSelection verifies that style.css defines a
// persistent visual indicator for .cs-wrap.has-selection .cs-trigger so the
// widget looks "selected" even when it does not have keyboard focus — mirroring
// the always-visible highlight of a checked radio button.
func TestStaticCSS_CustomSelectHasSelection(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, ".cs-wrap.has-selection .cs-trigger") {
		t.Error("style.css: .cs-wrap.has-selection .cs-trigger rule must be defined for persistent selection indicator")
	}
	if !strings.Contains(body, "border-color: var(--primary)") {
		t.Error("style.css: .cs-wrap.has-selection .cs-trigger must set border-color: var(--primary)")
	}
	// Background tint uses color-mix for accessible, theme-aware contrast.
	if !strings.Contains(body, "color-mix") {
		t.Error("style.css: .cs-wrap.has-selection .cs-trigger should use color-mix() for a subtle primary background tint")
	}
}

// TestStaticCSS_PanelLeaveAnimation verifies that style.css defines both
// halves of the panel cross-fade: @keyframes panel-leave and the
// .target-fields.panel-leaving rule.  The leave direction (upward slide) must
// be the mirror of the enter direction so the transition feels directional.
func TestStaticCSS_PanelLeaveAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "@keyframes panel-leave") {
		t.Error("style.css: @keyframes panel-leave must be declared for the target fieldset exit animation")
	}
	if !strings.Contains(body, ".target-fields.panel-leaving") {
		t.Error("style.css: .target-fields.panel-leaving rule must consume panel-leave so onTargetChange() can trigger it")
	}
	// Leave animation must move upward — opposite direction to the enter slide.
	if !strings.Contains(body, "calc(-1 * var(--panel-anim-dist))") {
		t.Error("style.css: panel-leave must use calc(-1 * var(--panel-anim-dist)) for the mirrored upward slide")
	}
	// Interaction must be blocked during the fade-out to prevent stray clicks.
	if !strings.Contains(body, "pointer-events: none") {
		t.Error("style.css: .target-fields.panel-leaving must declare pointer-events: none to block stray clicks during fade-out")
	}
}

// TestStaticCSS_AdvancedOptsAnimation verifies that style.css declares the
// rules required for the Advanced Options animated expand/collapse, and that
// they reuse the shared panel-appear / panel-leave keyframes and
// --panel-anim-dur token so vivid / off modes apply automatically.
func TestStaticCSS_AdvancedOptsAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// The height-animated container must have overflow:hidden to clip the content.
	if !strings.Contains(body, ".advanced-opts .adv-body") {
		t.Error("style.css: .advanced-opts .adv-body rule must be declared as the height-transition container")
	}
	// Height transition must consume the shared token, not a hard-coded value.
	if !strings.Contains(body, "transition: height var(--panel-anim-dur)") {
		t.Error("style.css: .adv-body must use transition: height var(--panel-anim-dur) so vivid/off modes apply")
	}
	// Entering animation must reuse panel-appear so the feel matches panel transitions.
	if !strings.Contains(body, "adv-entering") {
		t.Error("style.css: .adv-body.adv-entering rule must be declared to trigger the entrance animation")
	}
	if !strings.Contains(body, "adv-leaving") {
		t.Error("style.css: .adv-body.adv-leaving rule must be declared to trigger the exit animation")
	}
	// Both states must delegate to the shared keyframes to avoid duplication.
	if !strings.Contains(body, "panel-appear") {
		t.Error("style.css: adv-entering animation must reuse the panel-appear keyframes")
	}
	if !strings.Contains(body, "panel-leave") {
		t.Error("style.css: adv-leaving animation must reuse the panel-leave keyframes")
	}
	// The native browser triangle marker must be suppressed so the custom
	// ::before chevron is the only visible indicator.
	if !strings.Contains(body, "::-webkit-details-marker") {
		t.Error("style.css: .advanced-opts > summary::-webkit-details-marker must be hidden to suppress the native Chrome/Safari triangle")
	}
	if !strings.Contains(body, "list-style: none") {
		t.Error("style.css: .advanced-opts > summary must set list-style:none to suppress the native Firefox triangle marker")
	}
	// summary::before must carry the animated chevron.
	if !strings.Contains(body, "summary::before") {
		t.Error("style.css: .advanced-opts > summary::before rule must exist to render the custom animated chevron")
	}
	// The native Firefox ::marker must also be suppressed (belt-and-suspenders).
	if !strings.Contains(body, "summary::marker") {
		t.Error("style.css: .advanced-opts > summary::marker must blank the native Firefox arrow")
	}
	// Chevron rotation must use ease-in-out for an elegant deceleration.
	if !strings.Contains(body, "ease-in-out") {
		t.Error("style.css: summary::before transition must use ease-in-out for a graceful rotation feel")
	}
	// Duration is driven by --panel-anim-dur (via calc) so vivid/off cascade.
	// The multiplier must be 1.2 (= original 1.8 ÷ 1.5, i.e. 50% faster).
	if !strings.Contains(body, "* 1.2") {
		t.Error("style.css: summary::before transition duration multiplier must be 1.2 (50% faster than the original 1.8x setting)")
	}
	if !strings.Contains(body, "var(--panel-anim-dur)") {
		t.Error("style.css: summary::before transition duration must consume var(--panel-anim-dur) so vivid/off modes apply automatically")
	}
	// .adv-is-open class (not [open] attribute) drives the open rotation so
	// the chevron is always in sync with the height transition direction.
	if !strings.Contains(body, "adv-is-open") {
		t.Error("style.css: .adv-is-open class must be declared to rotate the chevron in sync with the height animation")
	}
}

// TestStaticCSS_CustomCheckbox verifies that style.css replaces the native
// checkbox appearance with a fully themed custom box driven by design tokens.
// Specifically it checks:
//   - The native input is hidden (appearance:none + position:absolute + opacity:0)
//   - span::before draws the custom box sized by --cb-size token
//   - The --cb-radius token controls the corner radius
//   - The --cb-anim-dur token drives the transition so vivid/off modes apply
//   - The checked state applies the primary background colour
//   - A white SVG checkmark is embedded as a background-image data-URI
//   - A focus-visible rule adds the focus ring via box-shadow + --focus-ring token
//   - Hover states exist for both unchecked (border highlight) and checked (darken)
func TestStaticCSS_CustomCheckbox(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Design tokens must be declared so they can be overridden per theme / anim mode.
	if !strings.Contains(body, "--cb-size") {
		t.Error("style.css: --cb-size token must be declared in :root for the custom checkbox box dimensions")
	}
	if !strings.Contains(body, "--cb-radius") {
		t.Error("style.css: --cb-radius token must be declared in :root for the custom checkbox corner radius")
	}
	if !strings.Contains(body, "--cb-anim-dur") {
		t.Error("style.css: --cb-anim-dur token must be declared in :root so vivid/off animation modes apply to checkboxes")
	}
	// Vivid and off modes must each override --cb-anim-dur so the token
	// system is consistent with panel and popup animation tokens.
	// Search from the opening brace of each selector to avoid matching the
	// inline comment in :root that also contains the literal text.
	vividStart := strings.Index(body, "[data-anim=\"vivid\"] {")
	offStart := strings.Index(body, "[data-anim=\"off\"] {")
	if vividStart == -1 || !strings.Contains(body[vividStart:vividStart+400], "--cb-anim-dur") {
		t.Error("style.css: [data-anim=\"vivid\"] must override --cb-anim-dur")
	}
	if offStart == -1 || !strings.Contains(body[offStart:offStart+400], "--cb-anim-dur") {
		t.Error("style.css: [data-anim=\"off\"] must override --cb-anim-dur")
	}
	// Native checkbox must be visually hidden.
	if !strings.Contains(body, "appearance: none") {
		t.Error("style.css: .checkbox-row input[type=checkbox] must set appearance:none to suppress native rendering")
	}
	// span::before must be declared as the custom visual box target.
	if !strings.Contains(body, "input[type=checkbox] + span::before") {
		t.Error("style.css: input[type=checkbox] + span::before selector must exist to draw the custom checkbox box")
	}
	// Box dimensions must reference the --cb-size token.
	if !strings.Contains(body, "var(--cb-size)") {
		t.Error("style.css: span::before must use var(--cb-size) for width/height so the box dimension is token-driven")
	}
	// Corner radius must reference the --cb-radius token.
	if !strings.Contains(body, "var(--cb-radius)") {
		t.Error("style.css: span::before must use var(--cb-radius) for border-radius so the shape is token-driven")
	}
	// Transition must consume --cb-anim-dur so speed is token-controlled.
	if !strings.Contains(body, "var(--cb-anim-dur)") {
		t.Error("style.css: span::before transition must reference var(--cb-anim-dur)")
	}
	// Checked state must apply the primary colour.
	if !strings.Contains(body, "input[type=checkbox]:checked + span::before") {
		t.Error("style.css: :checked + span::before selector must exist to fill the box with the primary colour")
	}
	if !strings.Contains(body, "background-color: var(--primary)") {
		t.Error("style.css: checked state must set background-color: var(--primary)")
	}
	// White SVG checkmark embedded as a data-URI background-image.
	if !strings.Contains(body, "data:image/svg+xml") {
		t.Error("style.css: checked span::before must embed an SVG checkmark via background-image data-URI")
	}
	// Keyboard focus ring via :focus-visible.
	if !strings.Contains(body, "focus-visible + span::before") {
		t.Error("style.css: :focus-visible + span::before rule must be declared to show the keyboard focus ring on the custom box")
	}
	if !strings.Contains(body, "var(--focus-ring)") {
		t.Error("style.css: focus-visible rule must use var(--focus-ring) for the box-shadow so focus colour matches the global token")
	}
}

// TestStaticCSS_RouteTable verifies that style.css contains the CSS rules
// needed to style the route-trace hop table and distinguish timed-out hops.
func TestStaticCSS_RouteTable(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// The route-table modifier class must be present.
	if !strings.Contains(body, ".route-table") {
		t.Error("style.css: .route-table modifier class must exist for the route hop table")
	}
	// The hop-timedout rule must be present to style unresponsive hops.
	if !strings.Contains(body, ".hop-timedout") {
		t.Error("style.css: .hop-timedout rule must exist for timed-out traceroute hops")
	}
}

// TestStaticCSS_RunAnimation verifies that style.css defines the dots
// run-button animation class and its associated @keyframes.
func TestStaticCSS_RunAnimation(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// Dots animation must be defined (used as the run-button loading state).
	if !strings.Contains(body, ".anim-dots") {
		t.Error("style.css: .anim-dots animation class must be defined")
	}
	if !strings.Contains(body, "@keyframes anim-dots-bounce") {
		t.Error("style.css: @keyframes anim-dots-bounce must be declared")
	}
	// Spinner must also be present (used elsewhere in the UI).
	if !strings.Contains(body, ".spinner") {
		t.Error("style.css: .spinner class must be defined")
	}
	if !strings.Contains(body, "@keyframes spin") {
		t.Error("style.css: @keyframes spin must be declared")
	}
	// The temporary animation picker and its removed sibling animations must
	// no longer exist in the stylesheet.
	for _, removed := range []string{".anim-picker", ".anim-opt", ".anim-pulse", ".anim-wave"} {
		if strings.Contains(body, removed) {
			t.Errorf("style.css: removed animation/picker rule %q must not be present", removed)
		}
	}
}

// TestStaticCSS_AutofillTheme verifies that style.css overrides the browser
// autofill background colour so the site theme is preserved when the browser
// fills in a previously entered value for the target-host input.
func TestStaticCSS_AutofillTheme(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, ":-webkit-autofill") {
		t.Error("style.css: :-webkit-autofill rules must be present to prevent browser autofill overriding the theme background")
	}
	// The override must use a box-shadow inset trick (the only cross-browser
	// approach that defeats the UA fill colour without disabling autofill).
	if !strings.Contains(body, "inset !important") {
		t.Error("style.css: autofill override must use 'inset !important' box-shadow technique")
	}
	// Text colour must also be explicitly restored.
	if !strings.Contains(body, "-webkit-text-fill-color") {
		t.Error("style.css: autofill override must set -webkit-text-fill-color to restore text colour")
	}
}

// TestStaticCSS_ErrorBannerFlex verifies that the updated error-banner uses
// flexbox layout (with .error-icon and .error-text children) for better visual
// separation between the icon and the message text.
func TestStaticCSS_ErrorBannerFlex(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// error-banner must use flex layout for icon + text alignment.
	if !strings.Contains(body, ".error-banner") {
		t.Error("style.css: .error-banner rule must be defined")
	}
	if !strings.Contains(body, ".error-icon") {
		t.Error("style.css: .error-icon rule must be defined inside .error-banner")
	}
	if !strings.Contains(body, ".error-text") {
		t.Error("style.css: .error-text rule must be defined inside .error-banner")
	}
}

// TestStaticCSS_HiddenAttributeEnforced verifies that style.css declares a
// [hidden] reset rule with !important so that component-level display
// properties (e.g. display:flex on .error-banner) cannot override the HTML
// hidden attribute and show elements that should be invisible.
func TestStaticCSS_HiddenAttributeEnforced(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// The reset rule must use !important so it wins over component display rules.
	if !strings.Contains(body, "[hidden]") {
		t.Error("style.css: [hidden] reset rule must be declared")
	}
	if !strings.Contains(body, "display: none !important") {
		t.Error("style.css: [hidden] rule must use 'display: none !important' to override component display values")
	}
}

// TestStaticCSS_RunBtnCentering verifies that style.css correctly centres both
// the run-button resting state (▶ glyph) and its loading state (dots animation)
// by enforcing line-height:1 on #run-btn and removing the margin offset from
// .anim-dots when it is a child of #run-btn.
func TestStaticCSS_RunBtnCentering(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	// line-height:1 must be set so the inherited body line-height (1.5) does
	// not add extra leading that shifts the glyph off the vertical centre.
	if !strings.Contains(body, "line-height: 1") {
		t.Error("style.css: #run-btn must set line-height: 1 for pixel-perfect vertical centering")
	}
	// The context-specific margin reset ensures the dots animation is not
	// shifted horizontally by its default margin-right value.
	if !strings.Contains(body, "#run-btn .anim-dots") {
		t.Error("style.css: #run-btn .anim-dots override must be defined to remove the inline-context margin")
	}
	if !strings.Contains(body, "margin: 0") {
		t.Error("style.css: #run-btn .anim-dots must set margin: 0 to restore flex centering symmetry")
	}
}

// TestStaticCSS_GeoMarkerStyles verifies that style.css defines the custom
// marker dot classes used by buildMarkerIcon() via L.divIcon.
func TestStaticCSS_GeoMarkerStyles(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	for _, cls := range []string{".geo-marker--origin", ".geo-marker--target", ".geo-marker__dia-pulse-core", ".geo-marker__dia-pulse-ring"} {
		if !strings.Contains(body, cls) {
			t.Errorf("style.css: class %q not found — required for custom Leaflet divIcon styling", cls)
		}
	}
}

// TestStaticCSS_GeoLegendAndDistance verifies that style.css defines the
// .geo-legend and .geo-distance classes used by the in-map legend control and
// the distance badge below the map.
func TestStaticCSS_GeoLegendAndDistance(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	for _, cls := range []string{".geo-legend", ".geo-legend__item", ".geo-legend__marker", ".geo-distance"} {
		if !strings.Contains(body, cls) {
			t.Errorf("style.css: class %q not found — required for map legend / distance badge", cls)
		}
	}
}

// TestStaticCSS_BodyIncludesOpacityTransition verifies that the theme-fade
// opacity transition is applied to .main (not body) so that applyTheme()'s
// transitionend listener fires correctly when only the main content area fades.
// The body rule itself must NOT carry opacity, since header and footer must
// remain visible during theme switches and use their own colour transitions.
func TestStaticCSS_BodyIncludesOpacityTransition(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// .main must include an opacity transition so body.theme-transitioning .main
	// triggers a CSS transition (and thus fires transitionend on the element).
	mainIdx := strings.Index(css, ".main {")
	if mainIdx == -1 {
		t.Fatal("style.css: .main rule not found")
	}
	endIdx := strings.Index(css[mainIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: .main rule closing brace not found")
	}
	mainBlock := css[mainIdx : mainIdx+endIdx+1]
	if !strings.Contains(mainBlock, "opacity") {
		t.Error("style.css: .main transition must include 'opacity' so theme-transitioning fade works (transitionend fires on .main)")
	}
}

// TestStaticCSS_InputBaseCaretColor verifies that the base input rule (outside
// of the :-webkit-autofill override) explicitly sets caret-color so the text
// insertion cursor stays theme-coloured even in dark themes.
func TestStaticCSS_InputBaseCaretColor(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	autofillIdx := strings.Index(css, ":-webkit-autofill")
	if autofillIdx == -1 {
		t.Fatal("style.css: :-webkit-autofill rule not found")
	}
	// caret-color must appear BEFORE the autofill override so we know it is in
	// the base input rule, not only as part of the autofill emergency patch.
	beforeAutofill := css[:autofillIdx]
	if !strings.Contains(beforeAutofill, "caret-color: var(--text)") {
		t.Error("style.css: base input rule must set caret-color: var(--text) — not only in the autofill override — to keep the cursor visible in dark themes")
	}
}

// TestStaticCSS_InputBaseTextFillColor verifies that the base input rule sets
// -webkit-text-fill-color so dark-theme text remains readable when the browser
// applies autocomplete suggestion overlay styles.
func TestStaticCSS_InputBaseTextFillColor(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	autofillIdx := strings.Index(css, ":-webkit-autofill")
	if autofillIdx == -1 {
		t.Fatal("style.css: :-webkit-autofill rule not found")
	}
	beforeAutofill := css[:autofillIdx]
	if !strings.Contains(beforeAutofill, "-webkit-text-fill-color: var(--text)") {
		t.Error("style.css: base input rule must set -webkit-text-fill-color: var(--text) to prevent dark-theme text appearing black")
	}
}

// TestStaticCSS_RadiusTokenDefined verifies that --radius is defined in :root
// so all component rules that use var(--radius) resolve to a valid value.
// A missing token causes silent fallback to 'initial' (no border-radius).
func TestStaticCSS_RadiusTokenDefined(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// --radius must be assigned inside :root.
	rootStart := strings.Index(css, ":root {")
	if rootStart == -1 {
		t.Fatal("style.css: :root block not found")
	}
	rootEnd := strings.Index(css[rootStart:], "\n}")
	if rootEnd == -1 {
		t.Fatal("style.css: :root closing brace not found")
	}
	rootBlock := css[rootStart : rootStart+rootEnd]
	if !strings.Contains(rootBlock, "--radius") {
		t.Error("style.css: --radius must be defined inside :root so var(--radius) components resolve correctly")
	}
}

// TestStaticCSS_ThemeTransitioning verifies that style.css defines the
// body.theme-transitioning rule which snaps opacity to 0 for the theme fade.
func TestStaticCSS_ThemeTransitioning(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "body.theme-transitioning") {
		t.Error("style.css: body.theme-transitioning rule not found")
	}
	if !strings.Contains(body, "--theme-fade-dur") {
		t.Error("style.css: --theme-fade-dur CSS custom property not found")
	}
}

// TestStaticCSS_GeoMapFading verifies that style.css defines the
// #geo-map.geo-map--fading rule used during tile-swap fade animation.
func TestStaticCSS_GeoMapFading(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, "geo-map--fading") {
		t.Error("style.css: geo-map--fading modifier class not found")
	}
	if !strings.Contains(body, "--map-fade-dur") {
		t.Error("style.css: --map-fade-dur CSS custom property not found")
	}
}

// TestStaticCSS_MapTileBar verifies that style.css declares the .geo-map-bar
// and .map-tile-btn rules required for the tile-variant dot selector bar.
func TestStaticCSS_MapTileBar(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	for _, selector := range []string{".geo-map-bar", ".map-tile-btn", ".map-tile-btn.active"} {
		if !strings.Contains(body, selector) {
			t.Errorf("style.css: selector %q not found — map tile bar requires it", selector)
		}
	}
}

// TestStaticCSS_DarkThemeColorScheme verifies that all three dark themes
// declare color-scheme: dark so Chrome/Safari use dark-mode form-control
// rendering and do not revert focused-input text to the UA default (black).
func TestStaticCSS_DarkThemeColorScheme(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	for _, theme := range []string{"dark", "deep-blue", "forest-green"} {
		themeIdx := strings.Index(css, `[data-theme="`+theme+`"]`)
		if themeIdx == -1 {
			t.Errorf("style.css: [data-theme=%q] block not found", theme)
			continue
		}
		// Find the closing brace of the block (next '}' at column 0).
		blockEnd := strings.Index(css[themeIdx:], "\n}")
		if blockEnd == -1 {
			t.Errorf("style.css: [data-theme=%q] block closing brace not found", theme)
			continue
		}
		block := css[themeIdx : themeIdx+blockEnd]
		if !strings.Contains(block, "color-scheme: dark") {
			t.Errorf("style.css: [data-theme=%q] must declare `color-scheme: dark` to fix dark-theme input text color", theme)
		}
	}
}

// TestStaticCSS_RootColorSchemeLight verifies that :root declares
// color-scheme: light as the baseline so light themes' form controls default
// to light-mode UA rendering.
func TestStaticCSS_RootColorSchemeLight(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	rootStart := strings.Index(css, ":root {")
	if rootStart == -1 {
		t.Fatal("style.css: :root block not found")
	}
	rootEnd := strings.Index(css[rootStart:], "\n}")
	if rootEnd == -1 {
		t.Fatal("style.css: :root closing brace not found")
	}
	rootBlock := css[rootStart : rootStart+rootEnd]
	if !strings.Contains(rootBlock, "color-scheme: light") {
		t.Error("style.css: :root must declare `color-scheme: light` as the default for light themes")
	}
}

// TestStaticCSS_MapTileBarOverlay verifies that .geo-map-bar uses
// position: absolute so it overlays the map instead of sitting above it.
func TestStaticCSS_MapTileBarOverlay(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	barIdx := strings.Index(css, ".geo-map-bar {")
	if barIdx == -1 {
		t.Fatal("style.css: .geo-map-bar rule not found")
	}
	blockEnd := strings.Index(css[barIdx:], "\n}")
	if blockEnd == -1 {
		t.Fatal("style.css: .geo-map-bar closing brace not found")
	}
	block := css[barIdx : barIdx+blockEnd]
	if !strings.Contains(block, "position: absolute") {
		t.Error("style.css: .geo-map-bar must use position:absolute to overlay the map")
	}
}

// TestStaticCSS_GeoMapOuterRelative verifies that .geo-map-outer has
// position: relative, providing the positioning context for .geo-map-bar.
func TestStaticCSS_GeoMapOuterRelative(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	outerIdx := strings.Index(css, ".geo-map-outer {")
	if outerIdx == -1 {
		t.Fatal("style.css: .geo-map-outer rule not found")
	}
	blockEnd := strings.Index(css[outerIdx:], "\n}")
	if blockEnd == -1 {
		t.Fatal("style.css: .geo-map-outer closing brace not found")
	}
	block := css[outerIdx : outerIdx+blockEnd]
	if !strings.Contains(block, "position: relative") {
		t.Error("style.css: .geo-map-outer must have position:relative to contain absolute .geo-map-bar")
	}
}

// TestStaticCSS_MapTileBtnCircle verifies that .map-tile-btn is styled as a
// circle (border-radius: 50%) matching the .theme-btn visual language.
func TestStaticCSS_MapTileBtnCircle(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	btnIdx := strings.Index(css, ".map-tile-btn {")
	if btnIdx == -1 {
		t.Fatal("style.css: .map-tile-btn rule not found")
	}
	blockEnd := strings.Index(css[btnIdx:], "\n}")
	if blockEnd == -1 {
		t.Fatal("style.css: .map-tile-btn closing brace not found")
	}
	block := css[btnIdx : btnIdx+blockEnd]
	if !strings.Contains(block, "border-radius: 50%") {
		t.Error("style.css: .map-tile-btn must use border-radius:50% (circle) to match .theme-btn style")
	}
}

// TestStaticCSS_MapTileBtnVariantColors verifies that per-variant colour swatches
// are declared for all three tile variants (light, osm, dark).
func TestStaticCSS_MapTileBtnVariantColors(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	for _, variant := range []string{"light", "osm", "dark"} {
		selector := `.map-tile-btn[data-tile-variant="` + variant + `"]`
		if !strings.Contains(css, selector) {
			t.Errorf("style.css: per-variant swatch rule %q not found", selector)
		}
	}
}

// TestStaticCSS_GeoMapIsolation verifies that #geo-map has isolation: isolate so
// that Leaflet's internal pane z-indices (200, 400…) are contained within the
// map's own stacking context and cannot bleed into .geo-map-outer, where the
// .geo-map-bar overlay sits at z-index: 10.  Without this, Leaflet's tile pane
// (z-index 200) would render above the dot-button overlay.
func TestStaticCSS_GeoMapIsolation(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	geoMapIdx := strings.Index(css, "#geo-map {")
	if geoMapIdx == -1 {
		t.Fatal("style.css: #geo-map rule not found")
	}
	endIdx := strings.Index(css[geoMapIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: #geo-map rule closing brace not found")
	}
	geoMapBlock := css[geoMapIdx : geoMapIdx+endIdx+1]
	if !strings.Contains(geoMapBlock, "isolation") {
		t.Error("style.css: #geo-map must have isolation: isolate to contain Leaflet's internal z-indices")
	}
	if !strings.Contains(geoMapBlock, "isolate") {
		t.Error("style.css: #geo-map isolation must be set to 'isolate'")
	}
}

// TestStaticCSS_HeaderHasColorTransition verifies that .site-header explicitly
// defines CSS transitions for background and color so the chrome strip smoothly
// cross-fades between theme palettes without ever disappearing (no opacity fade).
func TestStaticCSS_HeaderHasColorTransition(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	headerIdx := strings.Index(css, ".site-header  {")
	if headerIdx == -1 {
		t.Fatal("style.css: .site-header rule not found")
	}
	endIdx := strings.Index(css[headerIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: .site-header rule closing brace not found")
	}
	headerBlock := css[headerIdx : headerIdx+endIdx+1]
	if !strings.Contains(headerBlock, "transition") {
		t.Error("style.css: .site-header must have a transition property for smooth theme colour changes")
	}
	if !strings.Contains(headerBlock, "background") {
		t.Error("style.css: .site-header transition must include background")
	}
}

// TestStaticCSS_FooterHasColorTransition verifies that .site-footer explicitly
// defines CSS transitions for background and color, mirroring .site-header,
// so both chrome strips transition in visual unison on every theme change.
func TestStaticCSS_FooterHasColorTransition(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	footerIdx := strings.Index(css, ".site-footer  {")
	if footerIdx == -1 {
		t.Fatal("style.css: .site-footer rule not found")
	}
	endIdx := strings.Index(css[footerIdx:], "}")
	if endIdx == -1 {
		t.Fatal("style.css: .site-footer rule closing brace not found")
	}
	footerBlock := css[footerIdx : footerIdx+endIdx+1]
	if !strings.Contains(footerBlock, "transition") {
		t.Error("style.css: .site-footer must have a transition property for smooth theme colour changes")
	}
	if !strings.Contains(footerBlock, "background") {
		t.Error("style.css: .site-footer transition must include background")
	}
}

// TestStaticCSS_ThemeTransitioningMainOpacity verifies that
// body.theme-transitioning targets .main with opacity: 0 so only the main
// content area fades out during a theme switch (header and footer stay visible).
func TestStaticCSS_ThemeTransitioningMainOpacity(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// The rule that should appear is:  body.theme-transitioning .main { opacity: 0; }
	if !strings.Contains(css, "body.theme-transitioning .main") {
		t.Error("style.css: expected 'body.theme-transitioning .main' selector — only .main must fade, not the whole body")
	}
	ttIdx := strings.Index(css, "body.theme-transitioning .main")
	if ttIdx == -1 {
		return
	}
	endIdx := strings.Index(css[ttIdx:], "}")
	if endIdx != -1 {
		block := css[ttIdx : ttIdx+endIdx+1]
		if !strings.Contains(block, "opacity") {
			t.Error("style.css: body.theme-transitioning .main must set opacity (to 0) for the fade-out effect")
		}
	}
}

// TestStaticCSS_MarkerStyleSnippets verifies that style.css contains CSS rules
// for the diamond-pulse marker shape.
func TestStaticCSS_MarkerStyleSnippets(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// Diamond-pulse marker element classes must be present.
	for _, cls := range []string{
		".geo-marker__dia-pulse-core",
		".geo-marker__dia-pulse-ring",
	} {
		if !strings.Contains(css, cls) {
			t.Errorf("style.css: class %q not found — required for diamond-pulse marker style", cls)
		}
	}
}

// TestStaticCSS_PulseAnimation verifies that style.css declares the
// @keyframes geo-dia-pulse animation used by the diamond-pulse marker style.
func TestStaticCSS_PulseAnimation(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	if !strings.Contains(css, "@keyframes geo-dia-pulse") {
		t.Error("style.css: @keyframes geo-dia-pulse must be declared for the diamond-pulse marker animation")
	}
}

// TestStaticCSS_MarkerStyleTokensRoot verifies that style.css declares the
// marker design tokens inside :root.  The chrome tokens (--marker-border,
// --marker-inner, --marker-shadow) drive secondary styling for all diamond
// variants.  The role-colour tokens (--mc-origin, --mc-target) are the
// default values for the colour scheme and are overwritten at runtime by
// applyMarkerColorScheme().
func TestStaticCSS_MarkerStyleTokensRoot(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	for _, token := range []string{
		"--marker-border", "--marker-inner", "--marker-shadow",
		"--mc-origin", "--mc-target",
	} {
		if !strings.Contains(css, token) {
			t.Errorf("style.css: CSS custom property %q not declared — required for theme-adaptive marker chrome", token)
		}
	}

	// Tokens must be declared inside :root (must appear before the first
	// standalone [data-theme="..."] { block so they apply without any active theme).
	// We search for "\n[data-theme=" to skip comment text and .theme-btn selectors.
	rootEnd := strings.Index(css, "\n[data-theme=")
	if rootEnd == -1 {
		rootEnd = len(css)
	}
	rootBlock := css[:rootEnd]
	for _, token := range []string{"--marker-border", "--marker-inner", "--mc-origin", "--mc-target"} {
		if !strings.Contains(rootBlock, token) {
			t.Errorf("style.css: %s must be declared inside :root (before any [data-theme] block)", token)
		}
	}
}

// TestStaticCSS_MarkerStyleTokensDarkThemes verifies that the three dark
// themes (deep-blue, forest-green, dark) each override the marker design tokens
// so diamond marker chrome shifts from light to dark chrome automatically.
func TestStaticCSS_MarkerStyleTokensDarkThemes(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	// --marker-border must appear ≥4 times: once in :root + once per dark theme.
	borderCount := strings.Count(css, "--marker-border:")
	if borderCount < 4 {
		t.Errorf("style.css: --marker-border declared %d time(s), want ≥4 (root + deep-blue + forest-green + dark)", borderCount)
	}

	for _, themeAttr := range []string{`deep-blue`, `forest-green`, `dark`} {
		// Search for the standalone block selector "\n[data-theme=\"...\"] {"
		// to avoid matching .theme-btn[data-theme="..."] button selectors.
		themeBlock := "\n[data-theme=\"" + themeAttr + "\"] {"
		themeIdx := strings.Index(css, themeBlock)
		if themeIdx == -1 {
			t.Errorf("style.css: theme block %s not found", themeBlock)
			continue
		}
		// Bound the search to the block by looking for the next standalone theme or end.
		rest := css[themeIdx+1:]
		nextTheme := strings.Index(rest, "\n[data-theme=")
		var themeSection string
		if nextTheme != -1 {
			themeSection = css[themeIdx : themeIdx+1+nextTheme]
		} else {
			themeSection = css[themeIdx:]
		}
		if !strings.Contains(themeSection, "--marker-border") {
			t.Errorf("style.css: [data-theme=%q] must override --marker-border for dark-mode marker chrome", themeAttr)
		}
	}
}

// TestStaticCSS_McColorTokensInMarkerRules verifies that no hardcoded #22a55b /
// #e03c3c hex colours remain outside :root — all role-colour references in
// component rules must use var(--mc-origin) / var(--mc-target).
func TestStaticCSS_McColorTokensInMarkerRules(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	if !strings.Contains(css, "var(--mc-origin)") {
		t.Error("style.css: var(--mc-origin) not found — diamond marker rules must use CSS token for role colour")
	}
	if !strings.Contains(css, "var(--mc-target)") {
		t.Error("style.css: var(--mc-target) not found — diamond marker rules must use CSS token for role colour")
	}

	// Hardcoded origin/target hex values must not appear outside the :root block.
	rootEnd := strings.Index(css, "\n[data-theme=")
	if rootEnd == -1 {
		rootEnd = len(css)
	}
	postRoot := css[rootEnd:]
	if strings.Contains(postRoot, "#22a55b") {
		t.Error("style.css: hardcoded #22a55b found outside :root — use var(--mc-origin) instead")
	}
	if strings.Contains(postRoot, "#e03c3c") {
		t.Error("style.css: hardcoded #e03c3c found outside :root — use var(--mc-target) instead")
	}
}

// TestStaticCSS_ConnectorArrowIcon verifies that style.css defines the CSS
// class for the arrow divIcon markers used on the map.
func TestStaticCSS_ConnectorArrowIcon(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /style.css: want 200, got %d", rec.Code)
	}
	css := rec.Body.String()

	if !strings.Contains(css, ".connector-arrow-icon") {
		t.Error("style.css: selector .connector-arrow-icon not found — required for arrow divIcon markers")
	}
	for _, removed := range []string{".geo-connector-bar", ".connector-style-btn"} {
		if strings.Contains(css, removed) {
			t.Errorf("style.css: selector %q must be removed — style picker no longer exists", removed)
		}
	}
}
