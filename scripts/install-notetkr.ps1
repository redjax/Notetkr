<#
    .SYNOPSIS
    This script downloads and installs `notetkr` on Windows.

    .DESCRIPTION
    Downloads the latest release of `notetkr` from GitHub, and installs it to the
    current user's %LOCALAPPDATA%\notetkr directory. If `notetkr` is already installed,
    the user will be prompted to download and install again. The script will also
    append the install directory to the PATH environment variable.
    
    For private repositories, set the GITHUB_TOKEN environment variable before running.

    .PARAMETER Auto
    If specified, the script will install `notetkr` without prompting the user.

    .EXAMPLE
    .\install-notetkr.ps1
    
    .EXAMPLE
    $env:GITHUB_TOKEN = "your_token_here"
    .\install-notetkr.ps1 -Auto
#>

[CmdletBinding()]
Param(
    [switch] $Auto
)

## Stop immediately on error
$ErrorActionPreference = 'Stop'

## Set install directory to $env:LOCALAPPDATA\notetkr
$InstallPath = Join-Path $env:LOCALAPPDATA 'notetkr'
if (-not (Test-Path $InstallPath)) {
    New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
}

## Check if 'nt.exe' already exists in PATH or install path
$ExistingNotetkr = Get-Command nt -ErrorAction SilentlyContinue
if ($ExistingNotetkr) {
    if (-not $Auto) {
        $Confirm = Read-Host "notetkr is already installed at $($ExistingNotetkr.Path). Download and install again? (y/N)"
        if (-not ( $Confirm -in @('y', 'Y', 'yes', 'Yes', 'YES' ) )) {
            Write-Host "Cancelling installation."
            exit 0
        }
    }
}

## Get latest release tag from GitHub
## For private repos, use GITHUB_TOKEN environment variable
$Headers = @{}
if ($env:GITHUB_TOKEN) {
    $Headers['Authorization'] = "token $env:GITHUB_TOKEN"
}

