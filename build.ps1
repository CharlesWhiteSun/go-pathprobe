param(
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

Write-Host "==> Ensuring dependencies" -ForegroundColor Cyan
& go mod tidy

if (-not $SkipTests) {
    Write-Host "==> Running tests" -ForegroundColor Cyan
    & go test ./...
}

Write-Host "==> Building binary" -ForegroundColor Cyan
& go build -ldflags "-s -w" -o ./bin/pathprobe.exe ./cmd/pathprobe

Write-Host "Build succeeded" -ForegroundColor Green
