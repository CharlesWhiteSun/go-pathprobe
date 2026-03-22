package server_test

// static_js_traceroute_ui_test.go
//
// 驗證路由追蹤 UI 強化功能的靜態檔案測試：
//   1. 逾時倒數計時器（#tr-countdown）
//   2. 逾時輸入框改為分鐘數字輸入（#diag-timeout type=number）
//   3. 路由追蹤逾時後顯示統計摘要卡（#tr-stats）
//   4. 使用者取消後仍顯示統計摘要卡
//
// 每個測試函式只驗證單一關切點，以保持高隔離性與易維護性。

import (
	"strings"
	"testing"
)

// ── HTML 結構測試 ──────────────────────────────────────────────────────────

// TestStaticHTML_TracerouteCountdownElement 驗證 index.html 的
// #traceroute-progress 面板內含有進度條結構（#tr-countdown-bar、
// #tr-countdown-label），供 renderer.js 以視覺化進度條方式呈現倒數。
func TestStaticHTML_TracerouteCountdownElement(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="tr-countdown-bar"`) {
		t.Error("index.html: #tr-countdown-bar progress bar must be present inside #traceroute-progress")
	}
	if !strings.Contains(body, `id="tr-countdown-label"`) {
		t.Error("index.html: #tr-countdown-label text element must be present inside #traceroute-progress")
	}
	if !strings.Contains(body, "tr-countdown-wrap") {
		t.Error("index.html: .tr-countdown-wrap container must wrap the countdown bar and label")
	}
	if !strings.Contains(body, "tr-cbar") {
		t.Error("index.html: .tr-cbar class must be applied to the countdown bar fill element")
	}
}

// TestStaticHTML_TracerouteStatsContainer 驗證 index.html 的
// #traceroute-progress 面板內含有 #tr-stats 容器，供逾時或取消後注入
// route-stats-card 統計摘要。
func TestStaticHTML_TracerouteStatsContainer(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="tr-stats"`) {
		t.Error("index.html: #tr-stats container must be present inside #traceroute-progress for inline stats injection on timeout/cancel")
	}
	if !strings.Contains(body, "tr-stats-inpanel") {
		t.Error("index.html: #tr-stats must have class 'tr-stats-inpanel' for scoped styling within the panel")
	}
}

// TestStaticHTML_TimeoutInputIsNumber 驗證進階選項中的逾時輸入框
// 已改為 type="number" 且 min="1"，讓使用者以分鐘為單位輸入整數，
// 取代先前需要手動輸入 Go duration 字串（如 "30s"）的方式。
func TestStaticHTML_TimeoutInputIsNumber(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/")

	if !strings.Contains(body, `id="diag-timeout"`) {
		t.Fatal("index.html: #diag-timeout input must be present")
	}
	// Must be number type, not text.
	if !strings.Contains(body, `type="number"`) {
		t.Error("index.html: #diag-timeout must be type=\"number\" (minutes-based spinner, no raw duration string)")
	}
	// Must set minimum to 1 to prevent 0 or negative timeout.
	if !strings.Contains(body, `min="1"`) {
		t.Error("index.html: #diag-timeout must have min=\"1\" to prevent zero/negative timeout")
	}
}

// ── CSS 測試 ───────────────────────────────────────────────────────────────

// TestStaticCSS_TrCountdownStyle 驗證 style.css 定義了倒數進度條相關的
// CSS 規則：.tr-countdown-wrap（容器）、.tr-cbar-outer（軌道）、
// .tr-cbar（填充條）與 .tr-countdown-label（文字標籤）。
func TestStaticCSS_TrCountdownStyle(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, ".tr-countdown-wrap") {
		t.Error("style.css: .tr-countdown-wrap rule must be defined for the countdown container")
	}
	if !strings.Contains(body, ".tr-cbar-outer") {
		t.Error("style.css: .tr-cbar-outer rule must be defined for the countdown bar track")
	}
	if !strings.Contains(body, ".tr-cbar {") {
		t.Error("style.css: .tr-cbar rule must be defined for the countdown bar fill")
	}
	if !strings.Contains(body, ".tr-countdown-label") {
		t.Error("style.css: .tr-countdown-label rule must be defined for the remaining time text")
	}
}

