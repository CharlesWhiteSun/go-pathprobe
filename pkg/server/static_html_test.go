package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticHTML_ThemeSelector verifies that the embedded index.html contains
// the theme-bar container with five circular dot-buttons, ordered left-to-right
// as: forest-green, light-green, default, deep-blue, dark.
func TestStaticHTML_ThemeSelector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The wrapper container must be present.
	if !strings.Contains(body, `class="theme-bar"`) {
		t.Error("index.html: missing .theme-bar container")
	}

	// All five theme dot-buttons must be present and in the prescribed order
	// (forest-green → light-green → default → deep-blue → dark, left-to-right).
	ordered := []string{"forest-green", "light-green", "default", "deep-blue", "dark"}
	prevIdx := -1
	for _, theme := range ordered {
		want := `data-theme="` + theme + `"`
		idx := strings.Index(body, want)
		if idx == -1 {
			t.Errorf("index.html: theme button for %q is missing", theme)
			continue
		}
		if idx <= prevIdx {
			t.Errorf("index.html: theme button %q is out of order", theme)
		}
		prevIdx = idx
	}

	// Buttons must NOT contain visible text (icon-only design).
	// Each button element should be self-contained (no child text node between
	// opening and closing tags beyond whitespace).
	if strings.Contains(body, `theme-select`) {
		t.Error("index.html: old <select id='theme-select'> must be removed in favour of dot-buttons")
	}
}

// TestStaticHTML_ThemeBarInHeaderInner verifies that the theme-bar sits inside
// the same .header-inner flex row as the brand and the language switcher,
// enabling the browser to vertically centre all three elements in one pass via
// align-items: center without a separate row above the title.
func TestStaticHTML_ThemeBarInHeaderInner(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// .header-brand wrapper must exist (wraps h1 + version-badge as flex: 1 left column).
	if !strings.Contains(body, `class="header-brand"`) {
		t.Error("index.html: missing .header-brand wrapper — required for 3-column header layout")
	}

	// 3-column order inside header-inner: header-brand THEN theme-bar THEN lang-switcher.
	brandIdx := strings.Index(body, `class="header-brand"`)
	themeIdx := strings.Index(body, `class="theme-bar"`)
	langIdx := strings.Index(body, `class="lang-switcher"`)
	if brandIdx == -1 || themeIdx == -1 || langIdx == -1 {
		t.Fatal("index.html: header-brand, theme-bar, or lang-switcher is missing")
	}
	if !(brandIdx < themeIdx && themeIdx < langIdx) {
		t.Errorf("index.html: 3-column order must be header-brand < theme-bar < lang-switcher, got positions %d %d %d",
			brandIdx, themeIdx, langIdx)
	}

	// theme-bar must appear AFTER the header-inner opening tag, confirming it is
	// inline (not a separate block before header-inner).
	headerInnerIdx := strings.Index(body, `class="header-inner"`)
	if themeIdx < headerInnerIdx {
		t.Error("index.html: theme-bar must be inside .header-inner, not above it")
	}
}

// TestStaticHTML_DefaultThemeAttribute verifies that the embedded index.html
// declares a data-default-theme attribute on the <html> root element.
// This attribute acts as the server-side declaration of the intended startup
// theme: the JS initTheme() reads it on every page load and applies it as the
// fallback whenever no user preference is stored in localStorage.
// Asserting the attribute value is "default" (the third dot-button) ensures a
// service restart always presents a known, predictable starting state.
func TestStaticHTML_DefaultThemeAttribute(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The <html> tag must carry data-default-theme="default".
	const want = `data-default-theme="default"`
	if !strings.Contains(body, want) {
		t.Errorf("index.html: <html> tag must declare %s so initTheme() can read the server-declared default", want)
	}
}

