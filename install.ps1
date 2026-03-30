$ErrorActionPreference = 'Stop'

$repo  = "https://raw.githubusercontent.com/simonthoresen/null-space/main"
$dest  = Join-Path $PWD "NullSpace"

$files = @(
    "dist/null-space.exe",
    "dist/pinggy-helper.exe",
    "dist/start.ps1",
    "dist/games/example.js",
    "dist/plugins/profanity-filter.js"
)

Write-Host "Installing null-space to $dest"
Write-Host ""

foreach ($f in $files) {
    $rel    = $f -replace '^dist/', ''
    $target = Join-Path $dest $rel
    $dir    = Split-Path $target -Parent
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }
    Write-Host "  $rel"
    Invoke-WebRequest -UseBasicParsing -Uri "$repo/$f" -OutFile $target
}

New-Item -ItemType Directory -Force -Path (Join-Path $dest "logs") | Out-Null

# Create desktop shortcuts
$desktop  = [Environment]::GetFolderPath("Desktop")
$startPs1 = Join-Path $dest "start.ps1"
$shell    = New-Object -ComObject WScript.Shell

$public = $shell.CreateShortcut((Join-Path $desktop "NullSpace (public).lnk"))
$public.TargetPath       = "powershell.exe"
$public.Arguments        = "-ExecutionPolicy Bypass -File `"$startPs1`""
$public.WorkingDirectory = $dest
$public.Description      = "Start the null-space server (online multiplayer)"
$public.Save()

$private = $shell.CreateShortcut((Join-Path $desktop "NullSpace (private).lnk"))
$private.TargetPath       = "powershell.exe"
$private.Arguments        = "-ExecutionPolicy Bypass -File `"$startPs1`" --local"
$private.WorkingDirectory = $dest
$private.Description      = "Start null-space in local single-player mode"
$private.Save()

Write-Host ""
Write-Host "Done. Desktop shortcuts created: NullSpace (public), NullSpace (private)"
Write-Host ""
Write-Host "Double-click the shortcut, or run manually:"
Write-Host ""
Write-Host "  cd `"$dest`""
Write-Host "  .\start.ps1"
