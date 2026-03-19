'use strict';

// app.js — 組裝入口（Assembly Entry Point）
//
// 唯一職責：DOMContentLoaded 後，依序呼叫各模組初始化 API。
// 本檔案不含任何業務邏輯，所有邏輯由各自子模組負責。
//
// index.html 載入順序（依賴鏈）：
//   leaflet.js, i18n.js, config.js, locale.js, theme.js, form.js,
//   api-builder.js, renderer.js, map-connector.js, map.js,
//   api-client.js, history.js → app.js（本檔案最後載入）
//
// 全域 onclick 入口（由各子模組自行暴露）：
//   window.setTheme       ← theme.js
//   window.setLocale      ← locale.js
//   window.runDiag        ← api-client.js
//   window.loadHistoryEntry ← history.js

document.addEventListener('DOMContentLoaded', () => {
  window.PathProbe.Form.init();
  window.PathProbe.ApiClient.fetchVersion();
  window.PathProbe.History.loadHistory();
  window.PathProbe.Theme.initTheme();
  window.PathProbe.Locale.initLocale();
});