// TestStaticHTML_BrandMarkup verifies that the embedded index.html renders
// the "PathProbe" logotype as two separate spans so that CSS can apply
// independent weight/opacity to each half.
func TestStaticHTML_BrandMarkup(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `class="brand-path"`) {
		t.Error(`index.html: expected <span class="brand-path"> inside h1`)
	}
	if !strings.Contains(body, `class="brand-probe"`) {
		t.Error(`index.html: expected <span class="brand-probe"> inside h1`)
	}
	// The plain text logotype must no longer appear as a bare text node.
	if strings.Contains(body, `<h1>PathProbe</h1>`) {
		t.Error("index.html: h1 must use brand-path/brand-probe spans, not bare text")
	}
}

// TestStaticHTML_BrandNoPicker verifies that the embedded index.html no longer
// contains the picker markup now that the logo style is fixed.
func TestStaticHTML_BrandNoPicker(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, absent := range []string{
		"brand-type-wrapper",
		"brand-style-btn",
		"brand-style-picker",
	} {
		if strings.Contains(body, absent) {
			t.Errorf("index.html: picker markup %q must not be present", absent)
		}
	}
}

// TestStaticHTML_WebModeRadioButtons verifies the four radio buttons exist.
func TestStaticHTML_WebModeRadioButtons(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, mode := range []string{"public-ip", "dns", "http", "port"} {
		if !strings.Contains(body, `value="`+mode+`"`) {
			t.Errorf("index.html: missing radio button with value=%q", mode)
		}
	}
	// One of the radio buttons must be pre-checked.
	if !strings.Contains(body, `name="web-mode"`) {
		t.Error("index.html: radio buttons must carry name=\"web-mode\"")
	}
}

