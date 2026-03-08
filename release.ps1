#Requires -Version 5.1
<#
.SYNOPSIS
    Build pathrobe release binaries with cross-compilation support.

.DESCRIPTION
    Produces stripped, version-stamped binaries for Windows/Linux/macOS.
    Optionally embeds a requireAdministrator Windows manifest using the
    'rsrc' tool (go install github.com/akavel/rsrc@latest).

.PARAMETER Version
    Version tag string (e.g. v1.3.0).  Defaults to the latest annotated git
    tag, or "dev" if no tags exist.

.PARAMETER Targets
    Comma-separated list of GOOS/GOARCH pairs to build.
    Default: "windows/amd64,linux/amd64,darwin/amd64,darwin/arm64"

.PARAMETER WithManifest
    When specified, attempts to embed the requireAdministrator manifest into
    the Windows amd64 binary using the 'rsrc' tool.

.EXAMPLE
    .\release.ps1 -Version v1.3.0 -WithManifest
    .\release.ps1 -Targets "windows/amd64,linux/amd64"
#>
param(
    [string]$Version    = "",
    [string]$Targets    = "windows/amd64,linux/amd64,darwin/amd64,darwin/arm64",
    [switch]$WithManifest
)

$ErrorActionPreference = "Stop"

$BinDir       = "./bin"
$ManifestFile = "./cmd/pathprobe/pathprobe.manifest"
$SysoFile     = "./cmd/pathprobe/pathprobe.syso"
$MainPkg      = "./cmd/pathprobe"

# ---------------------------------------------------------------------------
# Resolve version string
# ---------------------------------------------------------------------------
if (-not $Version) {
    $tag = & git describe --tags --abbrev=0 2>$null
    $Version = if ($LASTEXITCODE -eq 0 -and $tag) { $tag.Trim() } else { "dev" }
}
$BuildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

Write-Host "==> PathProbe Release Builder" -ForegroundColor Cyan
Write-Host "    Version   : $Version"
Write-Host "    BuildTime : $BuildTime"
Write-Host "    Targets   : $Targets"
Write-Host ""

# ---------------------------------------------------------------------------
# Prepare output directory
# ---------------------------------------------------------------------------
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

# ---------------------------------------------------------------------------
# Quality gates
# ---------------------------------------------------------------------------
Write-Host "==> Ensuring dependencies" -ForegroundColor Cyan
& go mod tidy
if ($LASTEXITCODE -ne 0) { throw "go mod tidy failed" }

Write-Host "==> Running go vet" -ForegroundColor Cyan
& go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

Write-Host "==> Running tests" -ForegroundColor Cyan
& go test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw "tests failed" }

# ---------------------------------------------------------------------------
# Optional manifest embedding (Windows only)
# ---------------------------------------------------------------------------
$manifestEmbedded = $false
if ($WithManifest) {
    if (-not (Test-Path $ManifestFile)) {
        Write-Warning "Manifest file not found: $ManifestFile  (skipping)"
    } elseif (Get-Command rsrc -ErrorAction SilentlyContinue) {
        Write-Host "==> Embedding requireAdministrator manifest" -ForegroundColor Cyan
        & rsrc -manifest $ManifestFile -o $SysoFile
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "rsrc failed; proceeding without manifest"
        } else {
            $manifestEmbedded = $true
            Write-Host "    Manifest embedded into: $SysoFile"
        }
    } else {
        Write-Warning "-WithManifest specified but 'rsrc' is not on PATH."
        Write-Warning "Install with:  go install github.com/akavel/rsrc@latest"
        Write-Warning "Proceeding without manifest embedding."
    }
}

# ---------------------------------------------------------------------------
# Build common ldflags
# ---------------------------------------------------------------------------
$ldflags = "-s -w -X go-pathprobe/pkg/version.Version=$Version -X `"go-pathprobe/pkg/version.BuildTime=$BuildTime`""

# ---------------------------------------------------------------------------
# Cross-compile
# ---------------------------------------------------------------------------
$targetList = $Targets -split ","
foreach ($pair in $targetList) {
    $parts = $pair.Trim() -split "/"
    if ($parts.Length -ne 2) {
        Write-Warning "Skipping invalid target '$pair' (expected GOOS/GOARCH)"
        continue
    }
    $goos   = $parts[0]
    $goarch = $parts[1]

    $ext    = if ($goos -eq "windows") { ".exe" } else { "" }
    $outName = "pathprobe-$goos-$goarch$ext"
    $outPath = "$BinDir/$outName"

    Write-Host "==> Building $goos/$goarch  =>  $outPath" -ForegroundColor Cyan
    $env:GOOS   = $goos
    $env:GOARCH = $goarch
    $env:CGO_ENABLED = "0"

    & go build -ldflags $ldflags -o $outPath $MainPkg
    if ($LASTEXITCODE -ne 0) {
        $env:GOOS        = ""
        $env:GOARCH      = ""
        $env:CGO_ENABLED = ""
        throw "Build failed for $goos/$goarch"
    }
}

# Restore environment.
$env:GOOS        = ""
$env:GOARCH      = ""
$env:CGO_ENABLED = ""

# ---------------------------------------------------------------------------
# Clean up temporary syso (only needed for Windows resource compilation)
# ---------------------------------------------------------------------------
if ($manifestEmbedded -and (Test-Path $SysoFile)) {
    Remove-Item $SysoFile -ErrorAction SilentlyContinue
    Write-Host "    Cleaned up temporary $SysoFile"
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host "==> Release binaries:" -ForegroundColor Green
Get-ChildItem $BinDir -File | Where-Object { $_.Extension -ne ".syso" } |
    Select-Object Name, @{Name="Size (KB)"; Expression={[math]::Round($_.Length/1KB,1)}} |
    Format-Table -AutoSize

Write-Host "Release build succeeded  ($Version)" -ForegroundColor Green
