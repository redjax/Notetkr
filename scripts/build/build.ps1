[CmdletBinding()]
Param(
    [Parameter(Mandatory=$false, HelpMessage="Name for the binary output")]
    [string]$BinName = "nt",

    [Parameter(Mandatory=$false, HelpMessage="Target OS to build for")]
    [ValidateSet("windows", "linux", "darwin")]
    [string]$BuildOS = "windows",

    [Parameter(Mandatory=$false, HelpMessage="Target CPU architecture to build for")]
    [ValidateSet("amd64", "arm64", "386")]
    [string]$BuildArch = "amd64",

    [Parameter(Mandatory=$false, HelpMessage="Path to binary output directory")]
    [string]$BuildOutputDir = "dist\",

    [Parameter(Mandatory=$false, HelpMessage="Path to module entrypoint")]
    [string]$BuildTarget = ".\cmd\entrypoints\main.go",

    [Parameter(Mandatory=$false, HelpMessage="Print help and exit")]
    [switch]$Help
)

function Print-Help {
    Write-Host "-- | Build notetkr Go module | --" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "[ Parameters ]" -ForegroundColor Yellow
    Write-Host "  -Help: Print this help menu"
    Write-Host ""
    Write-Host "  -BinName (default: nt): Name for the binary output"
    Write-Host "  -BuildOS (default: windows): Target OS to build for [windows, linux, darwin]"
    Write-Host "  -BuildArch (default: amd64): Target CPU architecture to build for [amd64, arm64, 386]"
    Write-Host "  -BuildOutputDir (default: dist\): Path to binary output directory"
    Write-Host "  -BuildTarget (default: .\cmd\entrypoints\main.go): Path to module entrypoint"
}

if ($Help) {
    Print-Help
    exit 0
}

## Ensure Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go is not installed"
    exit 1
}

## Debug output
Write-Host "BinName: $BinName" -ForegroundColor Cyan
Write-Host "BuildOS: $BuildOS" -ForegroundColor Cyan
Write-Host "BuildArch: $BuildArch" -ForegroundColor Cyan
Write-Host "BuildOutputDir: $BuildOutputDir" -ForegroundColor Cyan
Write-Host "BuildTarget: $BuildTarget" -ForegroundColor Cyan

## Append .exe if building for Windows and BinName doesn't end with .exe
if ($BuildOS -eq "windows" -and $BinName -notmatch '\.exe$') {
    Write-Warning "Building for Windows but bin name does not end with '.exe'. Appending .exe to '$BinName'"
    $BinName = "$BinName.exe"
}

## Set environment variables for Go build
$env:GOOS = $BuildOS
$env:GOARCH = $BuildArch

## Auto-populate version info
try {
    $GitVersion = git describe --tags --always 2>$null
    if ($LASTEXITCODE -ne 0) { $GitVersion = "dev" }
} catch {
    $GitVersion = "dev"
}

try {
    $GitCommit = git rev-parse --short HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $GitCommit = "none" }
} catch {
    $GitCommit = "none"
}

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

## Build -ldflags string
$LD_FLAGS = "-s -w " +
    "-X 'github.com/redjax/notetkr/internal/version.Version=$GitVersion' " +
    "-X 'github.com/redjax/notetkr/internal/version.Commit=$GitCommit' " +
    "-X 'github.com/redjax/notetkr/internal/version.Date=$BuildDate'"

## Ensure output directory exists
if (-not (Test-Path $BuildOutputDir)) {
    New-Item -ItemType Directory -Path $BuildOutputDir -Force | Out-Null
}

$BuildOutput = Join-Path $BuildOutputDir.TrimEnd('\') $BinName

Write-Host "Building $BuildTarget, outputting to $BuildOutput" -ForegroundColor Green
Write-Host "Version: $GitVersion" -ForegroundColor Cyan
Write-Host "Commit: $GitCommit" -ForegroundColor Cyan
Write-Host "Date: $BuildDate" -ForegroundColor Cyan
Write-Host "-- [ Build start" -ForegroundColor Yellow

# Run go build
$buildCmd = "go build -ldflags=`"$LD_FLAGS`" -o `"$BuildOutput`" `"$BuildTarget`""
Invoke-Expression $buildCmd

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful" -ForegroundColor Green
    Write-Host "-- [ Build complete" -ForegroundColor Yellow
} else {
    Write-Error "Error building app."
    exit $LASTEXITCODE
}

exit 0