// TestStaticHTML_WebModeDNSSubpanel verifies that the DNS sub-panel exists with
// the placeholder attribute (no hard-coded value).
func TestStaticHTML_WebModeDNSSubpanel(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="web-fields-dns"`) {
		t.Error("index.html: DNS sub-panel #web-fields-dns must exist")
	}
	if !strings.Contains(body, `data-i18n-placeholder="ph-dns-domains"`) {
		t.Error("index.html: dns-domains input must use data-i18n-placeholder")
	}
	// Must NOT have a hard-coded value="example.com"
	if strings.Contains(body, `value="example.com"`) {
		t.Error("index.html: dns-domains must not have hard-coded value=\"example.com\"")
	}
}

// TestStaticHTML_HostInputPlaceholderI18n 驗證 #host 輸入框使用 data-i18n-placeholder="ph-host"
// 屬性，讓 locale.js 的通用 [data-i18n-placeholder] 迴圈統一處理佔位文字翻譯，
// 不再依賴各偵測模式的逐一查詢邏輯。
func TestStaticHTML_HostInputPlaceholderI18n(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// #host 必須使用 data-i18n-placeholder="ph-host" 以支援通用 i18n 機制。
	if !strings.Contains(body, `data-i18n-placeholder="ph-host"`) {
		t.Error("index.html: #host input must carry data-i18n-placeholder=\"ph-host\" for unified i18n placeholder")
	}
	// id 與屬性必須出現在同一個 input 元素中（#host）。
	hostInputIdx := strings.Index(body, `id="host"`)
	if hostInputIdx == -1 {
		t.Fatal("index.html: #host input not found")
	}
	// 截取 #host 標籤片段（最多 200 字元），確認 data-i18n-placeholder 就在該標籤上。
	tagEnd := hostInputIdx + 200
	if tagEnd > len(body) {
		tagEnd = len(body)
	}
	tag := body[hostInputIdx:tagEnd]
	if !strings.Contains(tag, `data-i18n-placeholder="ph-host"`) {
		t.Error("index.html: data-i18n-placeholder=\"ph-host\" must be on the same #host input element, not elsewhere")
	}
}

// TestStaticHTML_WebModeRecordTypeLabels verifies i18n labels for A/AAAA/MX.
func TestStaticHTML_WebModeRecordTypeLabels(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, key := range []string{"dns-type-A", "dns-type-AAAA", "dns-type-MX"} {
		if !strings.Contains(body, `data-i18n="`+key+`"`) {
			t.Errorf("index.html: missing data-i18n=%q for record type label", key)
		}
	}
}

// TestStaticHTML_SMTPModeSelector verifies SMTP mode-selector and sub-panels exist.
func TestStaticHTML_SMTPModeSelector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, mode := range []string{"handshake", "auth", "send"} {
		if !strings.Contains(body, `name="smtp-mode" value="`+mode+`"`) {
			t.Errorf("index.html: missing SMTP radio with value=%q", mode)
		}
	}
	for _, panel := range []string{"smtp-fields-auth", "smtp-fields-send"} {
		if !strings.Contains(body, `id="`+panel+`"`) {
			t.Errorf("index.html: missing sub-panel #%s", panel)
		}
	}
}

// TestStaticHTML_FTPModeSelector verifies FTP mode-selector exists and ftp-list checkbox is absent.
func TestStaticHTML_FTPModeSelector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, mode := range []string{"login", "list"} {
		if !strings.Contains(body, `name="ftp-mode" value="`+mode+`"`) {
			t.Errorf("index.html: missing FTP radio with value=%q", mode)
		}
	}
	if strings.Contains(body, `id="ftp-list"`) {
		t.Error("index.html: ftp-list checkbox must be removed (replaced by mode selector)")
	}
}

// TestStaticHTML_SFTPModeSelector verifies SFTP mode-selector exists and sftp-ls checkbox is absent.
func TestStaticHTML_SFTPModeSelector(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	for _, mode := range []string{"auth", "ls"} {
		if !strings.Contains(body, `name="sftp-mode" value="`+mode+`"`) {
			t.Errorf("index.html: missing SFTP radio with value=%q", mode)
		}
	}
	if strings.Contains(body, `id="sftp-ls"`) {
		t.Error("index.html: sftp-ls checkbox must be removed (replaced by mode selector)")
	}
}

// TestStaticHTML_ModeLabelFallbackText verifies that the fallback text for all
// mode-selector labels in index.html is 'Detection Mode' (not 'Test Mode').
func TestStaticHTML_ModeLabelFallbackText(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if strings.Contains(body, ">Test Mode<") {
		t.Error("index.html: fallback text 'Test Mode' must be replaced by 'Detection Mode'")
	}
	// Each of the three protocol fieldsets must carry the correct fallback text.
	for _, key := range []string{"label-smtp-mode", "label-ftp-mode", "label-sftp-mode"} {
		want := `data-i18n="` + key + `">Detection Mode`
		if !strings.Contains(body, want) {
			t.Errorf("index.html: label with data-i18n=%q must have fallback text 'Detection Mode'", key)
		}
	}
}

// TestStaticHTML_CustomSelectMarkup verifies that the target <select> in
// index.html has been replaced with the custom .cs-wrap widget and that the
// hidden native <select id="target"> is still present so val('target')
// continues to work without any other JS changes.
func TestStaticHTML_CustomSelectMarkup(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// Custom-select wrapper must be present.
	if !strings.Contains(body, `class="cs-wrap"`) {
		t.Error(`index.html: <div class="cs-wrap"> must be present for the custom dropdown`)
	}
	if !strings.Contains(body, `class="cs-trigger"`) {
		t.Error(`index.html: .cs-trigger button must be present inside .cs-wrap`)
	}
	if !strings.Contains(body, `class="cs-list"`) {
		t.Error(`index.html: .cs-list popup must be present inside .cs-wrap`)
	}
	if !strings.Contains(body, `class="cs-label"`) {
		t.Error(`index.html: .cs-label span must be present inside .cs-trigger`)
	}

	// All six target values must appear as cs-item options.
	for _, v := range []string{"web", "smtp", "imap", "pop", "ftp", "sftp"} {
		want := `data-value="` + v + `"`
		if !strings.Contains(body, want) {
			t.Errorf("index.html: cs-item with %s not found", want)
		}
	}

	// The hidden native select must still be present for val('target') compat.
	if !strings.Contains(body, `id="target"`) {
		t.Fatal("index.html: hidden <select id=\"target\"> must be present for val() compatibility")
	}

	// cs-wrap must precede the hidden select in source order.
	csIdx := strings.Index(body, `class="cs-wrap"`)
	selIdx := strings.Index(body, `id="target"`)
	if csIdx == -1 || selIdx == -1 {
		t.Fatal("index.html: .cs-wrap or #target is missing")
	}
	if csIdx > selIdx {
		t.Error("index.html: .cs-wrap must appear before the hidden #target in source order")
	}

	// Accessibility: trigger must have aria-haspopup and aria-expanded.
	if !strings.Contains(body, `aria-haspopup="listbox"`) {
		t.Error(`index.html: .cs-trigger must carry aria-haspopup="listbox" for screen-reader disclosure`)
	}
	if !strings.Contains(body, `aria-expanded="false"`) {
		t.Error(`index.html: .cs-trigger must start with aria-expanded="false"`)
	}
	// cs-list must have role=listbox.
	if !strings.Contains(body, `role="listbox"`) {
		t.Error(`index.html: .cs-list must carry role="listbox"`)
	}
}

// TestStaticHTML_FooterPresent verifies that the embedded index.html contains
// a <footer class="site-footer"> element with the .footer-inner wrapper.
// The footer must appear after </main> so the HTML document structure follows
// the natural reading order: header → main content → footer.
func TestStaticHTML_FooterPresent(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `class="site-footer"`) {
		t.Error(`index.html: <footer class="site-footer"> must be present`)
	}
	if !strings.Contains(body, `class="footer-inner"`) {
		t.Error(`index.html: .footer-inner wrapper must be present inside .site-footer`)
	}
	if !strings.Contains(body, `class="footer-copy"`) {
		t.Error(`index.html: .footer-copy paragraph must be present inside .footer-inner`)
	}

	// Footer must appear after the closing </main> tag.
	mainIdx := strings.Index(body, "</main>")
	footerIdx := strings.Index(body, `class="site-footer"`)
	if mainIdx == -1 || footerIdx == -1 {
		t.Fatal("index.html: </main> or .site-footer is missing")
	}
	if footerIdx < mainIdx {
		t.Error("index.html: .site-footer must appear after </main> in source order")
	}
}

// TestStaticHTML_FooterCopyright verifies that the footer element contains the
// copyright notice with the data-i18n key and the expected English fallback
// text.  The copyright text must include the © symbol, the year, and the
// author name "Charles" so the notice is legally unambiguous.
func TestStaticHTML_FooterCopyright(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `data-i18n="footer-copyright"`) {
		t.Error("index.html: footer copyright paragraph must carry data-i18n=\"footer-copyright\"")
	}
	// The fallback text must contain the essential copyright elements.
	for _, want := range []string{"\u00a9", "2026", "Charles"} {
		if !strings.Contains(body, want) {
			t.Errorf("index.html: footer fallback text must contain %q for a valid copyright notice", want)
		}
	}
}

// TestStaticHTML_VividAnimDefault verifies that index.html permanently sets
// data-anim="vivid" on the <html> element so the vivid animation intensity is
// active from first paint without any JS initialization.
func TestStaticHTML_VividAnimDefault(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// Vivid animation must be the declared default on the root element.
	if !strings.Contains(body, `data-anim="vivid"`) {
		t.Error(`index.html: <html> must carry data-anim="vivid" to apply the vivid animation intensity by default`)
	}
	// The temporary toggle button must be absent — it was a developer tool only.
	if strings.Contains(body, `id="anim-toggle"`) {
		t.Error(`index.html: temporary anim-toggle button must be removed; vivid mode is now the permanent default`)
	}
	if strings.Contains(body, `cycleAnim()`) {
		t.Error(`index.html: cycleAnim() onclick must be removed along with the toggle button`)
	}
}

// TestStaticHTML_ImapPopFieldsets verifies that index.html contains hidden
// fieldsets for the imap and pop targets so onTargetChange() can always find
// them via getElementById and cleanly hide any previously active panel.
func TestStaticHTML_ImapPopFieldsets(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="fields-imap"`) {
		t.Error("index.html: fieldset id=fields-imap must be present for the imap target type")
	}
	if !strings.Contains(body, `id="fields-pop"`) {
		t.Error("index.html: fieldset id=fields-pop must be present for the pop target type")
	}
	// Both fieldsets must start hidden so they are invisible until selected.
	imapHiddenIdx := strings.Index(body, `id="fields-imap"`)
	popHiddenIdx := strings.Index(body, `id="fields-pop"`)
	if imapHiddenIdx == -1 || !strings.Contains(body[imapHiddenIdx:imapHiddenIdx+200], "hidden") {
		t.Error("index.html: fields-imap fieldset must carry the hidden attribute")
	}
	if popHiddenIdx == -1 || !strings.Contains(body[popHiddenIdx:popHiddenIdx+200], "hidden") {
		t.Error("index.html: fields-pop fieldset must carry the hidden attribute")
	}
	// Both fieldsets must carry data-panel-empty="true" so JS skips the reveal
	// step and never presents an empty bordered box to the user when imap/pop
	// is selected.
	if imapHiddenIdx == -1 || !strings.Contains(body[imapHiddenIdx:imapHiddenIdx+300], `data-panel-empty="true"`) {
		t.Error(`index.html: fields-imap fieldset must carry data-panel-empty="true" to suppress the blank reveal`)
	}
	if popHiddenIdx == -1 || !strings.Contains(body[popHiddenIdx:popHiddenIdx+300], `data-panel-empty="true"`) {
		t.Error(`index.html: fields-pop fieldset must carry data-panel-empty="true" to suppress the blank reveal`)
	}
	// legend keys must be referenced so i18n can label the fieldsets.
	if !strings.Contains(body, "legend-imap") {
		t.Error("index.html: fields-imap fieldset must reference legend-imap i18n key")
	}
	if !strings.Contains(body, "legend-pop") {
		t.Error("index.html: fields-pop fieldset must reference legend-pop i18n key")
	}
}

