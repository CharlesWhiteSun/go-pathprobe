# go-pathprobe

本工具為本機網路探勘 CLI，涵蓋公網資訊、DNS 對比，以及後續可擴充的 Web/Mail/FTP/SFTP 診斷骨架。

## 主要里程碑現況
- M0 基礎骨架：Cobra CLI、子命令 `diag web|smtp|imap|pop|ftp|sftp`、共用旗標（json/report/mtr-count/log-level/timeout/insecure）、日誌/驗證框架。
- M1 公網資訊與 DNS 對比：
  - 透過 HTTPS echo 取得 Public IP（預設 `https://api.ipify.org`）。
  - DNS 比對：系統 DNS + DoH (Cloudflare 1.1.1.1, Google 8.8.8.8) 針對 A/AAAA/MX 查詢，回報來源與 RTT。
  - 可自訂比對網域與記錄型態。
- M2 路由探勘與端口可達性（初版）：
  - 依 Target 預設端口（web:443, smtp:25/465/587, imap:143/993, pop:110/995, ftp:21/990, sftp:22），可用 `--port` 覆寫。
  - MTR 風格重試次數沿用 `--mtr-count`，統計 loss/RTT；`--timeout` 控制單次探測逾時。

## 環境需求
- Go 1.25+（專案 go.mod 目前為 1.25.1）
- Windows 環境（範例 build.ps1），其他平台可自行調整 build 指令。

## 安裝/建置
```powershell
# 取得依賴並建置（不會自動跑測試）
./build.ps1

# 若需先跑測試再建置，請手動：
go test ./...
./build.ps1
```

## CLI 使用說明
根命令：`pathprobe`，主要子命令：`diag`。

共用旗標（所有 diag 子命令皆適用）：
- `--json`：輸出 JSON。
- `--report <path>`：指定報告輸出路徑（暫留）。
- `--mtr-count <n>`：Traceroute/MTR 每 hop 探測次數（>0）。
- `--timeout <duration>`：單次診斷總逾時（預設 5s）。
- `--insecure`：跳過 TLS 驗證（風險自負）。
- `--log-level <debug|info|warn|error>`：日誌層級。

### Web 診斷 (`diag web`)
用途：取得公網 IP、對多組 DNS 解析結果比對（偵測可能的汙染/差異）。

專屬旗標：
- `--dns-domain <d1,d2,...>`：指定要比對的網域（預設 `example.com`）。
- `--dns-type <A,AAAA,MX,...>`：指定記錄型態（預設 A,AAAA,MX；支援 A/AAAA/MX）。

範例：
```powershell
# 取得公網 IP，對 example.com 進行 A/AAAA/MX 比對
pathprobe diag web

# 指定網域與記錄型態，並輸出 JSON
pathprobe diag web --dns-domain example.com,google.com --dns-type A,MX --json --timeout 8s --log-level debug
```

### 其他子命令
`smtp|imap|pop|ftp|sftp` 目前為骨架（BasicRunner），將在後續里程碑補齊協定探勘。

## 測試
```powershell
go test ./...
```
涵蓋：
- netprobe：HTTPS Public IP 解析、DoH 解析、DNS 比對。
- diag：WebRunner 編排、全域選項驗證。
- cli：旗標傳遞與驗證（含 timeout/insecure/dns-domain/dns-type）。

## 開發指引
- 遵守 SOLID、避免硬編碼；以介面注入 fetcher/resolver/comparator。
- 新增功能時同步補齊單元/整合測試並執行 `go test ./...`。
- 建置前執行 `./build.ps1`（包含 go vet + build）。

## 後續計畫
- M2 路由探勘與端口可達性：ICMP/TCP traceroute、MTR 統計、Port reachability。
- M3+ 協定層探勘（Web/Mail/FTP/SFTP）、Geo 標註與報告輸出、封裝發佈。
