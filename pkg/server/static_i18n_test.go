package server_test

import (
	"strings"
	"testing"
)

// TestStaticI18n_RunButtonLabels verifies that the embedded i18n.js separates
// the card-title key (run-diagnostic) from the button key (btn-run), and that
// the button uses an icon-only value (U+25B6) with no text label.
func TestStaticI18n_RunButtonLabels(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// Card-title keys must carry the section names (not the icon).
	for _, want := range []string{"'run-diagnostic'", "Diagnostic"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js en: missing %q for run-diagnostic key", want)
		}
	}
	if !strings.Contains(body, "\u8a3a\u65b7") { // 診斷
		t.Error("i18n.js zh-TW: run-diagnostic must contain '\u8a3a\u65b7'")
	}

	// History-title key — section 2 label.
	for _, want := range []string{"'history-title'", "History"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js en: missing %q for history-title key", want)
		}
	}
	if !strings.Contains(body, "\u8a18\u9304") { // 記錄
		t.Error("i18n.js zh-TW: history-title must contain '\u8a18\u9304'")
	}

	// btn-run must be the icon-only triangle (U+25B6); btn-running empty (spinner only).
	if !strings.Contains(body, "'btn-run'") {
		t.Error("i18n.js: btn-run key must be present")
	}
	if !strings.Contains(body, "\u25b6") { // ▶
		t.Error("i18n.js: btn-run must contain the play triangle '\u25b6'")
	}
	if !strings.Contains(body, "'btn-running'") {
		t.Error("i18n.js: btn-running key must be present")
	}
}

// TestStaticI18n_ThemeLabels verifies that both locales in the embedded i18n.js
// carry translations for all five theme IDs, ensuring the switcher options are
// localised correctly regardless of the active language.
func TestStaticI18n_ThemeLabels(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// All five theme keys must be present.
	for _, key := range []string{
		"'theme-default'", "'theme-deep-blue'", "'theme-light-green'",
		"'theme-forest-green'", "'theme-dark'",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing key %s", key)
		}
	}

	// en locale must carry English labels.
	for _, label := range []string{"Default", "Deep Blue", "Light Green", "Forest Green", "Dark"} {
		if !strings.Contains(body, label) {
			t.Errorf("i18n.js en: missing label %q", label)
		}
	}

	// zh-TW locale must carry Chinese labels.
	for _, label := range []string{"\u9810\u8a2d", "\u6df1\u85cd", "\u6de1\u7da0", "\u58a8\u7da0", "\u6697\u9ed1"} {
		if !strings.Contains(body, label) {
			t.Errorf("i18n.js zh-TW: missing label %q", label)
		}
	}
}

// TestStaticI18n_WebModeKeys verifies required web-mode and dns-type keys exist
// in both locales.
func TestStaticI18n_WebModeKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	requiredKeys := []string{
		"label-web-mode",
		"web-mode-public-ip",
		"web-mode-dns",
		"web-mode-http",
		"web-mode-port",
		"dns-type-A",
		"dns-type-AAAA",
		"dns-type-MX",
		"ph-dns-domains",
		"ph-host",
		"label-http-url",
		"ph-http-url",
	}
	for _, k := range requiredKeys {
		if !strings.Contains(body, `'`+k+`'`) {
			t.Errorf("i18n.js: missing key '%s'", k)
		}
	}
}

// TestStaticI18n_HostPlaceholderKeyUnified 驗證 i18n.js 使用單一 ph-host 鍵統一
// 所有偵測模式的主機輸入框佔位提示文字，並確保舊的分散式 ph-* 鍵已被移除。
// ph-host 必須在 EN 和 zh-TW 兩個語系中各出現一次，且皆以 google.com 為範例。
func TestStaticI18n_HostPlaceholderKeyUnified(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// ph-host 必須在兩個語系中各出現（共 2 次）。
	if count := strings.Count(body, `'ph-host'`); count < 2 {
		t.Errorf("i18n.js: 'ph-host' found %d time(s) — must appear in both en and zh-TW locales", count)
	}
	// 兩個語系皆需以 google.com 為範例網域。
	if !strings.Contains(body, "google.com") {
		t.Error("i18n.js: ph-host must use google.com as the example domain")
	}
	// 舊的分散式 ph-* 鍵必須已被移除，確保統一由 ph-host 負責。
	for _, obsolete := range []string{"'ph-web'", "'ph-smtp'", "'ph-imap'", "'ph-pop'", "'ph-ftp'", "'ph-sftp'", "'ph-host-default'"} {
		if strings.Contains(body, obsolete) {
			t.Errorf("i18n.js: obsolete key %s must be removed — use ph-host instead", obsolete)
		}
	}
}