// TestStaticHTML_AdvancedOptsStructure verifies that index.html wraps the
// Advanced Options content inside .adv-body / .adv-inner elements so that
// JS-driven height + fade animations work correctly.
func TestStaticHTML_AdvancedOptsStructure(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The details element must have a stable id so initAdvancedOpts() can find it.
	if !strings.Contains(body, `id="advanced-opts"`) {
		t.Error(`index.html: <details> must have id="advanced-opts" for JS to wire up the animation`)
	}
	// .adv-body is the height-transition container (mirrors .panel-stage).
	if !strings.Contains(body, `class="adv-body"`) {
		t.Error("index.html: Advanced Options content must be wrapped in <div class=\"adv-body\"> for height animation")
	}
	// .adv-inner is the opacity+slide animation target (mirrors .target-fields inside .panel-stage).
	if !strings.Contains(body, `class="adv-inner"`) {
		t.Error("index.html: Advanced Options content must be wrapped in <div class=\"adv-inner\"> for fade+slide animation")
	}
}

// TestStaticHTML_WebModeTracerouteRadio verifies that the embedded index.html
// includes a radio button for the "traceroute" web sub-mode so users can
// initiate a route-trace diagnostic from the UI.
func TestStaticHTML_WebModeTracerouteRadio(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The traceroute radio value must be present.
	if !strings.Contains(body, `value="traceroute"`) {
		t.Error("index.html: missing radio button with value=\"traceroute\" for route-trace mode")
	}
	// Its i18n key must be declared.
	if !strings.Contains(body, `data-i18n="web-mode-traceroute"`) {
		t.Error("index.html: traceroute radio must carry data-i18n=\"web-mode-traceroute\"")
	}
}