try {
    $ReleaseApi = 'https://api.github.com/repos/redjax/notetkr/releases/latest'
    if ($Headers.Count -gt 0) {
        $Release = Invoke-RestMethod -Uri $ReleaseApi -Headers $Headers -UseBasicParsing
    } else {
        $Release = Invoke-RestMethod -Uri $ReleaseApi -UseBasicParsing
    }
} catch {
    if ($_.Exception.Response.StatusCode -eq 404) {
        Write-Error @"
Error: No releases found for notetkr.
If this is a private repository, set the GITHUB_TOKEN environment variable:
  `$env:GITHUB_TOKEN = "your_github_token"
  .\scripts\install-notetkr.ps1

Or make the repository public, then create a release by running the Release workflow on GitHub.
"@
    } else {
        Write-Error "Failed to fetch latest release info: $($_.Exception.Message)"
    }
    throw $_.Exception
}

$Version = $Release.tag_name.TrimStart('v')
Write-Host "Installing notetkr version $Version"

## Detect CPU architecture
$ArchNorm = $null
try {
    $ArchCode = (Get-CimInstance Win32_Processor | Select-Object -First 1).Architecture
    
    # Convert ArchCode to normalized name (GoReleaser Windows naming)
    switch ($ArchCode) {
        9 { $ArchNorm = 'x86_64' }
        12 { $ArchNorm = 'arm64' }
        default {
            Write-Error "Unsupported architecture code: $ArchCode"
            throw "Unsupported architecture code: $ArchCode"
        }
    }
} catch {
    # Fallback to environment variable
    $EnvArch = $env:PROCESSOR_ARCHITECTURE
    if ($EnvArch -match '^(AMD64|x86_64)$') {
        $ArchNorm = 'x86_64'
    } elseif ($EnvArch -match '^ARM64$') {
        $ArchNorm = 'arm64'
    } else {
        Write-Error "Unsupported architecture: $EnvArch"
        throw "Unsupported architecture: $EnvArch"
    }
}

if (-not $ArchNorm) {
    Write-Error "Failed to detect system architecture"
    throw "Failed to detect system architecture"
}

## Build asset file name (GoReleaser naming: notetkr_Windows_x86_64.zip)
$FileName = "notetkr_Windows_$ArchNorm.zip"

## For private repos, find the asset URL from the release
if ($env:GITHUB_TOKEN) {
    Write-Host "Fetching asset download URL for $FileName..."
    
    # Find the asset with matching name
    $Asset = $Release.assets | Where-Object { $_.name -eq $FileName } | Select-Object -First 1
    
    if (-not $Asset) {
        Write-Error "Error: Could not find asset $FileName in release $($Release.tag_name)"
        throw "Asset not found"
    }
    
    # Use the API URL with Accept header for private repo authentication
    $AssetUrl = $Asset.url
    $DownloadHeaders = @{
        'Authorization' = "token $env:GITHUB_TOKEN"
        'Accept' = 'application/octet-stream'
    }
} else {
    # Public repo - use direct download URL
    $DownloadUrl = "https://github.com/redjax/notetkr/releases/download/$($Release.tag_name)/$FileName"
    $DownloadHeaders = @{}
}

## Create temp folder
$TempDir = Join-Path -Path ([System.IO.Path]::GetTempPath()) -ChildPath ("notetkr_install_" + [Guid]::NewGuid())
Write-Debug "Using temp dir: $($TempDir)"
try {
    New-Item -ItemType Directory -Path $TempDir | Out-Null
} catch {
    Write-Error "Failed to create temp dir: $($_.Exception.Message)"
    throw $_.Exception
}

$ZipPath = Join-Path $TempDir $FileName

Write-Host "Downloading $FileName from GitHub..."
try {
    if ($env:GITHUB_TOKEN) {
        # Download using API URL with authentication
        Invoke-WebRequest -Uri $AssetUrl -Headers $DownloadHeaders -OutFile $ZipPath -UseBasicParsing
    } else {
        # Download using direct URL
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
    }
} catch {
    Write-Error "Failed to download asset: $($_.Exception.Message)"
    Remove-Item -Recurse -Force $TempDir
    throw $_.Exception
}

Write-Host "Extracting package..."
try {
    Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force
} catch {
    Write-Error "Failed to extract archive: $_"
    Remove-Item -Recurse -Force $TempDir
    throw $_.Exception
}

## Expect binary named "nt.exe" in root of zip
$NotetkrExePath = Join-Path $TempDir 'nt.exe'
if (-not (Test-Path $NotetkrExePath)) {
    Write-Error "Extracted package missing expected nt.exe"
    Remove-Item -Recurse -Force $TempDir
    return
}

## Copy executable to install path
$DestExePath = Join-Path $InstallPath 'nt.exe'
try {
    Copy-Item -Path $NotetkrExePath -Destination $DestExePath -Force
} catch {
    Write-Error "Failed to install nt.exe: $($_.Exception.Message)"
    Remove-Item -Recurse -Force $TempDir
    throw $_.Exception
}

Remove-Item -Recurse -Force $TempDir

Write-Host "`nnotetkr installed successfully to $DestExePath"

## Add to PATH if not present
$UserPath = [Environment]::GetEnvironmentVariable('PATH', [EnvironmentVariableTarget]::User)

if ( -not ( $UserPath -split ';' | Where-Object { $_ -eq $InstallPath } ) ) {
    try {
        [Environment]::SetEnvironmentVariable('PATH', "$UserPath;$InstallPath", 'User')

        Write-Host "Added '$InstallPath' to user PATH environment variable. Close and reopen your shell for changes to take effect."
    } catch {
        Write-Error "Failed to update PATH environment variable: $($_.Exception.Message)"
        
        Write-Warning @"
'$InstallPath' is not in your user PATH environment variable."

"Add it by running this once in PowerShell:"
    $> [Environment]::SetEnvironmentVariable('PATH', "`$UserPath;`$InstallPath", 'User')

Then close & re-open your shell for changes to take effect.
"@

        throw $_.Exception
    }
}
