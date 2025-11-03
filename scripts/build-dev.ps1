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
    go build -o ./dist/nt.exe -ldflags "-X 'main.buildType=development'" ./cmd/entrypoints/main.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Go build failed"
        exit $LASTEXITCODE
    }
}

Write-Host "Build completed successfully. Output located at ./dist/nt.exe" -ForegroundColor Green