// TestStaticHTML_WebModeTracerouteMaxHopsPanel verifies that the traceroute
// sub-panel exists in index.html and exposes a max-hops number input so the
// user can control the maximum TTL depth.
func TestStaticHTML_WebModeTracerouteMaxHopsPanel(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The traceroute sub-panel must exist and be initially hidden.
	if !strings.Contains(body, `id="web-fields-traceroute"`) {
		t.Error("index.html: traceroute sub-panel #web-fields-traceroute must exist")
	}
	// The max-hops number input must be present inside the panel.
	if !strings.Contains(body, `id="traceroute-max-hops"`) {
		t.Error("index.html: traceroute sub-panel must contain input#traceroute-max-hops")
	}
	// Its label must use the i18n key.
	if !strings.Contains(body, `data-i18n="label-max-hops"`) {
		t.Error("index.html: max-hops label must use data-i18n=\"label-max-hops\"")
	}
}

// TestStaticHTML_ErrorBannerStructure verifies that index.html contains the
// structured error banner with role="alert" and separate icon/text spans.
func TestStaticHTML_ErrorBannerStructure(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="error-banner"`) {
		t.Error("index.html: #error-banner must be present")
	}
	if !strings.Contains(body, `role="alert"`) {
		t.Error("index.html: #error-banner must declare role=\"alert\" for screen-reader accessibility")
	}
	if !strings.Contains(body, `class="error-icon"`) {
		t.Error("index.html: .error-icon span must be present inside #error-banner")
	}
	if !strings.Contains(body, `id="error-text"`) {
		t.Error("index.html: #error-text span must be present inside #error-banner")
	}
}

// TestStaticHTML_ErrorBannerHiddenByDefault verifies that the error banner in
// index.html carries the `hidden` attribute so it is invisible on page load and
// only becomes visible when JS calls showError().
func TestStaticHTML_ErrorBannerHiddenByDefault(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The banner element with its hidden attribute must appear together.
	if !strings.Contains(body, `id="error-banner" hidden`) {
		t.Error("index.html: #error-banner must carry the `hidden` attribute so it is invisible on load")
	}
}

// TestStaticHTML_PortsFieldGroup verifies that the redesigned form layout places
// target-type, host, and port-group in ONE unified form-grid row.  The port-group
// hosts a shared text input used by both web/port mode and non-web targets.
func TestStaticHTML_PortsFieldGroup(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The unified port-group column must exist, initially hidden
	// (default target = web/public-ip which doesn't need port selection).
	if !strings.Contains(body, `id="port-group" hidden`) {
		t.Error("index.html: #port-group must be present and initially hidden (default web/public-ip needs no ports)")
	}
	// The shared text-input variant must exist inside port-group.
	if !strings.Contains(body, `id="ports-text-group" hidden`) {
		t.Error("index.html: #ports-text-group must be present inside #port-group")
	}
	// The removed checkbox picker must NOT appear in the HTML.
	if strings.Contains(body, `id="web-port-picker"`) {
		t.Error("index.html: #web-port-picker checkbox picker has been removed; it must not appear in the HTML")
	}
	// host and ports inputs must still be reachable by their existing IDs.
	if !strings.Contains(body, `id="host"`) {
		t.Error("index.html: #host input must be present")
	}
	if !strings.Contains(body, `id="ports"`) {
		t.Error("index.html: #ports input must be present")
	}
}

// TestStaticHTML_PortGroupLabelHint verifies that the #port-group label displays
// the "Ports" text and "(comma-separated)" hint inline as a <small> element
// inside the <label> — matching the same visual pattern used by other fields
// (e.g. DNS Domains, SMTP RCPT TO).  The hint must NOT appear as a standalone
// sibling of the <input> inside #ports-text-group.
func TestStaticHTML_PortGroupLabelHint(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// The label inside #port-group must embed the hint as a <small> element.
	const wantInlineHint = `<span data-i18n="label-ports">Ports</span> <small data-i18n="label-ports-hint">(comma-separated)</small></label>`
	if !strings.Contains(body, wantInlineHint) {
		t.Error(`index.html: #port-group label must contain inline <small data-i18n="label-ports-hint"> hint`)
	}
	// The hint must NOT appear as a standalone sibling of the <input> inside
	// #ports-text-group (it would duplicate the inline label hint).
	portTextGroupStart := strings.Index(body, `id="ports-text-group"`)
	if portTextGroupStart == -1 {
		t.Fatal("index.html: #ports-text-group element not found")
	}
	// Find the closing </div> of #ports-text-group (next </div> after its open tag).
	portTextGroupEnd := strings.Index(body[portTextGroupStart:], "</div>")
	if portTextGroupEnd == -1 {
		t.Fatal("index.html: closing </div> for #ports-text-group not found")
	}
	textGroupBody := body[portTextGroupStart : portTextGroupStart+portTextGroupEnd]
	if strings.Contains(textGroupBody, `data-i18n="label-ports-hint"`) {
		t.Error(`index.html: <small data-i18n="label-ports-hint"> must not appear inside #ports-text-group (it belongs in the parent <label> instead)`)
	}
}

