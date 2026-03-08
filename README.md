# go-pathprobe

_pathprobe_ 是一套多協定網路路徑與服務診斷工具。  
主要以 **內嵌 Web UI（HTTP 伺服器）** 作為操作介面，支援公網 IP 偵測、DNS 對比、端口可達性統計、HTTP / SMTP / FTP / FTPS / SFTP 協定層深度探測、Geo 地圖標記，以及診斷歷史紀錄管理。

## 功能概覽

| 里程碑 | 項目 | 說明 |
|--------|------|------|
| M0 | 基礎骨架 | Cobra CLI、diag 子命令群、共用旗標框架、結構化 slog 日誌 |
| M1 | 公網資訊與 DNS 對比 | STUN / HTTPS echo 雙來源取得公網 IP；系統 DNS + DoH (Cloudflare / Google) 查詢 A/AAAA/MX 並比對差異 |
| M2 | 端口可達性 | TCP MTR 風格多次探測、每 hop 統計 loss% / min/avg/max RTT |
| M3 | Web / HTTP 深度探測 | HTTP(S) 狀態碼、回應標頭、重導向鏈追蹤 |
| M4 | Mail / 檔案傳輸協定探測 | SMTP EHLO / STARTTLS / AUTH、FTP / FTPS（implicit & explicit AUTH TLS）+ PASV LIST、SFTP SSH 握手演算法 + 目錄列舉 |
| M5 | Geo 標註與多格式報告 | MaxMind GeoLite2 離線 DB 標註 IP 地理位置與 ASN |
| M6 | 封裝與發佈 | ldflags 版本注入、ICMP 權限偵測 + TCP 降級提示、跨平台交叉編譯腳本 |
| M7-A | REST API 伺服器 | pkg/server：GET /api/health、POST /api/diag、POST /api/diag/stream |
| M7-B | 內嵌 Web UI | 靜態資源（//go:embed web）；表單驅動診斷、即時 SSE 進度流 |
| M7-C | SSE 串流進度 | Runner ProgressHook；POST /api/diag/stream 逐事件推送 |
| M7-D | 歷史紀錄 + 互動地圖 | Store 介面 + MemoryStore；GET /api/history、GET /api/history/{id}；嵌入 Leaflet 1.9.4 地圖視覺化 |

---

## 架構

```
cmd/pathprobe/          ← 程式進入點、Windows manifest
pkg/
  cli/                  ← Cobra 命令樹（root / diag / serve / version）
  server/               ← HTTP 伺服器、路由、所有 API Handler、嵌入 Web UI
    web/                ← 靜態資源（index.html / style.css / app.js / leaflet.*）
  store/                ← Store 介面、MemoryStore（診斷歷史）
  diag/                 ← Runner 介面、Dispatcher、Request / DiagReport 資料模型
  netprobe/             ← 各協定探測實作（TCP port、DNS、HTTP、SMTP、FTP、SFTP）
  geo/                  ← GeoLite2 封裝（Locator 介面、NoopLocator、GeoLite2Locator）
  report/               ← AnnotatedReport 建構、TableWriter / JSONWriter / HTMLWriter
  syscheck/             ← OS 能力偵測（ICMP raw socket 可用性）
  version/              ← build-time 版本變數（由 ldflags 注入）
  logging/              ← slog 工廠
```

**設計原則**：全程依賴介面注入（PortProber、PublicIPFetcher、Locator、Store 等），Runner 之間透過 DiagReport nil-safe 方法傳遞結構化結果，向下相容且不互相耦合。

---

## 環境需求

- Go 1.22+
- Windows / Linux / macOS（cross-compile 支援全平台）
- 原生 ICMP 探測需管理員 / root 權限；無權限時自動降級為 TCP 模式並顯示警告
- Geo 標註需另行下載 [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) 資料庫（.mmdb），不提供則靜默略過

---

## 建置

```powershell
# 本機快速建置（Windows amd64，含版本 ldflags）
.\build.ps1

# 跨平台發佈（預設四個目標）
.\release.ps1

# 指定版本與目標
.\release.ps1 -Version v1.4.0 -Targets "windows/amd64,linux/amd64"

# 同時嵌入 requireAdministrator manifest（需安裝 rsrc）
.\release.ps1 -Version v1.4.0 -WithManifest
# go install github.com/akavel/rsrc@latest
```

`release.ps1` 會依序執行 `go mod tidy` → `go vet` → `go test` 通過後再進行交叉編譯，輸出至 `./bin/`。

---

## 啟動 Web 伺服器

```powershell
# 預設監聽 :8080
.\bin\pathprobe.exe serve

# 自訂監聽位址
.\bin\pathprobe.exe serve --addr :9090

# 啟用 Geo 標註（需 GeoLite2 .mmdb）
.\bin\pathprobe.exe serve --geo-db-city ./GeoLite2-City.mmdb --geo-db-asn ./GeoLite2-ASN.mmdb

# 自訂寫入逾時（長時間診斷時調高）
.\bin\pathprobe.exe serve --addr :8080 --write-timeout 120s
```