// TestStaticCSS_TrStatsInpanel 驗證 style.css 定義了 .tr-stats-inpanel 規則，
// 提供統計摘要卡片與躍點表格之間的分隔線與間距。
func TestStaticCSS_TrStatsInpanel(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/style.css")

	if !strings.Contains(body, ".tr-stats-inpanel") {
		t.Error("style.css: .tr-stats-inpanel rule must be defined for the stats card container inside the traceroute panel")
	}
}

// ── i18n 測試 ──────────────────────────────────────────────────────────────

// TestStaticI18n_TracerouteCountdownKey 驗證 i18n.js 在 EN 與 ZH 兩個
// locale 都定義了 'traceroute-countdown' key，供 renderer.js 更新倒數文字。
func TestStaticI18n_TracerouteCountdownKey(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	count := strings.Count(body, "'traceroute-countdown'")
	if count < 2 {
		t.Errorf("i18n.js: 'traceroute-countdown' must be defined in both EN and ZH locales; found %d occurrence(s)", count)
	}
	// Must include the {t} placeholder for countdown formatting.
	if !strings.Contains(body, "{t}") {
		t.Error("i18n.js: traceroute-countdown value must contain '{t}' placeholder for the formatted time string")
	}
}

// TestStaticI18n_TimeoutLabelUpdated 驗證 i18n.js 中的 'label-timeout' key
// 在 EN locale 已包含 "min" (minutes)，在 ZH locale 已包含「分鐘」，
// 反映輸入框已從 duration 字串改為分鐘數字的語意變更。
func TestStaticI18n_TimeoutLabelUpdated(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	// EN: must reference "min".
	if !strings.Contains(body, "Timeout (min)") {
		t.Error("i18n.js: EN 'label-timeout' must reference '(min)' to indicate the input unit is minutes")
	}
	// ZH: must reference "分鐘".
	if !strings.Contains(body, "逾時（分鐘）") {
		t.Error("i18n.js: ZH 'label-timeout' must reference '分鐘' to indicate the input unit is minutes")
	}
}

// ── JS renderer.js 測試 ────────────────────────────────────────────────────

