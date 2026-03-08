$ErrorActionPreference = "Stop"

Write-Host "==> Ensuring dependencies" -ForegroundColor Cyan
& go mod tidy

Write-Host "==> Running go vet" -ForegroundColor Cyan
& go vet ./...

Write-Host "==> Building binary" -ForegroundColor Cyan
$version   = (& git describe --tags --abbrev=0 2>$null)
if (-not $version) { $version = "dev" }
$buildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
& go build -ldflags "-s -w -X go-pathprobe/pkg/version.Version=$version -X go-pathprobe/pkg/version.BuildTime=$buildTime" -o ./bin/pathprobe.exe ./cmd/pathprobe

Write-Host "Build succeeded" -ForegroundColor Green