伺服器啟動後，瀏覽器開啟 http://localhost:8080 即可使用 Web UI。  
以 Ctrl-C 中斷，伺服器會進行 5 秒優雅關閉。

### 可用旗標

| 旗標 | 預設值 | 說明 |
|------|--------|------|
| `--addr` | :8080 | 監聽位址，格式 host:port 或 :port |
| `--write-timeout` | 120s | HTTP 寫入逾時（建議 >= 診斷最長時間） |
| `--geo-db-city` | — | GeoLite2-City.mmdb 路徑（IP 地理位置） |
| `--geo-db-asn` | — | GeoLite2-ASN.mmdb 路徑（AS 號碼與組織） |
| `--log-level` | info | 日誌層級：debug / info / warn / error |

---

## Web UI 操作流程

### 1. 執行診斷

1. 從「Target」下拉選單選擇目標類型（web / smtp / ftp / sftp / imap / pop）
2. 填入「Target Host」與「Ports」（可多個，逗號分隔）
3. 依目標類型填寫專屬欄位（DNS 網域、SMTP 帳密、FTP 選項等）
4. 展開「Advanced Options」可調整 MTR Count 與 Timeout
5. 點擊「▶ Run Diagnostic」

診斷過程中，Progress Log 會即時顯示每個探測階段的狀態（透過 SSE 推送）。

### 2. 查看結果

診斷完成後，Results 區塊顯示：

| 區塊 | 內容 |
|------|------|
| Summary | Target 類型、Host、執行時間、公網 IP |
| Port Connectivity | 每個端口的 Sent / Recv / 丟包率 / Min/Avg/Max RTT |
| Protocol Results | 各協定握手結果（OK / FAIL）與摘要訊息 |
| Geo Information | 公網 IP 與目標主機的地理位置（City / Country / ASN） |
| 互動地圖 | Leaflet + OpenStreetMap 地圖，標記公網 IP 與目標主機位置（需 Geo DB） |

### 3. 歷史紀錄

頁面底部的「Diagnostic History」面板列出最近 100 筆診斷記錄（最新在前）。  
點擊任一歷史項目，可將該次診斷結果載入 Results 區塊並重繪地圖，方便與目前結果比對。

---

## REST API

Web UI 底層透過以下 API 與伺服器溝通，亦可直接呼叫進行自動化整合。

| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /api/health | 存活探針，回傳版本與 build 時間 |
| POST | /api/diag | 執行診斷，同步回傳完整 AnnotatedReport（JSON） |
| POST | /api/diag/stream | 執行診斷，以 SSE 串流逐步推送 progress / result / error 事件 |
| GET | /api/history | 取得歷史清單（[]HistoryListItem，最新在前） |
| GET | /api/history/{id} | 取得指定歷史記錄的完整 AnnotatedReport |

**請求範例（POST /api/diag）：**

```json
{
  "target": "web",
  "options": {
    "timeout": "30s",
    "net": { "host": "example.com", "ports": [443] },
    "web": { "domains": ["example.com"], "types": ["A", "MX"] }
  }
}
```

---

## 測試

```powershell
go test -count=1 ./...
```

主要測試涵蓋範圍：

| 套件 | 測試重點 |
|------|----------|
| `pkg/netprobe` | TCP 端口探測、DNS 解析、DoH、HTTP 探測、SMTP StartTLS / AUTH、FTP Banner / AUTH TLS、SFTP 握手 |
| `pkg/diag` | Runner 編排、DiagReport nil-safe 方法、全域選項驗證 |
| `pkg/geo` | NoopLocator 降級行為、DB 路徑驗證 |
| `pkg/report` | AnnotatedReport 建構、TableWriter / JSONWriter / HTMLWriter 輸出正確性 |
| `pkg/server` | API Handler（health / diag / history）路由、SSE 事件格式、錯誤回應結構 |
| `pkg/store` | MemoryStore Save/List/Get、FIFO 淘汰行為、容量預設值 |
| `pkg/syscheck` | ICMPChecker 介面契約一致性 |
| `pkg/version` | Version / BuildTime 變數非空斷言 |
| `pkg/cli` | 旗標傳遞、版本命令輸出 |

---

## 開發指引

- 遵守 SOLID 原則；所有外部依賴以介面注入，便於測試替換。
- 新增 Runner 時須實作 `Runner` 介面並向 `Dispatcher` 註冊；以 `req.Report.AddProto()` 等 nil-safe 方法寫入結構化結果。
- 新增 API 端點時在 `pkg/server/server.go` 的 `New()` 中向 mux 註冊，Handler 以 `writeJSON` / `writeError` 統一回應格式。
- 新增功能同步補齊單元測試，並執行 `go test -count=1 ./...` 確認全部通過。
- 不使用 CGO（`CGO_ENABLED=0`），確保跨平台靜態連結。