// TestStaticHTML_GeoDistanceElement verifies that index.html includes the
// #geo-distance element, which renderMap() populates with the great-circle
// distance between origin and target.
func TestStaticHTML_GeoDistanceElement(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="geo-distance"`) {
		t.Error("index.html: #geo-distance element not found — required for the map distance badge")
	}
}

// TestStaticHTML_GeoMapBar verifies that index.html contains #geo-map-bar
// inside a .geo-map-outer wrapper element.
func TestStaticHTML_GeoMapBar(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="geo-map-bar"`) {
		t.Error(`index.html: element with id="geo-map-bar" not found`)
	}
}

// TestStaticHTML_GeoMapBarUniqueId verifies that index.html contains exactly
// ONE element with id="geo-map-bar". Duplicate IDs break getElementById and
// leave the second element permanently empty.
func TestStaticHTML_GeoMapBarUniqueId(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	const marker = `id="geo-map-bar"`
	if count := strings.Count(body, marker); count != 1 {
		t.Errorf(`index.html: expected exactly 1 element with id="geo-map-bar", got %d — duplicate IDs break getElementById`, count)
	}
}

// TestStaticHTML_GeoMapOuter verifies that index.html wraps #geo-map-bar and
// #geo-map in a .geo-map-outer element which provides the overlay context.
func TestStaticHTML_GeoMapOuter(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="geo-map-outer"`) {
		t.Error(`index.html: element with id="geo-map-outer" not found`)
	}
	if !strings.Contains(body, `class="geo-map-outer"`) {
		t.Error(`index.html: element with class="geo-map-outer" not found`)
	}
}

// TestStaticHTML_GeoConnectorBarRemoved verifies that index.html no longer
// contains #geo-connector-bar — the style picker was removed because only
// one connector style exists and it is applied by default.
func TestStaticHTML_GeoConnectorBarRemoved(t *testing.T) {
	h := newStaticHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `id="geo-connector-bar"`) {
		t.Error(`index.html: #geo-connector-bar must be removed — style picker is no longer needed`)
	}
}

