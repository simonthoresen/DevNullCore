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

# Create desktop shortcut
$desktop  = [Environment]::GetFolderPath("Desktop")
$lnkPath  = Join-Path $desktop "NullSpace.lnk"
$startPs1 = Join-Path $dest "start.ps1"

$shell    = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($lnkPath)
$shortcut.TargetPath       = "powershell.exe"
$shortcut.Arguments        = "-ExecutionPolicy Bypass -File `"$startPs1`""
$shortcut.WorkingDirectory = $dest
$shortcut.Description      = "Start the null-space server"
$shortcut.Save()

Write-Host ""
Write-Host "Done. Desktop shortcut created: NullSpace"
Write-Host ""
Write-Host "Double-click the shortcut, or run manually:"
Write-Host ""
Write-Host "  cd NullSpace"
Write-Host "  .\start.ps1"
