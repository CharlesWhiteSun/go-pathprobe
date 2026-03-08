# go-pathprobe

多協定網路路徑與服務診斷 CLI 工具，涵蓋公網 IP 偵測、DNS 對比、端口可達性統計、HTTP/SMTP/FTP/FTPS/SFTP 協定層深度探測，以及帶 Geo 標註的 HTML / JSON / 表格三種格式報告輸出。

## 功能概覽

| 里程碑 | 項目 | 說明 |
|--------|------|------|
| M0 | 基礎骨架 | Cobra CLI、`diag` 子命令群、共用旗標框架、結構化 slog 日誌 |
| M1 | 公網資訊與 DNS 對比 | STUN / HTTPS echo 雙來源取得公網 IP；系統 DNS + DoH (Cloudflare / Google) 查詢 A/AAAA/MX 並比對差異 |
| M2 | 端口可達性 | TCP MTR 風格多次探測、每 hop 統計 loss% / min/avg/max RTT |
| M3 | Web / HTTP 深度探測 | HTTP(S) 狀態碼、回應標頭、重導向鏈追蹤 |
| M4 | Mail / 檔案傳輸協定探測 | SMTP EHLO / STARTTLS / AUTH、FTP / FTPS（implicit & explicit AUTH TLS）+ PASV LIST、SFTP SSH 握手演算法 + 目錄列舉 |
| M5 | Geo 標註與多格式報告 | MaxMind GeoLite2 離線 DB 標註 IP 地理位置與 ASN；輸出 CLI 表格、JSON、內嵌 Leaflet+OSM 互動式 HTML 地圖報告 |
| M6 | 封裝與發佈 | `ldflags` 版本注入、ICMP 權限偵測 + TCP 降級提示、跨平台交叉編譯腳本、Windows `requireAdministrator` manifest |

---

## 架構

```
cmd/pathprobe/          ← 程式進入點、Windows manifest
pkg/
  cli/                  ← Cobra 命令樹（root / diag / version）
  diag/                 ← Runner 介面、Dispatcher、Request / DiagReport 資料模型
  netprobe/             ← 各協定探測實作（TCP port、DNS、HTTP、SMTP、FTP、SFTP）
  geo/                  ← GeoLite2 封裝（Locator 介面、NoopLocator、GeoLite2Locator）
  report/               ← Writer 介面、TableWriter / JSONWriter / HTMLWriter（內嵌模板）
  syscheck/             ← OS 能力偵測（ICMP raw socket 可用性）
  version/              ← build-time 版本變數（由 ldflags 注入）
  logging/              ← slog 工廠
```

**設計原則**：全程依賴介面注入（`PortProber`、`PublicIPFetcher`、`DNSResolver`、`Locator`、`Writer` 等），Runner 之間透過 `DiagReport` nil-safe 方法傳遞結構化結果，向下相容且不互相耦合。

---

## 環境需求

- Go 1.25.1+
- Windows / Linux / macOS（cross-compile 支援全平台）
- 原生 ICMP 探測需要管理員 / root 權限；無權限時自動降級為 TCP 模式並於啟動時顯示警告
- Geo 標註需另行下載 [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) 資料庫（`.mmdb`），不提供則靜默略過

---

## 建置

```powershell
# 本機快速建置（Windows amd64，含版本 ldflags）
.\build.ps1

# 跨平台發佈（預設四個目標）
.\release.ps1

# 指定版本與目標
.\release.ps1 -Version v1.3.0 -Targets "windows/amd64,linux/amd64"

# 同時嵌入 requireAdministrator manifest（需安裝 rsrc）
.\release.ps1 -Version v1.3.0 -WithManifest
# go install github.com/akavel/rsrc@latest
```

`release.ps1` 會依序執行 `go mod tidy` → `go vet` → `go test` 通過後再進行交叉編譯，輸出至 `./bin/`。

---

## CLI 使用說明

### 查詢版本

```powershell
pathprobe version
# PathProbe v1.2 (built 2026-03-08T19:21:05Z)
```

### 共用旗標（所有 `diag` 子命令適用）

| 旗標 | 預設值 | 說明 |
|------|--------|------|
| `--json` | false | 輸出 JSON |
| `--report <path>` | — | 將 HTML 互動報告寫入指定路徑 |
| `--geo-db-city <path>` | — | GeoLite2-City.mmdb 路徑（IP 地理位置） |
| `--geo-db-asn <path>` | — | GeoLite2-ASN.mmdb 路徑（AS 號碼與組織） |
| `--mtr-count <n>` | 5 | 每個端口的 TCP MTR 探測次數 |
| `--timeout <duration>` | 5s | 單次診斷總逾時 |
| `--insecure` | false | 跳過 TLS 憑證驗證 |
| `--log-level` | info | 日誌層級：debug / info / warn / error |

---

### `diag web` — Web 與 DNS 診斷

偵測公網 IP（STUN + HTTPS echo 雙來源）、對多組網域執行多 DNS 來源比對。

```powershell
# 基本使用
pathprobe diag web

# 指定網域、記錄型態，輸出 JSON
pathprobe diag web --dns-domain example.com,google.com --dns-type A,MX --json

# 帶 Geo 標註輸出 HTML 報告
pathprobe diag web --geo-db-city ./GeoLite2-City.mmdb --geo-db-asn ./GeoLite2-ASN.mmdb --report ./report.html
```

