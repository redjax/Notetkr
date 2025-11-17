[CmdletBinding()]
Param(
    [Parameter(Mandatory=$false, HelpMessage="Use goreleaser to build the development version")]
    [switch]$UseGoreleaser
)

## Ensure Go is installed
if ( -not ( Get-Command go -ErrorAction SilentlyContinue ) ) {
    Write-Error "Go is not installed"
    exit 1
}

## Use Goreleaser
if ( $UseGoreleaser ) {
    if ( -not ( Get-Command goreleaser -ErrorAction SilentlyContinue ) ) {
        Write-Error "Goreleaser is not installed"
        exit 1
    }

    Write-Host "Using goreleaser to build development version" -ForegroundColor Cyan

    Write-Host "Building local/development version of app with goreleaser" -ForegroundColor Cyan
    goreleaser build --single-target --clean --snapshot --output ./dist/nt.exe
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Goreleaser build failed"
        exit $LASTEXITCODE
    }
## Use Go
} else {
    Write-Host "Building local/development version of app with Go" -ForegroundColor Cyan
    
    # Get version info
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

    $LD_FLAGS = "-s -w " +
        "-X 'github.com/redjax/notetkr/internal/version.Version=$GitVersion' " +
        "-X 'github.com/redjax/notetkr/internal/version.Commit=$GitCommit' " +
        "-X 'github.com/redjax/notetkr/internal/version.Date=$BuildDate'"

    go build -o ./dist/nt.exe -ldflags "$LD_FLAGS" ./cmd/entrypoints/main.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Go build failed"
        exit $LASTEXITCODE
    }
}

Write-Host "Build completed successfully. Output located at ./dist/nt.exe" -ForegroundColor Green