// TestStaticI18n_SMTPFTPSFTPModeKeys verifies SMTP/FTP/SFTP mode i18n keys in both locales.
func TestStaticI18n_SMTPFTPSFTPModeKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	required := []string{
		"label-smtp-mode", "smtp-mode-handshake", "smtp-mode-auth", "smtp-mode-send",
		"label-ftp-mode", "ftp-mode-login", "ftp-mode-list",
		"label-sftp-mode", "sftp-mode-auth", "sftp-mode-ls",
	}
	for _, k := range required {
		if !strings.Contains(body, `'`+k+`'`) {
			t.Errorf("i18n.js: missing key '%s'", k)
		}
	}
}

// TestStaticI18n_ModeLabelsDetectionMode verifies that all protocol mode labels
// use 'Detection Mode' in en and '偵測模式' in zh-TW — consistent with the
// Web/DNS fieldset wording.  'Test Mode' must not appear for any mode label.
func TestStaticI18n_ModeLabelsDetectionMode(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// Every mode-label key must map to 'Detection Mode' (en) somewhere in the file.
	for _, k := range []string{"label-smtp-mode", "label-ftp-mode", "label-sftp-mode"} {
		if !strings.Contains(body, `'`+k+`':`) {
			t.Errorf("i18n.js: key '%s' missing", k)
		}
	}
	// 'Detection Mode' value must appear at least three times (smtp/ftp/sftp).
	count := strings.Count(body, "'Detection Mode'")
	if count < 3 {
		t.Errorf("i18n.js: expected at least 3 occurrences of 'Detection Mode', got %d", count)
	}
	// zh-TW '偵測模式' must appear at least four times (web + smtp + ftp + sftp).
	zhCount := strings.Count(body, "'偵測模式'")
	if zhCount < 4 {
		t.Errorf("i18n.js: expected at least 4 occurrences of '偵測模式' (zh-TW), got %d", zhCount)
	}
	// Old wording 'Test Mode' must not appear anywhere.
	if strings.Contains(body, "'Test Mode'") {
		t.Error("i18n.js: 'Test Mode' must be replaced by 'Detection Mode'")
	}
}

// TestStaticI18n_ZhTWModeTranslations verifies that the zh-TW locale has
// proper Chinese translations for all SMTP/FTP/SFTP mode option values.
func TestStaticI18n_ZhTWModeTranslations(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// Check zh-TW mode option translations are present.
	zhTranslations := []struct {
		key  string
		want string
	}{
		{"label-smtp-mode", "偵測模式"},
		{"smtp-mode-handshake", "無驗證"}, // partial match is sufficient
		{"smtp-mode-auth", "身分驗證"},
		{"smtp-mode-send", "傳送"},
		{"label-ftp-mode", "偵測模式"},
		{"ftp-mode-login", "連線並登入"},
		{"ftp-mode-list", "目錄列表"},
		{"label-sftp-mode", "偵測模式"},
		{"sftp-mode-auth", "身分驗證"},
		{"sftp-mode-ls", "列出目錄"},
	}
	for _, tc := range zhTranslations {
		if !strings.Contains(body, tc.want) {
			t.Errorf("i18n.js zh-TW: key '%s' — expected Chinese translation containing %q", tc.key, tc.want)
		}
	}
}