| 專屬旗標 | 預設值 | 說明 |
|----------|--------|------|
| `--dns-domain` | example.com | 要比對的網域，逗號分隔 |
| `--dns-type` | A,AAAA,MX | 記錄型態，逗號分隔 |

---

### `diag smtp` — SMTP 郵件伺服器診斷

探測 SMTP 握手、EHLO 能力協商、STARTTLS 升級、AUTH (PLAIN / LOGIN) 流程。

```powershell
pathprobe diag smtp --host mail.example.com --port 587 \
    --smtp-domain example.com \
    --smtp-user user@example.com --smtp-pass secret \
    --smtp-starttls
```

| 專屬旗標 | 說明 |
|----------|------|
| `--host` | SMTP 伺服器位址 |
| `--port` | 連接埠（預設 25,465,587） |
| `--smtp-domain` | EHLO 使用的網域名稱 |
| `--smtp-user / --smtp-pass` | AUTH 認證帳密 |
| `--smtp-starttls` | 啟用 STARTTLS |
| `--smtp-tls` | 隱式 TLS（SMTPS） |
| `--smtp-from / --smtp-to` | 測試信封（MAIL FROM / RCPT TO） |

---

### `diag ftp` — FTP / FTPS 診斷

探測 FTP 控制連線、Banner、隱式 FTPS 或 AUTH TLS 顯式加密、PASV LIST 目錄列舉。

```powershell
pathprobe diag ftp --host ftp.example.com --port 21 \
    --ftp-user ftpuser --ftp-pass secret \
    --ftp-auth-tls --ftp-list
```

| 專屬旗標 | 說明 |
|----------|------|
| `--ftp-user / --ftp-pass` | 帳密（省略則 anonymous） |
| `--ftp-tls` | 隱式 FTPS（port 990） |
| `--ftp-auth-tls` | 顯式 FTPS（AUTH TLS） |
| `--ftp-list` | 嘗試 PASV + LIST |

---

### `diag sftp` — SSH / SFTP 診斷

探測 SSH 握手演算法（KEX / HostKey / Cipher / MAC）、密碼或私鑰認證、遠端目錄列舉。

```powershell
pathprobe diag sftp --host sftp.example.com --port 22 \
    --sftp-user sftpuser --sftp-pass secret --sftp-ls

# 使用私鑰
pathprobe diag sftp --host sftp.example.com \
    --sftp-user deploy --sftp-key ~/.ssh/id_ed25519 --sftp-ls
```

| 專屬旗標 | 說明 |
|----------|------|
| `--sftp-user / --sftp-pass` | 帳密認證 |
| `--sftp-key <path>` | PEM 私鑰路徑（公鑰認證） |
| `--sftp-ls` | 嘗試列出遠端預設目錄 |

---

### `diag imap` / `diag pop` — 連線可達性

目前執行 TCP 多點端口可達性探測（IMAPv4: 143/993，POP3: 110/995）。

---

## 報告輸出

執行任何 `diag` 子命令時，加上以下旗標即可產生對應格式輸出：

| 旗標 | 輸出格式 | 說明 |
|------|----------|------|
| `--json` | JSON | 結構化結果，適合管線處理或自動化 |
| _(預設)_ | CLI 表格 | `PORT CONNECTIVITY` + `PROTOCOL RESULTS` 兩張表格 |
| `--report <path>` | HTML | 自包含 HTML 檔（內嵌 Leaflet 1.9.4 + OSM 互動地圖） |

HTML 報告包含：
- 本機公網 IP 與目標主機雙點地圖標記 + 路徑連線
- 端口統計表格（loss% / min/avg/max RTT）
- 協定探測結果表格
- 無 Geo 資料時自動降級為純表格（不顯示地圖）

---

## 測試

```powershell
go test -count=1 ./...
```

主要測試涵蓋範圍：

| 套件 | 測試重點 |
|------|----------|
| `pkg/netprobe` | TCP 端口探測、DNS 解析、DoH、HTTP 探測、SMTP StartTLS / AUTH、FTP Banner / AUTH TLS / 匿名登入、SFTP 握手 |
| `pkg/diag` | Runner 編排、DiagReport nil-safe 方法、全域選項驗證 |
| `pkg/geo` | NoopLocator 降級行為、DB 路徑驗證 |
| `pkg/report` | AnnotatedReport 建構、TableWriter / JSONWriter / HTMLWriter 輸出正確性 |
| `pkg/syscheck` | ICMPChecker 介面、RawICMPChecker 不 panic/契約一致性 |
| `pkg/version` | Version / BuildTime 變數非空斷言 |
| `pkg/cli` | 旗標傳遞、版本命令輸出 |

---

## 開發指引

- 遵守 SOLID 原則；所有外部依賴以介面注入，便於測試替換。
- 新增 Runner 時須實作 `Runner` 介面並向 `Dispatcher` 註冊；以 `req.Report.AddProto()` 等 nil-safe 方法寫入結構化結果。
- 新增功能同步補齊單元測試，並執行 `go test -count=1 ./...` 確認全部通過。
- 不使用 CGO（`CGO_ENABLED=0`），確保跨平台靜態連結。
