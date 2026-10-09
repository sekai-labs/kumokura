<#
.SYNOPSIS
    Builds Windows MSI Installer for Kumokura using WiX toolset.
.PARAMETER Version
    Version string (e.g. "0.1.0" or "v0.1.0"). Default "0.1.0".
.PARAMETER DistDir
    Directory containing built binaries (kumokura.exe, kumokura-tui.exe, kumokura-desktop.exe).
.PARAMETER OutputDir
    Directory where the generated .msi installer will be placed.
#>
[CmdletBinding()]
param(
    [string]$Version = "0.1.0",
    [string]$DistDir = "dist",
    [string]$OutputDir = "release-artifacts"
)

$ErrorActionPreference = "Stop"

# Strip leading 'v' if present (e.g., "v0.1.0" -> "0.1.0")
$CleanVersion = $Version.TrimStart('v')
# WiX requires version format: major.minor.build (e.g. 1.2.3)
if ($CleanVersion -notmatch '^\d+(\.\d+){1,3}$') {
    $CleanVersion = "0.1.0"
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")

$ResolvedDistDir = if ([System.IO.Path]::IsPathRooted($DistDir)) { $DistDir } else { Join-Path $RepoRoot $DistDir }
$ResolvedOutputDir = if ([System.IO.Path]::IsPathRooted($OutputDir)) { $OutputDir } else { Join-Path $RepoRoot $OutputDir }
$WxsPath = Join-Path $ScriptDir "kumokura.wxs"

if (-not (Test-Path $ResolvedDistDir)) {
    throw "Distribution directory not found: $ResolvedDistDir"
}

# Verify required binaries exist
$RequiredFiles = @("kumokura.exe", "kumokura-tui.exe", "kumokura-desktop.exe")
foreach ($file in $RequiredFiles) {
    $filePath = Join-Path $ResolvedDistDir $file
    if (-not (Test-Path $filePath)) {
        throw "Required binary not found: $filePath"
    }
}

if (-not (Test-Path $ResolvedOutputDir)) {
    New-Item -ItemType Directory -Path $ResolvedOutputDir -Force | Out-Null
}

$MsiFileName = "kumokura_${Version}_windows_amd64.msi"
$MsiOutputPath = Join-Path $ResolvedOutputDir $MsiFileName

Write-Host "[Kumokura MSI] Building MSI installer for version $CleanVersion..." -ForegroundColor Cyan
Write-Host "[Kumokura MSI] Binaries: $ResolvedDistDir" -ForegroundColor Cyan
Write-Host "[Kumokura MSI] Output: $MsiOutputPath" -ForegroundColor Cyan

# Check if WiX v3 (candle / light) or WiX v4 (wix build) is available
$HasCandle = Get-Command candle.exe -ErrorAction SilentlyContinue
$HasLight = Get-Command light.exe -ErrorAction SilentlyContinue
$HasWix = Get-Command wix.exe -ErrorAction SilentlyContinue

if ($HasCandle -and $HasLight) {
    Write-Host "[Kumokura MSI] Using WiX v3 toolset (candle + light)..." -ForegroundColor Green
    $WixObj = Join-Path $ResolvedOutputDir "kumokura.wixobj"

    & candle.exe -nologo `
        "-dVersion=$CleanVersion" `
        "-dSourceDir=$ResolvedDistDir" `
        -arch x64 `
        -out "$WixObj" `
        "$WxsPath"
    if ($LASTEXITCODE -ne 0) { throw "WiX candle failed with exit code $LASTEXITCODE" }

    & light.exe -nologo `
        -ext WixUIExtension `
        -out "$MsiOutputPath" `
        "$WixObj"
    if ($LASTEXITCODE -ne 0) { throw "WiX light failed with exit code $LASTEXITCODE" }

    Remove-Item -Path $WixObj -Force -ErrorAction SilentlyContinue
    $WixPdb = [System.IO.Path]::ChangeExtension($MsiOutputPath, ".wixpdb")
    if (Test-Path $WixPdb) {
        Remove-Item -Path $WixPdb -Force -ErrorAction SilentlyContinue
    }
} elseif ($HasWix) {
    Write-Host "[Kumokura MSI] Using WiX v4+ toolset (wix build)..." -ForegroundColor Green
    & wix.exe build `
        "-d" "Version=$CleanVersion" `
        "-d" "SourceDir=$ResolvedDistDir" `
        -arch x64 `
        -o "$MsiOutputPath" `
        "$WxsPath"
    if ($LASTEXITCODE -ne 0) { throw "WiX build failed with exit code $LASTEXITCODE" }
} else {
    throw "Neither WiX v3 (candle.exe/light.exe) nor WiX v4+ (wix.exe) was found in PATH."
}

if (-not (Test-Path $MsiOutputPath)) {
    throw "Expected MSI output file was not created: $MsiOutputPath"
}

Write-Host "[Kumokura MSI] Successfully created MSI installer: $MsiOutputPath" -ForegroundColor Green