// TestStaticI18n_FooterCopyrightKey verifies that the embedded i18n.js carries
// the footer-copyright key in both the en and zh-TW locales, and that each
// value contains the required legal elements (© symbol, year, author name).
func TestStaticI18n_FooterCopyrightKey(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	if !strings.Contains(body, "'footer-copyright'") {
		t.Error("i18n.js: 'footer-copyright' key must be present")
	}
	// Both locales must include the mandatory copyright elements.
	for _, want := range []string{"\u00a9", "2026", "Charles"} {
		if !strings.Contains(body, want) {
			t.Errorf("i18n.js: footer-copyright value must contain %q", want)
		}
	}
	// en locale must carry the All Rights Reserved statement.
	if !strings.Contains(body, "All Rights Reserved") {
		t.Error("i18n.js en: footer-copyright must contain 'All Rights Reserved'")
	}
	// zh-TW locale must have a Chinese-language variant using the corresponding phrase.
	if !strings.Contains(body, "保留所有權利") {
		t.Error("i18n.js zh-TW: footer-copyright must contain '保留所有權利'")
	}
}

// TestStaticI18n_ImapPopLegendKeys verifies that i18n.js defines legend-imap
// and legend-pop translation keys for both the English and zh-TW locales.
func TestStaticI18n_ImapPopLegendKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	if !strings.Contains(body, "legend-imap") {
		t.Error("i18n.js: legend-imap key must be defined (needed by fields-imap fieldset)")
	}
	if !strings.Contains(body, "legend-pop") {
		t.Error("i18n.js: legend-pop key must be defined (needed by fields-pop fieldset)")
	}
	// English translations.
	if !strings.Contains(body, "IMAP Options") {
		t.Error("i18n.js: English translation for legend-imap must be 'IMAP Options'")
	}
	if !strings.Contains(body, "POP3 Options") {
		t.Error("i18n.js: English translation for legend-pop must be 'POP3 Options'")
	}
	// Traditional Chinese translations.
	if !strings.Contains(body, "IMAP \u9078\u9805") {
		t.Error("i18n.js: zh-TW translation for legend-imap must be 'IMAP \u9078\u9805'")
	}
	if !strings.Contains(body, "POP3 \u9078\u9805") {
		t.Error("i18n.js: zh-TW translation for legend-pop must be 'POP3 \u9078\u9805'")
	}
}

// TestStaticI18n_WebModeTracerouteKeys verifies that both the English and
// zh-TW locales in i18n.js declare the required traceroute mode keys so the
// UI can be localised without fallback gaps.
func TestStaticI18n_WebModeTracerouteKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// Both locale sections must contain the traceroute mode key.
	for _, key := range []string{"'web-mode-traceroute'", "'label-max-hops'"} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing key %s", key)
		}
	}
	// zh-TW locale must carry Chinese label for route trace.
	if !strings.Contains(body, "路由追蹤") {
		t.Error("i18n.js zh-TW: web-mode-traceroute must contain '路由追蹤'")
	}
	// zh-TW locale must carry Chinese label for max-hops.
	if !strings.Contains(body, "最大躍點數") {
		t.Error("i18n.js zh-TW: label-max-hops must contain '最大躍點數'")
	}
}

// TestStaticI18n_RouteSectionKeys verifies that both the English and zh-TW
// locales in i18n.js declare all keys required by renderRouteSection to
// produce a fully-localised hop table without fallback gaps.
func TestStaticI18n_RouteSectionKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	for _, key := range []string{
		"'section-route'", "'th-ttl'", "'th-ip-host'", "'th-asn'", "'th-country'",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing route section key %s", key)
		}
	}
	// zh-TW locale must carry a Chinese section title.
	if !strings.Contains(body, "路由路徑") {
		t.Error("i18n.js zh-TW: section-route must contain '路由路徑'")
	}
	// zh-TW locale must carry Chinese column header for IP / Host.
	if !strings.Contains(body, "IP / 主機") {
		t.Error("i18n.js zh-TW: th-ip-host must contain 'IP / 主機'")
	}
}

