'use strict';

// history.js — 歷史紀錄管理模組
// Exposes: PathProbe.History = { loadHistory, rerenderLast }
//
// 職責範圍：
//   _lastHistoryItems   — 最後一次載入的歷史項目快取（供 rerenderLast 使用）
//   formatHistoryTime() — 將 UTC ISO 時間字串格式化為本地化顯示
//   loadHistory()       — GET /api/history 並重繪歷史面板
//   renderHistoryList() — 將歷史項目渲染至 #history-list
//   loadHistoryEntry()  — GET /api/history/:id 並顯示為目前診斷結果
//   rerenderLast()      — 以快取的項目重繪清單（供 locale.js 呼叫）
//
// 跨模組相依（runtime-resolved，全部透過 window.PathProbe 存取）：
//   PathProbe.ApiClient  — showError()
//   PathProbe.Renderer   — renderReport()
//   PathProbe.Map        — renderMap()
//
// 全域暴露：window.loadHistoryEntry（供 #history-list 的 onclick 屬性呼叫）

(() => {

  // ── 私有狀態 ─────────────────────────────────────────────────────────
  // 快取最後一次載入的歷史項目，以便 rerenderLast() 可在語言切換後重繪。
  let _lastHistoryItems = null;

  // ── 私有工具函式 ──────────────────────────────────────────────────────

  /**
   * HTML 跳脫（XSS 防護）。
   * 各模組保有私有副本以維持 IIFE 封裝的獨立性。
   */
  function esc(s) {
    return String(s)
      .replace(/&/g,  '&amp;')
      .replace(/</g,  '&lt;')
      .replace(/>/g,  '&gt;')
      .replace(/"/g,  '&quot;')
      .replace(/'/g,  '&#39;');
  }

  /**
   * 將原始伺服器錯誤委派至 api-client.js 顯示。
   * 透過 PathProbe.ApiClient 執行期解析，不產生強依賴。
   */
  function _showError(msg) {
    if (window.PathProbe && window.PathProbe.ApiClient) {
      window.PathProbe.ApiClient.showError(msg);
    }
  }

  // ── 時間格式化 ────────────────────────────────────────────────────────

  /**
   * 將 UTC ISO 時間字串格式化為本地化顯示字串。
   *
   * 讀取 document.documentElement.lang 而非直接存取 PathProbe.Locale._locale，
   * 以隔離 locale 模組的私有狀態。locale.js 在每次 applyLocale() 時都會同步
   * 更新 document.documentElement.lang，確保此處取得的值永遠是最新語言設定。
   */
  function formatHistoryTime(isoString) {
    if (!isoString) return '';
    const locale = document.documentElement.lang || 'en';
    try {
      return new Date(isoString).toLocaleString(locale);
    } catch (_) {
      return new Date(isoString).toLocaleString();
    }
  }

  // ── 歷史清單渲染 ──────────────────────────────────────────────────────

  /** 將歷史項目陣列渲染至 #history-list。 */
  function renderHistoryList(items) {
    // 快取以供 rerenderLast() 在語言切換後重繪使用。
    _lastHistoryItems = items;

    const emptyEl = document.getElementById('history-empty');
    const listEl  = document.getElementById('history-list');
    if (!listEl || !emptyEl) return;

    if (items.length === 0) {
      emptyEl.hidden = false;
      listEl.hidden  = true;
      return;
    }

    emptyEl.hidden = true;
    listEl.hidden  = false;
    listEl.innerHTML = items.map(item => {
      const ts = formatHistoryTime(item.created_at);
      const id = JSON.stringify(String(item.id));
      return '<li class="history-item" onclick="loadHistoryEntry(' + id + ')">' +
        '<span class="hi-badge">' + esc(item.target      || '\u2014') + '</span>' +
        '<span class="hi-host">'  + esc(item.host        || '\u2014') + '</span>' +
        '<span class="hi-time">'  + esc(ts)                           + '</span>' +
      '</li>';
    }).join('');
  }

  // ── API 呼叫 ──────────────────────────────────────────────────────────

  /** 從伺服器取得歷史清單並重繪面板（非致命）。 */
  async function loadHistory() {
    try {
      const r = await fetch('/api/history');
      if (!r.ok) return;
      const items = await r.json();
      renderHistoryList(Array.isArray(items) ? items : []);
    } catch (_) { /* 非致命 — 面板維持現有狀態 */ }
  }

  /**
   * 取得單筆歷史記錄並顯示為目前診斷結果。
   *
   * 設計要點：
   *   - showError   → PathProbe.ApiClient.showError()（執行期解析）
   *   - renderReport → PathProbe.Renderer.renderReport()（執行期解析）
   *   - renderMap   → PathProbe.Map.renderMap()（執行期解析）
   *   - resultEl.hidden = false 必須在 renderMap() 之前執行，
   *     以確保 Leaflet 初始化時容器具有非零版面尺寸（防止地圖空白）。
   */
  async function loadHistoryEntry(id) {
    const resultEl = document.getElementById('results');
    try {
      const r = await fetch('/api/history/' + encodeURIComponent(id));
      if (!r.ok) {
        _showError('History entry not found: ' + id);
        return;
      }
      const report = await r.json();
      // renderReport — 執行期解析，透過 PathProbe.Renderer
      if (window.PathProbe && window.PathProbe.Renderer) {
        window.PathProbe.Renderer.renderReport(report);
      }
      // 先顯示 #results，再初始化地圖（確保 Leaflet 容器有版面尺寸）
      if (resultEl) resultEl.hidden = false;
      // renderMap — 執行期解析，透過 PathProbe.Map
      if (window.PathProbe && window.PathProbe.Map) {
        window.PathProbe.Map.renderMap(report.PublicGeo, report.TargetGeo);
      }
      if (resultEl) {
        resultEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    } catch (err) {
      _showError('Failed to load history entry: ' + err.message);
    }
  }

  // ── rerenderLast（供 locale.js 呼叫）────────────────────────────────

  /**
   * 以快取的歷史項目重繪清單。
   * locale.js 在每次 applyLocale() 後透過 PathProbe.History.rerenderLast()
   * 呼叫此函式，以便時間戳記反映最新的語言設定。
   */
  function rerenderLast() {
    if (_lastHistoryItems) renderHistoryList(_lastHistoryItems);
  }

  // ── 公開 API ─────────────────────────────────────────────────────────

  // 全域暴露 loadHistoryEntry，供 renderHistoryList 產生的 onclick 屬性呼叫。
  window.loadHistoryEntry = loadHistoryEntry;

  const _ns = window.PathProbe || {};
  _ns.History = { loadHistory, rerenderLast };
  window.PathProbe = _ns;

})();