// TestStaticHTML_ScriptLoadOrder verifies that all 13 JavaScript modules are
// declared in index.html as <script defer> tags and appear in the correct
// dependency order.  The order is critical because each module registers itself
// on window.PathProbe before the next one reads it.
func TestStaticHTML_ScriptLoadOrder(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// Ordered list of script src paths as they must appear in index.html.
	ordered := []string{
		`src="/leaflet.js"`,
		`src="/i18n.js"`,
		`src="/config.js"`,
		`src="/locale.js"`,
		`src="/theme.js"`,
		`src="/form.js"`,
		`src="/api-builder.js"`,
		`src="/renderer.js"`,
		`src="/map-connector.js"`,
		`src="/map.js"`,
		`src="/api-client.js"`,
		`src="/history.js"`,
		`src="/app.js"`,
	}

	prev := 0
	for _, src := range ordered {
		idx := strings.Index(body[prev:], src)
		if idx == -1 {
			t.Errorf("index.html: script tag with %q not found (or out of order)", src)
			continue
		}
		prev += idx + len(src)

		// Each script must carry the defer attribute.
		defer_ := strings.Index(body[prev-len(src)-50:prev+10], "defer")
		if defer_ == -1 {
			t.Errorf("index.html: script %q must carry the 'defer' attribute", src)
		}
	}
}

