'use strict';

// api-client.js — API 呼叫、SSE 串流處理、錯誤呈現
// Exposes: PathProbe.ApiClient = { runDiag, handleSSEMessage, appendProgress,
//                                  fetchVersion, showError, localizeError }
//   runDiag()         — 執行診斷（POST /api/diag/stream）
//   handleSSEMessage() — 解析 SSE 訊息並分派給各模組
//   appendProgress()   — 新增 progress 進度列（禁用 innerHTML，防 XSS）
//   fetchVersion()     — 讀取版本號碼（GET /api/health）
//   showError()        — 顯示使用者可讀的錯誤訊息
//   localizeError()    — 將原始伺服器錯誤轉換為 i18n 本地化字串
//
// 模組相依（runtime-resolved，全部透過 window.PathProbe 存取）：
//   PathProbe.Locale       — t(key), getLocale()
//   PathProbe.ApiBuilder   — buildRequest()
//   PathProbe.Form         — getRunningHTML()
//   PathProbe.Renderer     — renderReport()
//   PathProbe.Map          — renderMap()
//   PathProbe.History      — loadHistory()
//
// 全域暴露：window.runDiag（供 index.html #run-btn onclick 呼叫）

(() => {

  // ── 模組狀態 ─────────────────────────────────────────────────────────

  /** 目前進行中請求的 AbortController，由 cancelDiag() 用來中斷請求。 */
  let _currentAbortController = null;

  /** 是否為路由追蹤模式——影響 AbortError / 網路錯誤的 UI 處理策略。 */
  let _isTraceroute = false;

  // ── 執行期相依注入（runtime shims）────────────────────────────────────

  function _t(key) {
    return (window.PathProbe && window.PathProbe.Locale)
      ? window.PathProbe.Locale.t(key)
      : key;
  }

  function _buildRequest() {
    return (window.PathProbe && window.PathProbe.ApiBuilder)
      ? window.PathProbe.ApiBuilder.buildRequest()
      : { target: '', options: {} };
  }

  function _getRunningHTML() {
    return (window.PathProbe && window.PathProbe.Form)
      ? window.PathProbe.Form.getRunningHTML()
      : '<span class="anim-dots"><span></span><span></span><span></span></span>';
  }

  // ── 錯誤處理 ─────────────────────────────────────────────────────────

  /**
   * 將原始伺服器錯誤字串（可能含 Go 內部訊息）轉換為本地化的友善描述。
   * 三段分支：timeout → err-timeout、no-runner → err-no-runner、其餘去前綴。
   */
  function localizeError(msg) {
    if (!msg) return _t('err-unknown');
    const lower = msg.toLowerCase();
    if (lower.includes('timed out') || lower.includes('deadline exceeded')) {
      return _t('err-timeout');
    }
    if (lower.includes('no runner registered') || lower.includes('no handler registered')) {
      return _t('err-no-runner');
    }
    // 去除 "diagnostic error: " 前綴以提升顯示整潔度。
    return msg.replace(/^diagnostic error:\s*/i, '');
  }

  /** 顯示使用者可讀的錯誤橫幅。 */
  function showError(msg) {
    const banner  = document.getElementById('error-banner');
    const textEl  = document.getElementById('error-text');
    const friendly = localizeError(msg);
    if (textEl) {
      textEl.textContent = friendly;
    } else {
      // 相容舊版佈局（無 error-text span）。
      banner.textContent = '\u26a0  ' + friendly;
    }
    banner.hidden = false;
  }

  // ── 進度記錄 ─────────────────────────────────────────────────────────

  /**
   * 將單筆進度事件附加到進度記錄列。
   * 嚴禁使用 innerHTML — 以 textContent 防止 XSS（進度字串來自伺服器）。
   */
  function appendProgress(el, ev) {
    if (!el) return;
    const entry = document.createElement('div');
    entry.className = 'progress-entry';
    const stageSpan = document.createElement('span');
    stageSpan.className = 'stage';
    stageSpan.textContent = ev.stage || '';
    const msgSpan = document.createElement('span');
    msgSpan.className = 'msg';
    msgSpan.textContent = ev.message || '';
    entry.appendChild(stageSpan);
    entry.appendChild(msgSpan);
    el.appendChild(entry);
    el.scrollTop = el.scrollHeight;
  }

  // ── SSE 訊息解析 ──────────────────────────────────────────────────────

  /**
   * 解析單筆 SSE 訊息區塊並分派給對應模組（progress / result / error）。
   * result 事件：先顯示 #results（resultEl.hidden = false）再呼叫 renderMap()，
   * 確保 Leaflet 初始化時容器具有非零尺寸（防止地圖空白）。
   */
  function handleSSEMessage(raw, progressEl, resultEl) {
    let evtName = '', dataStr = '';
    for (const line of raw.split('\n')) {
      if (line.startsWith('event: '))     evtName = line.slice(7).trim();
      else if (line.startsWith('data: ')) dataStr = line.slice(6);
    }
    if (!dataStr) return;

    let payload;
    try { payload = JSON.parse(dataStr); } catch { return; }

    if (evtName === 'progress') {
      if (payload.stage === 'traceroute-hop' && payload.hop &&
          window.PathProbe && window.PathProbe.Renderer) {
        window.PathProbe.Renderer.appendLiveHop(payload.hop);
      } else {
        appendProgress(progressEl, payload);
      }
    } else if (evtName === 'result') {
      if (progressEl) progressEl.hidden = true;
      if (window.PathProbe && window.PathProbe.Renderer) {
        window.PathProbe.Renderer.hideTracerouteProgress();
        window.PathProbe.Renderer.renderReport(payload);
      }
      // 先顯示 #results，再初始化地圖（確保 Leaflet 容器有版面尺寸）
      resultEl.hidden = false;
      // renderMap — 執行期解析，透過 PathProbe.Map
      if (window.PathProbe && window.PathProbe.Map) {
        // payload.Route 含路由追蹤躁點資料（HasGeo/Lat/Lon），傳入後
        // renderMap() 會在地圖上呼現實網路路徑之多點連線。
        // 非路由追蹤模式時 payload.Route 為 null/undefined，
        // renderMap() 自動回落至原始兩點弧線行為。
        window.PathProbe.Map.renderMap(payload.PublicGeo, payload.TargetGeo, payload.Route);
      }
      // loadHistory — 執行期解析，透過 PathProbe.History
      if (window.PathProbe && window.PathProbe.History) {
        window.PathProbe.History.loadHistory();
      }
      resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
    } else if (evtName === 'error') {
      const errMsg = payload.error || '';
      // "context canceled" / "deadline exceeded" during traceroute = server-side
      // timeout; keep the hop table visible and show a finalized status instead
      // of an alarming error banner.
      const isCtxCancel = errMsg.toLowerCase().includes('context canceled') ||
                          errMsg.toLowerCase().includes('deadline exceeded');
      if (_isTraceroute && isCtxCancel) {
        if (window.PathProbe && window.PathProbe.Renderer) {
          window.PathProbe.Renderer.finalizeTracerouteProgress('traceroute-timeout');
        }
      } else {
        // Non-traceroute or unexpected error: clear progress and show banner.
        if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = true; }
        if (window.PathProbe && window.PathProbe.Renderer) {
          window.PathProbe.Renderer.hideTracerouteProgress();
        }
        showError(errMsg || 'diagnostic error');
      }
    }
  }

  // ── 主要診斷執行器 ───────────────────────────────────────────────────

  /** 執行診斷流程：POST /api/diag/stream，以 ReadableStream 接收 SSE。 */
  async function runDiag() {
    const btn        = document.getElementById('run-btn');
    const cancelBtn  = document.getElementById('cancel-btn');
    const errorEl    = document.getElementById('error-banner');
    const resultEl   = document.getElementById('results');
    const progressEl = document.getElementById('progress-log');

    _currentAbortController = new AbortController();

    btn.disabled    = true;
    btn.innerHTML   = _getRunningHTML();
    errorEl.hidden  = true;
    resultEl.hidden = true;
    if (cancelBtn) cancelBtn.hidden = false;

    try {
      const req = _buildRequest();
      if (req === null) {
        // buildRequest 內部驗證失敗；錯誤訊息已透過 showError 顯示。
        if (progressEl) { progressEl.hidden = true; progressEl.innerHTML = ''; }
        return;
      }

      // 判斷是否為路由追蹤模式——展示即時躍點進度面板而非文字記錄。
      const isTraceroute = req.target === 'web' &&
                           req.options && req.options.web &&
                           req.options.web.mode === 'traceroute';
      _isTraceroute = isTraceroute;

      if (isTraceroute) {
        if (progressEl) progressEl.hidden = true;
        const host     = (req.options.net && req.options.net.host) || '';
        const maxHops  = (req.options.web && req.options.web.max_hops) || 30;
        const mtrCount = req.options.mtr_count || 5;
        if (window.PathProbe && window.PathProbe.Renderer) {
          window.PathProbe.Renderer.initTracerouteProgress(host, maxHops, mtrCount);
        }
      } else {
        if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = false; }
      }

      const resp = await fetch('/api/diag/stream', {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify(req),
        signal:  _currentAbortController.signal,
      });

      // 非 2xx（反向代理驗證錯誤等），嘗試解析 JSON body。
      if (!resp.ok) {
        const data = await resp.json().catch(() => ({ error: 'HTTP ' + resp.status }));
        showError(data.error || 'Server returned HTTP ' + resp.status);
        return;
      }

      // 以 ReadableStream 解析 SSE（瀏覽器 EventSource 不支援 POST）。
      const reader  = resp.body.getReader();
      const decoder = new TextDecoder();
      let   buffer  = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // SSE 訊息以空行（\n\n）分隔。
        let boundary;
        while ((boundary = buffer.indexOf('\n\n')) !== -1) {
          const raw = buffer.slice(0, boundary);
          buffer    = buffer.slice(boundary + 2);
          handleSSEMessage(raw, progressEl, resultEl);
        }
      }
      // 排空緩衝區剩餘內容。
      if (buffer.trim()) handleSSEMessage(buffer, progressEl, resultEl);

    } catch (err) {
      if (err.name === 'AbortError') {
        // User actively cancelled — hide the traceroute panel and log a friendly note.
        if (window.PathProbe && window.PathProbe.Renderer) {
          window.PathProbe.Renderer.hideTracerouteProgress();
        }
        if (progressEl) {
          progressEl.innerHTML = '';
          progressEl.hidden = false;
          appendProgress(progressEl, { stage: 'info', message: _t('traceroute-cancelled').replace('{n}', '') });
        }
        return;
      }
      // Network-level failure (e.g. server closed stream after context cancel):
      // for traceroute, preserve the hop table in finalized state instead of
      // showing an alarming "network error" banner.
      if (_isTraceroute) {
        if (window.PathProbe && window.PathProbe.Renderer) {
          window.PathProbe.Renderer.finalizeTracerouteProgress('traceroute-timeout');
        }
      } else {
        if (progressEl) { progressEl.innerHTML = ''; progressEl.hidden = true; }
        if (window.PathProbe && window.PathProbe.Renderer) {
          window.PathProbe.Renderer.hideTracerouteProgress();
        }
        showError('Request failed: ' + err.message);
      }
    } finally {
      btn.disabled    = false;
      btn.textContent = _t('btn-run');
      if (cancelBtn) cancelBtn.hidden = true;
      _currentAbortController = null;
      _isTraceroute = false;
    }
  }

  // ── 版本徽章 ─────────────────────────────────────────────────────────

  /** 從 /api/health 取得版本號碼並填入 #version-badge（非致命）。 */
  async function fetchVersion() {
    try {
      const r = await fetch('/api/health');
      if (!r.ok) return;
      const { version } = await r.json();
      const el = document.getElementById('version-badge');
      if (el && version) el.textContent = version;
    } catch (_) { /* 非致命 — 版本徽章維持空白 */ }
  }

  // ── 公開 API ─────────────────────────────────────────────────────────

  /** 中斷正在進行中的診斷請求（如果有）。由 #cancel-btn onclick 呼叫。 */
  function cancelDiag() {
    if (_currentAbortController) {
      _currentAbortController.abort();
    }
  }

  // 全域暴露 runDiag / cancelDiag，供 index.html 按鈕 onclick 屬性呼叫。
  window.runDiag = runDiag;
  window.cancelDiag = cancelDiag;

  const _ns = window.PathProbe || {};
  _ns.ApiClient = {
    runDiag,
    cancelDiag,
    handleSSEMessage,
    appendProgress,
    fetchVersion,
    showError,
    localizeError,
  };
  window.PathProbe = _ns;

})();
