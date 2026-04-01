# null-space installer
# Usage: irm https://github.com/simonthoresen/null-space/raw/main/install.ps1 | iex
#    or: save this file and run it directly

param(
    [string]$InstallDir = (Join-Path $PWD "null-space")
)

$repo = "simonthoresen/null-space"
$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  null-space installer" -ForegroundColor Cyan
Write-Host ""

# Find the latest release
Write-Host "  Fetching latest release..." -NoNewline
$headers = @{ Accept = "application/vnd.github+json" }
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/tags/latest" -Headers $headers -TimeoutSec 15
$zipAsset = $release.assets | Where-Object { $_.name -eq "null-space.zip" } | Select-Object -First 1
if (-not $zipAsset) { throw "No null-space.zip found in latest release." }
Write-Host " OK" -ForegroundColor Green

# Download the zip
$tempZip = Join-Path ([System.IO.Path]::GetTempPath()) "null-space-install.zip"
Write-Host "  Downloading null-space.zip..." -NoNewline
Invoke-WebRequest -Uri $zipAsset.browser_download_url -OutFile $tempZip -TimeoutSec 120
Write-Host " OK" -ForegroundColor Green

# Extract to temp folder, then merge into install dir (preserves user's custom files)
Write-Host "  Installing to $InstallDir..." -NoNewline
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "null-space-install"
if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
Expand-Archive -Path $tempZip -DestinationPath $tempDir -Force
Get-ChildItem -Path $tempDir -Recurse -File | ForEach-Object {
    $rel  = $_.FullName.Substring($tempDir.Length + 1)
    $dest = Join-Path $InstallDir $rel
    $dir  = Split-Path $dest -Parent
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    Copy-Item -Path $_.FullName -Destination $dest -Force
}
Remove-Item $tempZip -Force
Remove-Item $tempDir -Recurse -Force
New-Item -ItemType Directory -Path (Join-Path $InstallDir "logs") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $InstallDir "state") -Force | Out-Null
Write-Host " OK" -ForegroundColor Green

# Write version stamp
$version = ""
if ($release.body -match 'at ([0-9a-f]{40})') { $version = $Matches[1] }
if ($version) { Set-Content -Path (Join-Path $InstallDir ".version") -Value $version -NoNewline }

# Create desktop shortcuts
$desktop  = [Environment]::GetFolderPath("Desktop")
$startPs1 = Join-Path $InstallDir "start.ps1"
$shell    = New-Object -ComObject WScript.Shell

$public = $shell.CreateShortcut((Join-Path $desktop "NullSpace (public).lnk"))
$public.TargetPath       = "powershell.exe"
$public.Arguments        = "-ExecutionPolicy Bypass -File `"$startPs1`""
$public.WorkingDirectory = $InstallDir
$public.Description      = "Start the null-space server (online multiplayer)"
$public.Save()

$private = $shell.CreateShortcut((Join-Path $desktop "NullSpace (private).lnk"))
$private.TargetPath       = "powershell.exe"
$private.Arguments        = "-ExecutionPolicy Bypass -File `"$startPs1`" --local"
$private.WorkingDirectory = $InstallDir
$private.Description      = "Start null-space in local single-player mode"
$private.Save()

Write-Host ""
Write-Host "  Installed! Desktop shortcuts created." -ForegroundColor Green
Write-Host ""
Write-Host "  To start manually:"
Write-Host "    cd `"$InstallDir`""
Write-Host "    .\start.ps1"
Write-Host ""