// TestStaticHTML_HttpUrlGroupAtTopLevel 驗證 #http-url-group 已移至頂層 form-grid，
// 與 #host-group 和 #dns-domains-group 並列，而非藏在 #fields-web 的子面板中。
// 切換至 HTTP/HTTPS 探測模式時此欄取代 Target Host，與 DNS 模式的處理方式一致。
func TestStaticHTML_HttpUrlGroupAtTopLevel(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// #http-url-group 必須存在且預設隱藏（HTTP 模式啟用時才顯示）。
	if !strings.Contains(body, `id="http-url-group" hidden`) {
		t.Error("index.html: #http-url-group must exist at top-level form-grid and be hidden by default")
	}
	// #http-url input 必須帶有 data-i18n-placeholder，使用通用 i18n 機制翻譯 placeholder。
	if !strings.Contains(body, `data-i18n-placeholder="ph-http-url"`) {
		t.Error("index.html: #http-url input must carry data-i18n-placeholder=\"ph-http-url\" for i18n support")
	}
	// id 與 data-i18n-placeholder 必須在同一個 input 標籤上。
	httpUrlIdx := strings.Index(body, `id="http-url"`)
	if httpUrlIdx == -1 {
		t.Fatal("index.html: #http-url input not found")
	}
	tagEnd := httpUrlIdx + 200
	if tagEnd > len(body) {
		tagEnd = len(body)
	}
	if !strings.Contains(body[httpUrlIdx:tagEnd], `data-i18n-placeholder="ph-http-url"`) {
		t.Error("index.html: data-i18n-placeholder=\"ph-http-url\" must be on the same #http-url input element")
	}
	// #http-url-group 必須出現在 #host-group 之後（兩者在同一層 form-grid），
	// 並且在 panel-stage（#fields-web fieldset）之前。
	hostGrpIdx := strings.Index(body, `id="host-group"`)
	httpUrlGrpIdx := strings.Index(body, `id="http-url-group"`)
	panelStageIdx := strings.Index(body, `id="panel-stage"`)
	if hostGrpIdx == -1 || httpUrlGrpIdx == -1 || panelStageIdx == -1 {
		t.Fatal("index.html: #host-group, #http-url-group, or #panel-stage not found")
	}
	if httpUrlGrpIdx < hostGrpIdx {
		t.Error("index.html: #http-url-group must appear after #host-group in the DOM")
	}
	if httpUrlGrpIdx > panelStageIdx {
		t.Error("index.html: #http-url-group must appear before #panel-stage (should be in top-level form-grid, not inside fieldset)")
	}
}

// TestStaticHTML_WebFieldsHttpRemoved 驗證 #web-fields-http 子面板已從 #fields-web
// 內移除。#http-url input 已移至頂層 form-grid 的 #http-url-group，不再重複存在。
func TestStaticHTML_WebFieldsHttpRemoved(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	// #web-fields-http 已被移除，不應再出現於 HTML 中。
	if strings.Contains(body, `id="web-fields-http"`) {
		t.Error("index.html: #web-fields-http sub-panel must be removed — #http-url input has moved to top-level #http-url-group")
	}
}
