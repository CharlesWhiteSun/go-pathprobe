$ErrorActionPreference = "Stop"

Write-Host "==> Ensuring dependencies" -ForegroundColor Cyan
& go mod tidy

Write-Host "==> Running go vet" -ForegroundColor Cyan
& go vet ./...

Write-Host "==> Building binary" -ForegroundColor Cyan

# Resolve the most recent tag (falls back to "dev" if no tags exist).
$tag = (& git describe --tags --abbrev=0 2>$null)
if (-not $tag) { $tag = "dev" }

# Resolve the abbreviated commit hash (6 characters).
$hash = (& git rev-parse --short=6 HEAD 2>$null)
if (-not $hash) { $hash = "unknown" }

# Mark as dirty when there are uncommitted changes OR unpushed commits.
$dirtyFiles = (& git status --porcelain 2>$null)
$hasUncommitted = ($null -ne $dirtyFiles -and $dirtyFiles.Trim() -ne '')

$hasUnpushed = $false
$unpushed = (& git log "@{upstream}..HEAD" --oneline 2>$null)
if ($LASTEXITCODE -eq 0 -and $null -ne $unpushed -and $unpushed.Trim() -ne '') {
    $hasUnpushed = $true
}

if ($hasUncommitted -or $hasUnpushed) {
    $version = "$tag-$hash-dirty"
} else {
    $version = "$tag-$hash"
}

Write-Host "    version = $version" -ForegroundColor DarkGray

& go build -ldflags "-s -w -X go-pathprobe/pkg/version.Version=$version" -o ./bin/pathprobe.exe ./cmd/pathprobe

Write-Host "Build succeeded" -ForegroundColor Green
