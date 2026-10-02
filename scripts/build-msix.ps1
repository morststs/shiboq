# Builds an unsigned MSIX package for Microsoft Store submission.
#
#   pwsh scripts/build-msix.ps1 -Version 1.0.0
#
# Requires Windows with the Windows SDK (MakeAppx.exe) and an already built
# build/bin/shiboq.exe (wails build -platform windows/amd64).
# The package is intentionally NOT signed: the Store re-signs it after certification.
#
# This file is ASCII-only on purpose. Windows PowerShell 5.1 reads BOM-less
# scripts in the system ANSI code page, so non-ASCII text here could break parsing.
param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$ExePath = 'build/bin/shiboq.exe',
    [string]$OutPath = 'build/bin/shiboq.msix'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# --- Version: X.Y.Z (optional leading "v") -> X.Y.Z.0 --------------------
# Store rules: the 4th part is reserved for the Store and must be 0,
# and the 1st part must not be 0.
if ($Version -notmatch '^v?(\d+)\.(\d+)\.(\d+)$') {
    throw "Version must look like 1.2.3 (got '$Version')."
}
$parts = @([int]$Matches[1], [int]$Matches[2], [int]$Matches[3])
foreach ($p in $parts) {
    if ($p -gt 65535) { throw "Each version part must be 0-65535 (got '$Version')." }
}
if ($parts[0] -eq 0) {
    throw "The Microsoft Store does not accept a major version of 0 (got '$Version'). Use 1.0.0 or higher."
}
$msixVersion = '{0}.{1}.{2}.0' -f $parts[0], $parts[1], $parts[2]

# --- Inputs --------------------------------------------------------------
if (-not (Test-Path $ExePath)) {
    throw "$ExePath not found. Run 'wails build -platform windows/amd64' first."
}
$manifestSrc = Join-Path $root 'build/msix/AppxManifest.xml'
$iconSrc = Join-Path $root 'build/appicon.png'

# --- Locate MakeAppx.exe (PATH first, then the newest installed Windows SDK)
$makeAppx = $null
$cmd = Get-Command makeappx.exe -ErrorAction SilentlyContinue
if ($cmd) {
    $makeAppx = $cmd.Source
} else {
    $kits = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin'
    if (Test-Path $kits) {
        $found = Get-ChildItem -Path $kits -Directory |
            Where-Object { $_.Name -match '^\d+\.\d+\.\d+\.\d+$' } |
            Sort-Object { [version]$_.Name } -Descending |
            ForEach-Object { Join-Path $_.FullName 'x64\makeappx.exe' } |
            Where-Object { Test-Path $_ } |
            Select-Object -First 1
        if ($found) { $makeAppx = $found }
    }
}
if (-not $makeAppx) {
    throw 'MakeAppx.exe not found. Install the Windows SDK.'
}

# --- Manifest: reject placeholders, set the version -----------------------
$manifest = New-Object System.Xml.XmlDocument
$manifest.PreserveWhitespace = $true
$manifest.Load($manifestSrc)
$identity = $manifest.Package.Identity
$identityValues = @(
    $identity.GetAttribute('Name'),
    $identity.GetAttribute('Publisher'),
    [string]$manifest.Package.Properties.PublisherDisplayName
)
if ($identityValues -match 'REPLACE_WITH_') {
    throw "build/msix/AppxManifest.xml still contains REPLACE_WITH_ placeholders. Fill in the values from Partner Center (Product identity) first."
}
$identity.SetAttribute('Version', $msixVersion)

# --- Stage the package contents ---------------------------------------------
$staging = Join-Path $root 'build/msix/staging'
if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }
$assets = Join-Path $staging 'Assets'
New-Item -ItemType Directory -Force -Path $assets | Out-Null

Copy-Item $ExePath (Join-Path $staging 'shiboq.exe')
# MIT / BSD / Apache all require keeping the copyright notices when redistributing.
Copy-Item 'LICENSE' $staging
Copy-Item 'THIRD_PARTY_LICENSES.md' $staging
$manifest.Save((Join-Path $staging 'AppxManifest.xml'))

# --- Logos: resized from build/appicon.png so they never drift from the app icon
Add-Type -AssemblyName System.Drawing
function Save-ResizedPng([string]$Source, [string]$Destination, [int]$Size) {
    $src = [System.Drawing.Image]::FromFile($Source)
    try {
        $bmp = New-Object System.Drawing.Bitmap($Size, $Size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
        try {
            $g = [System.Drawing.Graphics]::FromImage($bmp)
            try {
                $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
                $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                $g.Clear([System.Drawing.Color]::Transparent)
                $g.DrawImage($src, 0, 0, $Size, $Size)
            } finally { $g.Dispose() }
            $bmp.Save($Destination, [System.Drawing.Imaging.ImageFormat]::Png)
        } finally { $bmp.Dispose() }
    } finally { $src.Dispose() }
}
Save-ResizedPng $iconSrc (Join-Path $assets 'Square44x44Logo.png') 44
Save-ResizedPng $iconSrc (Join-Path $assets 'Square150x150Logo.png') 150
Save-ResizedPng $iconSrc (Join-Path $assets 'StoreLogo.png') 50

# --- Pack ----------------------------------------------------------------
$outFull = Join-Path $root $OutPath
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $outFull) | Out-Null
& $makeAppx pack /o /d $staging /p $outFull
if ($LASTEXITCODE -ne 0) {
    throw "MakeAppx failed with exit code $LASTEXITCODE."
}
Remove-Item -Recurse -Force $staging

Write-Host "Built $OutPath (version $msixVersion, unsigned)."
