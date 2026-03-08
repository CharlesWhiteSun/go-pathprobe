$ErrorActionPreference = "Stop"

Write-Host "==> Ensuring dependencies" -ForegroundColor Cyan
& go mod tidy

Write-Host "==> Running go vet" -ForegroundColor Cyan
& go vet ./...

Write-Host "==> Building binary" -ForegroundColor Cyan
& go build -ldflags "-s -w" -o ./bin/pathprobe.exe ./cmd/pathprobe

Write-Host "Build succeeded" -ForegroundColor Green