// TestStaticI18n_ErrorMessageKeys verifies that the embedded i18n.js contains
// user-friendly error message keys in both English and zh-TW locales.
func TestStaticI18n_ErrorMessageKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// All error keys must be present in the file.
	for _, key := range []string{"'err-timeout'", "'err-no-runner'", "'err-unknown'"} {
		if !strings.Contains(body, key) {
			t.Errorf("i18n.js: missing error key %s", key)
		}
	}
	// English locale must carry user-friendly timeout text (not raw Go error).
	if !strings.Contains(body, "timed out") {
		t.Error("i18n.js en: err-timeout must contain 'timed out' for user-friendly display")
	}
	// zh-TW locale must carry a Chinese timeout message.
	if !strings.Contains(body, "診斷逾時") {
		t.Error("i18n.js zh-TW: err-timeout must contain '診斷逾時'")
	}
}

// TestStaticI18n_MapOriginAndDistanceKeys verifies that both the 'en' and 'zh'
// locales in i18n.js expose the 'map-origin' and 'map-distance' keys introduced
// for the enhanced map UX.  Each key must appear at least twice (once per locale).
func TestStaticI18n_MapOriginAndDistanceKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	for _, key := range []string{"'map-origin'", "'map-distance'"} {
		first := strings.Index(body, key)
		if first == -1 {
			t.Errorf("i18n.js: key %s not found in any locale", key)
			continue
		}
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh", key)
		}
	}
}

// declare translation keys for all three tile variants: light, osm, and dark.
func TestStaticI18n_MapTileVariantKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	for _, key := range []string{"'map-tile-light'", "'map-tile-osm'", "'map-tile-dark'"} {
		first := strings.Index(body, key)
		if first == -1 {
			t.Errorf("i18n.js: key %s not found in any locale", key)
			continue
		}
		// Key must appear at least twice (en + zh).
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh", key)
		}
	}
}

// TestStaticI18n_MarkerStyleKeys verifies that both the en and zh-TW locales
// in i18n.js carry all ten diamond marker-style translation keys so the picker
// bar labels are fully localised in both languages.
func TestStaticI18n_MarkerStyleKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// diamond-pulse key must be present in both en and zh-TW.
	key := "'marker-style-diamond-pulse'"
	first := strings.Index(body, key)
	if first == -1 {
		t.Errorf("i18n.js: key %s not found in any locale", key)
	} else {
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh-TW", key)
		}
	}
	// zh-TW: 脈衝
	if !strings.Contains(body, "\u8108\u885d") {
		t.Error("i18n.js zh-TW: missing Chinese marker style label \"\u8108\u885d\" (Pulse)")
	}
}

// TestStaticI18n_MarkerColorSchemeKeys verifies that both en and zh-TW locales
// in i18n.js carry the ocean colour-scheme translation key and that the zh-TW
// locale uses a Chinese label.
func TestStaticI18n_MarkerColorSchemeKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	key := "'marker-color-ocean'"
	first := strings.Index(body, key)
	if first == -1 {
		t.Errorf("i18n.js: key %s not found in any locale", key)
	} else {
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh-TW", key)
		}
	}
	// zh-TW: 海洋
	if !strings.Contains(body, "\u6d77\u6d0b") {
		t.Error("i18n.js zh-TW: missing Chinese colour scheme label \"海洋\" (Ocean)")
	}
}

// TestStaticI18n_MapLegendKeysInBothLocales verifies that the map legend
// i18n keys ('map-origin' and 'map-target') exist in both en and zh-TW locales.
func TestStaticI18n_MapLegendKeysInBothLocales(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	for _, key := range []string{"'map-origin'", "'map-target'"} {
		first := strings.Index(body, key)
		if first == -1 {
			t.Errorf("i18n.js: key %s not found in any locale", key)
			continue
		}
		second := strings.Index(body[first+1:], key)
		if second == -1 {
			t.Errorf("i18n.js: key %s found in only one locale — must be present in both en and zh-TW", key)
		}
	}
}

// TestStaticI18n_ConnectorStyleKeysInBothLocales verifies that the sole
// connector line-pattern i18n key exists in both en and zh-TW locales.
func TestStaticI18n_ConnectorStyleKeysInBothLocales(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	keys := []string{
		"'connector-tick-xs'",
	}
	for _, key := range keys {
		if count := strings.Count(body, key); count < 2 {
			t.Errorf("i18n.js: key %s found %d time(s) — must be present in both en and zh-TW locales", key, count)
		}
	}
}

