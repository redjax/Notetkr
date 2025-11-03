if ( -not ( Get-Command goreleaser -ErrorAction SilentlyContinue ) ) {
    Write-Error "Goreleaser is not installed"
    exit 1
}

Write-Host "Building local/development version of app with goreleaser" -ForegroundColor Cyan
goreleaser build --single-target --clean --snapshot --output ./dist/nt.exe
if ($LASTEXITCODE -ne 0) {
    Write-Error "Goreleaser build failed"
    exit $LASTEXITCODE
}

Write-Host "Build completed successfully. Output located at ./dist/nt.exe" -ForegroundColor Green
