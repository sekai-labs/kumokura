<#
.SYNOPSIS
    Automated installer script for Kumokura on Windows.
.DESCRIPTION
    Downloads the latest release archive of Kumokura (or specified version),
    extracts binaries to %LOCALAPPDATA%\Programs\Kumokura, adds the directory to
    User PATH, and creates a Start Menu shortcut for the Desktop GUI.
.PARAMETER Version
    Specific release version tag (e.g. "v0.1.0"). Default is "latest".
.PARAMETER InstallDir
    Target directory for binaries. Default: "$env:LOCALAPPDATA\Programs\Kumokura".
.PARAMETER NoShortcut
    Switch to disable creating Start Menu shortcut.
.PARAMETER Uninstall
    Switch to remove Kumokura and cleanup PATH and shortcuts.
#>
[CmdletBinding()]
param (
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\Kumokura",
    [switch]$NoShortcut,
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"
$Repo = "sekai-labs/kumokura"

function Write-Step {
    param([string]$Message)
    Write-Host "[Kumokura] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[Kumokura] $Message" -ForegroundColor Green
}

function Write-Err {
    param([string]$Message)
    Write-Host "[Kumokura ERROR] $Message" -ForegroundColor Red
}

# ----------------- UNINSTALLATION -----------------
if ($Uninstall) {
    Write-Step "Uninstalling Kumokura..."

    # 1. Remove Start Menu shortcut
    $ShortcutPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Kumokura.lnk"
    if (Test-Path $ShortcutPath) {
        Remove-Item -Path $ShortcutPath -Force
        Write-Step "Removed Start Menu shortcut."
    }

    # 2. Remove Install directory
    if (Test-Path $InstallDir) {
        Remove-Item -Path $InstallDir -Recurse -Force
        Write-Step "Removed $InstallDir."
    }

    # 3. Remove from User PATH
    $UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
    if ($UserPath -like "*$InstallDir*") {
        $NewPath = ($UserPath -split ';' | Where-Object { $_ -ne $InstallDir -and $_ -ne "" }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $NewPath, [EnvironmentVariableTarget]::User)
        Write-Step "Removed $InstallDir from User PATH."
    }

    Write-Success "Kumokura has been successfully uninstalled."
    return
}

# ----------------- INSTALLATION -----------------
Write-Step "Starting Kumokura Windows installation..."

# 1. Determine release tag and download URL
$Headers = @{ "User-Agent" = "Kumokura-Windows-Installer" }
if ($Version -eq "latest") {
    $ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $ReleaseInfo = Invoke-RestMethod -Uri $ApiUrl -Headers $Headers -UseBasicParsing
        $Tag = $ReleaseInfo.tag_name
    } catch {
        Write-Err "Failed to query latest release from GitHub API: $_"
        exit 1
    }
} else {
    $Tag = $Version
    if (-not $Tag.StartsWith("v")) {
        $Tag = "v$Tag"
    }
    $ApiUrl = "https://api.github.com/repos/$Repo/releases/tags/$Tag"
    try {
        $ReleaseInfo = Invoke-RestMethod -Uri $ApiUrl -Headers $Headers -UseBasicParsing
    } catch {
        Write-Err "Failed to find release $Tag on GitHub: $_"
        exit 1
    }
}

Write-Step "Target release: $Tag"

# 2. Find asset for windows amd64 zip
$Asset = $ReleaseInfo.assets | Where-Object { $_.name -like "*windows_amd64*.zip" } | Select-Object -First 1
if (-not $Asset) {
    # Fallback to standard name construction
    $DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/kumokura_${Tag}_windows_amd64.zip"
} else {
    $DownloadUrl = $Asset.browser_download_url
}

Write-Step "Download URL: $DownloadUrl"

# 3. Download archive to temp directory
$TempZip = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "kumokura_$Tag.zip")
Write-Step "Downloading archive to $TempZip..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing
} catch {
    Write-Err "Download failed: $_"
    exit 1
}

# 4. Extract to target directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Write-Step "Extracting to $InstallDir..."
try {
    Expand-Archive -Path $TempZip -DestinationPath $InstallDir -Force
} catch {
    Write-Err "Extraction failed: $_"
    Remove-Item -Path $TempZip -Force -ErrorAction SilentlyContinue
    exit 1
}

# Cleanup zip
Remove-Item -Path $TempZip -Force -ErrorAction SilentlyContinue

# Verify expected binaries
$ExpectedFiles = @("kumokura.exe", "kumokura-tui.exe", "kumokura-desktop.exe")
foreach ($file in $ExpectedFiles) {
    $fullPath = Join-Path $InstallDir $file
    if (Test-Path $fullPath) {
        Write-Step "Verified binary: $file"
    } else {
        Write-Host "Warning: Expected $file not found in extracted archive." -ForegroundColor Yellow
    }
}

# 5. Add to User PATH if not present
$UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($UserPath -notlike "*$InstallDir*") {
    Write-Step "Adding $InstallDir to User PATH..."
    $NewPath = if ([string]::IsNullOrWhiteSpace($UserPath)) { $InstallDir } else { "$UserPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, [EnvironmentVariableTarget]::User)
    $env:Path = "$env:Path;$InstallDir"
    Write-Success "PATH updated for current session and persistent user profile."
} else {
    Write-Step "$InstallDir already present in User PATH."
}

# 6. Create Start Menu shortcut
if (-not $NoShortcut) {
    $DesktopExe = Join-Path $InstallDir "kumokura-desktop.exe"
    if (Test-Path $DesktopExe) {
        $ProgramsFolder = [System.IO.Path]::Combine($env:APPDATA, "Microsoft\Windows\Start Menu\Programs")
        $ShortcutPath = Join-Path $ProgramsFolder "Kumokura.lnk"
        Write-Step "Creating Start Menu shortcut at $ShortcutPath..."
        try {
            $WScriptShell = New-Object -ComObject WScript.Shell
            $Shortcut = $WScriptShell.CreateShortcut($ShortcutPath)
            $Shortcut.TargetPath = $DesktopExe
            $Shortcut.WorkingDirectory = $InstallDir
            $Shortcut.Description = "Kumokura S3 Storage Manager (Desktop GUI)"
            $Shortcut.Save()
            Write-Success "Shortcut created successfully."
        } catch {
            Write-Host "Warning: Could not create Start Menu shortcut: $_" -ForegroundColor Yellow
        }
    }
}

Write-Success "=========================================================="
Write-Success "Kumokura $Tag installed successfully!"
Write-Success "Location: $InstallDir"
Write-Success "Binaries available:"
Write-Success "  - kumokura         (CLI)"
Write-Success "  - kumokura-tui     (Terminal Dual-Pane UI)"
Write-Success "  - kumokura-desktop (Fyne Desktop GUI)"
Write-Success "=========================================================="
Write-Host "Note: Restart your terminal window to ensure PATH updates take effect." -ForegroundColor Cyan