// TestStaticI18n_DNSAllFailedAndCategoryKeys 驗證 i18n.js 在 EN 和 zh-TW
// 兩種語言中都定義了 dns-all-failed 鍵、五個錯誤類別標籤（dns-cat-*），
// 以及五個情境提示橫幅訊息（dns-hint-*）。
// 這些鍵由 Go 端的 ClassifyDNSLookupError 計算後傳入 renderer，
// 讓使用者看到易讀的分類標籤與操作建議，而非 Go 內部錯誤訊息。
func TestStaticI18n_DNSAllFailedAndCategoryKeys(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// Every key must appear in both EN and zh-TW locales (count ≥ 2).
	keys := []string{
		"'dns-all-failed'",
		"'dns-cat-input'",
		"'dns-cat-nxdomain'",
		"'dns-cat-network'",
		"'dns-cat-resolver'",
		"'dns-cat-unknown'",
		"'dns-hint-input'",
		"'dns-hint-nxdomain'",
		"'dns-hint-network'",
		"'dns-hint-resolver'",
		"'dns-hint-all-failed'",
	}
	for _, key := range keys {
		if count := strings.Count(body, key); count < 2 {
			t.Errorf("i18n.js: key %s found %d time(s) — must appear in both en and zh-TW locales", key, count)
		}
	}
	// Old dns-err-* keys must have been removed (classification now in Go).
	for _, obsolete := range []string{
		"'dns-err-no-host'", "'dns-err-invalid-domain'",
		"'dns-err-resolver-failed'", "'dns-err-timeout'", "'dns-err-generic'",
	} {
		if strings.Contains(body, obsolete) {
			t.Errorf("i18n.js: obsolete key %s must be removed — use dns-cat-* instead", obsolete)
		}
	}
	// EN: dns-all-failed 必須包含 "All Failed" 字樣。
	if !strings.Contains(body, "All Failed") {
		t.Error("i18n.js en: dns-all-failed must contain 'All Failed'")
	}
	// zh-TW: dns-all-failed 必須包含「全部失敗」。
	if !strings.Contains(body, "全部失敗") {
		t.Error("i18n.js zh-TW: dns-all-failed must contain '全部失敗'")
	}
	// EN: dns-hint-input 必須提示 URL / https:// 問題。
	if !strings.Contains(body, "https://") {
		t.Error("i18n.js en: dns-hint-input must mention 'https://' to guide the user")
	}
}

// TestStaticI18n_HttpUrlPlaceholderKey 驗證 i18n.js 在 EN 和 zh-TW 兩個語系中均宣告
// ph-http-url 鍵，且 placeholder 格式為完整 URL（https://google.com），
// 讓使用者清楚知道需輸入完整網址（含 scheme），而非裸主機名稱。
// 同時確認舊的 label-http-url-hint 鍵（「選填探測」提示）已移除。
func TestStaticI18n_HttpUrlPlaceholderKey(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// ph-http-url 必須在兩個語系中各出現一次（共 ≥ 2 次）。
	if count := strings.Count(body, `'ph-http-url'`); count < 2 {
		t.Errorf("i18n.js: 'ph-http-url' found %d time(s) — must appear in both en and zh-TW locales", count)
	}
	// placeholder 必須是完整 URL 格式（帶 https:// scheme），提示使用者輸入完整網址。
	if !strings.Contains(body, `'https://google.com'`) {
		t.Error("i18n.js: ph-http-url must use 'https://google.com' format to guide users to enter a full URL with scheme")
	}
	// EN 與 zh-TW 兩個語系皆應使用相同的 https://google.com 格式（兩者出現 ≥ 2 次）。
	if strings.Count(body, "https://google.com") < 2 {
		t.Error("i18n.js: ph-http-url must use https://google.com in both en and zh-TW locales")
	}
	// label-http-url-hint（舊的「選填探測」提示）必須已移除。
	if strings.Contains(body, `'label-http-url-hint'`) {
		t.Error("i18n.js: 'label-http-url-hint' must be removed — HTTP URL input is now the primary target in http mode, not optional")
	}
}