// TestStaticJS_InitTracerouteProgressCountdown 驗證 renderer.js 的
// initTracerouteProgress 函式：
//   - 接受第四個參數 timeoutSec
//   - 啟動 setInterval 倒數計時器（_trCountdownTimer）
//   - 參考 'traceroute-countdown' i18n key
//   - 呼叫 _formatCountdown 格式化時間顯示
//   - 更新 #tr-countdown-bar 的 width 呈現進度條
//   - 重置 _trCollectedHops 為空陣列
func TestStaticJS_InitTracerouteProgressCountdown(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function initTracerouteProgress(")
	if fnStart == -1 {
		t.Fatal("renderer.js: initTracerouteProgress not found")
	}
	// Locate end of function (next top-level function definition).
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
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

	if !strings.Contains(fnBody, "timeoutSec") {
		t.Error("renderer.js: initTracerouteProgress must accept timeoutSec parameter for the countdown")
	}
	if !strings.Contains(fnBody, "setInterval") {
		t.Error("renderer.js: initTracerouteProgress must start a setInterval countdown timer")
	}
	if !strings.Contains(fnBody, "traceroute-countdown") {
		t.Error("renderer.js: initTracerouteProgress must reference 'traceroute-countdown' i18n key for the countdown text")
	}
	if !strings.Contains(fnBody, "_formatCountdown") {
		t.Error("renderer.js: initTracerouteProgress must call _formatCountdown to format the time display")
	}
	if !strings.Contains(fnBody, "_trCollectedHops") {
		t.Error("renderer.js: initTracerouteProgress must reset _trCollectedHops = [] at start of each run")
	}
	// Countdown progress bar: must reference the bar element ID and update its width.
	if !strings.Contains(fnBody, "tr-countdown-bar") {
		t.Error("renderer.js: initTracerouteProgress must reference #tr-countdown-bar to display countdown as a shrinking bar")
	}
	if !strings.Contains(fnBody, "style.width") {
		t.Error("renderer.js: initTracerouteProgress must set style.width on the countdown bar to visualise progress")
	}
}

// TestStaticJS_CountdownHelpers 驗證 renderer.js 定義了倒數計時所需的
// 私有輔助函式：_stopCountdown、_formatCountdown、_renderInlineStats。
func TestStaticJS_CountdownHelpers(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	if !strings.Contains(body, "function _stopCountdown(") {
		t.Error("renderer.js: _stopCountdown helper must be defined to clear the countdown interval")
	}
	if !strings.Contains(body, "clearInterval") {
		t.Error("renderer.js: _stopCountdown must call clearInterval to stop the timer")
	}
	if !strings.Contains(body, "function _formatCountdown(") {
		t.Error("renderer.js: _formatCountdown helper must be defined to format seconds as 'Mm SSs'")
	}
	if !strings.Contains(body, "function _renderInlineStats(") {
		t.Error("renderer.js: _renderInlineStats helper must be defined to inject stats into #tr-stats on timeout/cancel")
	}
}

// TestStaticJS_AppendLiveHopAccumulatesHops 驗證 renderer.js 的
// appendLiveHop 函式在追加表格行的同時，也將正規化的躍點資料推入
// _trCollectedHops 陣列，供後續的 _renderInlineStats 計算統計。
func TestStaticJS_AppendLiveHopAccumulatesHops(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function appendLiveHop(")
	if fnStart == -1 {
		t.Fatal("renderer.js: appendLiveHop not found")
	}
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

	if !strings.Contains(fnBody, "_trCollectedHops.push") {
		t.Error("renderer.js: appendLiveHop must push normalised hop data to _trCollectedHops for stats accumulation")
	}
	// The normalised object must include PascalCase keys to match renderRouteStats expectations.
	if !strings.Contains(fnBody, "LossPct") {
		t.Error("renderer.js: appendLiveHop accumulated hop must include LossPct key (PascalCase)")
	}
	if !strings.Contains(fnBody, "AvgRTT") {
		t.Error("renderer.js: appendLiveHop accumulated hop must include AvgRTT key (PascalCase)")
	}
}

// TestStaticJS_FinalizeRendersInlineStats 驗證 finalizeTracerouteProgress
// 函式在更新標題和停止計時器之後，呼叫 _renderInlineStats() 注入統計卡片，
// 使使用者在逾時或取消後仍能看到完整的路由品質摘要。
func TestStaticJS_FinalizeRendersInlineStats(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function finalizeTracerouteProgress(")
	if fnStart == -1 {
		t.Fatal("renderer.js: finalizeTracerouteProgress not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
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

	if !strings.Contains(fnBody, "_stopCountdown") {
		t.Error("renderer.js: finalizeTracerouteProgress must call _stopCountdown() to stop the countdown timer")
	}
	if !strings.Contains(fnBody, "_renderInlineStats") {
		t.Error("renderer.js: finalizeTracerouteProgress must call _renderInlineStats() to show the route stats summary")
	}
	// Countdown bar wrap should be hidden once finalized.
	if !strings.Contains(fnBody, "cbarWrap") {
		t.Error("renderer.js: finalizeTracerouteProgress must hide #tr-countdown-wrap (cbarWrap) when finalized")
	}
}

// TestStaticJS_HideTracerouteProgressStopsCountdown 驗證
// hideTracerouteProgress 函式在隱藏面板前會停止倒數計時器，
// 並清除已累積的狀態（_trCollectedHops、#tr-stats），以確保下次啟動時乾淨。
// 注意：_trStartTime 不在此清除，以便 renderReport() 能計算花費時間。
func TestStaticJS_HideTracerouteProgressStopsCountdown(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function hideTracerouteProgress(")
	if fnStart == -1 {
		t.Fatal("renderer.js: hideTracerouteProgress not found")
	}
	nextFn := strings.Index(body[fnStart+1:], "\n  function ")
	var fnBody string
	if nextFn != -1 {
		fnBody = body[fnStart : fnStart+1+nextFn]
	} else {
		end := fnStart + 500
		if end > len(body) {
			end = len(body)
		}
		fnBody = body[fnStart:end]
	}

	if !strings.Contains(fnBody, "_stopCountdown") {
		t.Error("renderer.js: hideTracerouteProgress must call _stopCountdown() to stop the timer before hiding")
	}
	if !strings.Contains(fnBody, "_trCollectedHops") {
		t.Error("renderer.js: hideTracerouteProgress must clear _trCollectedHops for the next run")
	}
}

// ── JS api-client.js 測試 ─────────────────────────────────────────────────

// TestStaticJS_AbortErrorTracerouteFinalize 驗證 api-client.js 的 AbortError
// 處理邏輯在 _isTraceroute 為 true 時，呼叫
// finalizeTracerouteProgress('traceroute-cancelled') 而非 hideTracerouteProgress，
// 以保留已收集的躍點資料並顯示統計摘要。
func TestStaticJS_AbortErrorTracerouteFinalize(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	fnStart := strings.Index(body, "function runDiag(")
	if fnStart == -1 {
		t.Fatal("api-client.js: runDiag function not found")
	}
	abortIdx := strings.Index(body[fnStart:], "AbortError")
	if abortIdx == -1 {
		t.Fatal("api-client.js: AbortError handler not found in runDiag")
	}
	windowStart := fnStart + abortIdx
	end := windowStart + 800
	if end > len(body) {
		end = len(body)
	}
	window := body[windowStart:end]

	// Must branch on _isTraceroute.
	if !strings.Contains(window, "_isTraceroute") {
		t.Error("api-client.js: AbortError handler must branch on _isTraceroute to distinguish traceroute vs non-traceroute cancel")
	}
	// Traceroute branch must call finalizeTracerouteProgress with 'traceroute-cancelled'.
	if !strings.Contains(window, "finalizeTracerouteProgress('traceroute-cancelled')") {
		t.Error("api-client.js: AbortError handler must call finalizeTracerouteProgress('traceroute-cancelled') for traceroute cancel — keeps hop table + shows stats")
	}
	// Non-traceroute branch must still call hideTracerouteProgress.
	if !strings.Contains(window, "hideTracerouteProgress") {
		t.Error("api-client.js: AbortError handler non-traceroute branch must call hideTracerouteProgress()")
	}
}

// TestStaticJS_InitTracerouteProgressReceivesTimeout 驗證 api-client.js 的
// runDiag 函式在初始化路由追蹤進度面板時，從 #diag-timeout 讀取使用者
// 設定的分鐘數轉換成秒（userTimeoutSec），作為倒數計時的基準—
// 而非使用 req.options.timeout（可能已被自動延長）。
func TestStaticJS_InitTracerouteProgressReceivesTimeout(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-client.js")

	// Must read #diag-timeout from DOM (user-configured minutes, not auto-extended value).
	if !strings.Contains(body, "getElementById('diag-timeout')") {
		t.Error("api-client.js: runDiag must read #diag-timeout to determine the user-configured countdown duration")
	}
	// Must derive userTimeoutSec from the minutes input.
	if !strings.Contains(body, "userTimeoutSec") {
		t.Error("api-client.js: runDiag must derive userTimeoutSec (minutes × 60) from #diag-timeout for the countdown")
	}
	// Must pass it to initTracerouteProgress.
	if !strings.Contains(body, "initTracerouteProgress(host, maxHops, mtrCount, userTimeoutSec)") {
		t.Error("api-client.js: runDiag must pass userTimeoutSec as the 4th argument to initTracerouteProgress()")
	}
}

// ── JS api-builder.js 測試 ────────────────────────────────────────────────

// TestStaticJS_ApiBuilderTimeoutMinutes 驗證 api-builder.js 的 buildRequest
// 函式將整數分鐘輸入值（#diag-timeout）轉換為後端接受的秒數格式（如 "60s"）。
func TestStaticJS_ApiBuilderTimeoutMinutes(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/api-builder.js")

	fnStart := strings.Index(body, "function buildRequest(")
	if fnStart == -1 {
		t.Fatal("api-builder.js: buildRequest not found")
	}
	end := fnStart + 1200
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	// Must read timeout as integer (minutes), not raw string.
	if !strings.Contains(fnBody, "timeoutMin") {
		t.Error("api-builder.js: buildRequest must read timeout input as minutes integer (variable named timeoutMin or similar)")
	}
	// Must multiply by 60 to convert minutes → seconds.
	if !strings.Contains(fnBody, "* 60") && !strings.Contains(fnBody, "*60") {
		t.Error("api-builder.js: buildRequest must convert minutes to seconds by multiplying by 60")
	}
	// Must produce a string with 's' suffix for the backend.
	if !strings.Contains(fnBody, "+ 's'") && !strings.Contains(fnBody, "+'s'") {
		t.Error("api-builder.js: buildRequest must append 's' suffix to produce a valid Go duration string (e.g. \"60s\")")
	}
}

// TestStaticJS_FormatElapsed 驗證 renderer.js 定義了 _formatElapsed 輔助函式，
// 能將毫秒格式化為「Xm Ys」或「Zs」字串，供路由摘要「花費時間」欄位使用。
func TestStaticJS_FormatElapsed(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	if !strings.Contains(body, "_formatElapsed") {
		t.Error("renderer.js: _formatElapsed helper must be defined for formatting elapsed time in the route stats card")
	}
	// Must handle the ms→sec conversion.
	if !strings.Contains(body, "Math.round(ms / 1000)") {
		t.Error("renderer.js: _formatElapsed must convert ms to seconds with Math.round(ms / 1000)")
	}
}

// TestStaticJS_RenderRouteStatsElapsed 驗證 renderRouteStats 函式接受
// 可選的 elapsedMs 參數，並在其大於 0 時將「花費時間」加入統計項目。
func TestStaticJS_RenderRouteStatsElapsed(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function renderRouteStats(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderRouteStats not found")
	}
	end := fnStart + 3000
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	if !strings.Contains(fnBody, "elapsedMs") {
		t.Error("renderer.js: renderRouteStats must accept elapsedMs parameter for the elapsed time stat item")
	}
	if !strings.Contains(fnBody, "route-stats-elapsed") {
		t.Error("renderer.js: renderRouteStats must reference 'route-stats-elapsed' i18n key when elapsedMs > 0")
	}
	if !strings.Contains(fnBody, "_formatElapsed") {
		t.Error("renderer.js: renderRouteStats must call _formatElapsed to format the elapsed duration")
	}
}

// TestStaticI18n_RouteStatsElapsed 驗證 i18n.js 在 EN 與 ZH 兩個語系
// 都定義了 'route-stats-elapsed' 鍵值，供路由摘要「花費時間」欄位顯示。
func TestStaticI18n_RouteStatsElapsed(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/i18n.js")

	if !strings.Contains(body, "'route-stats-elapsed'") {
		t.Error("i18n.js: 'route-stats-elapsed' key must be defined in both EN and ZH locales")
	}
	// EN value.
	if !strings.Contains(body, "Time spent") {
		t.Error("i18n.js: EN 'route-stats-elapsed' value must be 'Time spent'")
	}
	// ZH value.
	if !strings.Contains(body, "\u82b1\u8cbb\u6642\u9593") {
		t.Error("i18n.js: ZH 'route-stats-elapsed' value must be '花費時間'")
	}
}

// TestStaticJS_RenderReportElapsed 驗證 renderReport 函式在渲染路由摘要時
// 會計算花費時間（elapsedMs）並將其傳入 renderRouteSection。
func TestStaticJS_RenderReportElapsed(t *testing.T) {
	body := fetchBody(t, newStaticHandler(t), "/renderer.js")

	fnStart := strings.Index(body, "function renderReport(")
	if fnStart == -1 {
		t.Fatal("renderer.js: renderReport not found")
	}
	end := fnStart + 800
	if end > len(body) {
		end = len(body)
	}
	fnBody := body[fnStart:end]

	if !strings.Contains(fnBody, "elapsedMs") {
		t.Error("renderer.js: renderReport must compute elapsedMs from _trStartTime for traceroute results")
	}
	if !strings.Contains(fnBody, "_trStartTime") {
		t.Error("renderer.js: renderReport must read _trStartTime to derive elapsed wall-clock time")
	}
	if !strings.Contains(fnBody, "renderRouteSection") {
		t.Error("renderer.js: renderReport must call renderRouteSection (passing elapsedMs) to include elapsed stat")
	}
}
