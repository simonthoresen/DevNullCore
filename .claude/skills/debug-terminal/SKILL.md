---
name: debug-terminal
description: >
  Run the dev-null testbed binary in a Windows Terminal tab, take a
  screenshot mid-run, and kill the tab cleanly afterward. Use when
  debugging SSH rendering artifacts (staircase, color, delta rendering)
  in the dev-null project. Covers testbed setup, WT profile config,
  screenshot timing, cleanup, and interpreting results.
compatibility: Windows 11 + Windows Terminal. Requires dist/testbed.exe built via `go build -o dist/testbed.exe ./testbed/`.
metadata:
  author: dev-null project
  version: "1.0"
---

# debug-terminal

Run the `testbed/` binary inside a real Windows Terminal tab and capture
a screenshot for visual artifact analysis. All learnings from past
debugging sessions are encoded here.

## Prerequisites

### 1. Build the testbed

```bash
go build -o dist/testbed.exe ./testbed/
```

### 2. Add the `testbed-autoclose` WT profile (one-time setup)

WT does not auto-close tabs when a process exits by default. Without this
profile, stale tabs accumulate and pollute every screenshot.

Add to `%LOCALAPPDATA%\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json`,
inside `profiles.list`:

```json
{
    "closeOnExit": "always",
    "commandline": "%SystemRoot%\\System32\\cmd.exe",
    "guid": "{a1b2c3d4-e5f6-7890-abcd-ef1234567890}",
    "hidden": true,
    "name": "testbed-autoclose"
}
```

With `closeOnExit: always`, the tab disappears the moment the process
exits — no "You can now close this terminal" banner, no stale tabs.

## Core PowerShell helper

Paste this in a PowerShell session before running tests:

```powershell
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32 {
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@

$testbedPath = 'C:\Users\simonhul\Source\dev-null\dist\testbed.exe'

function BringWTToFront {
    $wt = Get-Process -Name WindowsTerminal -ErrorAction SilentlyContinue |
          Select-Object -First 1
    if ($wt -and $wt.MainWindowHandle -ne [IntPtr]::Zero) {
        [Win32]::ShowWindow($wt.MainWindowHandle, 9) | Out-Null  # SW_RESTORE
        [Win32]::SetForegroundWindow($wt.MainWindowHandle) | Out-Null
    }
}

function RunAndShot($title, $tbArgs, $outFile) {
    # Kill stale testbed before starting
    Get-Process -Name testbed -ErrorAction SilentlyContinue | Stop-Process -Force

    # Open new tab using the autoclose profile
    Start-Process 'wt.exe' -ArgumentList '--window', '0', 'new-tab',
        '--profile', 'testbed-autoclose',
        '--title', $title,
        'cmd', '/c', "`"$testbedPath`" $tbArgs"

    # Bring WT to front EARLY (before bubbletea enters alt screen)
    # If you wait until after alt-screen entry you may catch the blank flash.
    Start-Sleep -Milliseconds 300
    BringWTToFront

    # Wait for SSH handshake + first rendered frame (~1-1.5s total from launch)
    Start-Sleep -Milliseconds 1000

    # Screenshot full primary screen
    $screen = [System.Windows.Forms.Screen]::PrimaryScreen
    $bmp = New-Object System.Drawing.Bitmap($screen.Bounds.Width, $screen.Bounds.Height)
    $gfx = [System.Drawing.Graphics]::FromImage($bmp)
    $gfx.CopyFromScreen($screen.Bounds.Location, [System.Drawing.Point]::Empty,
                         $screen.Bounds.Size)
    $bmp.Save($outFile)
    $gfx.Dispose(); $bmp.Dispose()

    # Kill testbed — cmd.exe exits — tab auto-closes (closeOnExit: always)
    Get-Process -Name testbed -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Milliseconds 700
    Write-Output "Shot + closed: $outFile"
}
```

## Running a test

```powershell
# SSH mode (exercises full pipeline)
RunAndShot 'tb-ssh'   '--port 22333 --frames 500' 'C:\Temp\shot-ssh.png'

# SSH mode + ONLCR writer
RunAndShot 'tb-onlcr' '--port 22444 --frames 500 --onlcr' 'C:\Temp\shot-onlcr.png'

# Direct mode — no SSH, baseline (no pipeline artifacts expected)
RunAndShot 'tb-nossh' '--frames 500 --no-ssh' 'C:\Temp\shot-nossh.png'
```

Use different ports for each concurrent instance to avoid `bind: only one
usage of each socket address` errors from a previous run.

## Timing rules

| Delay | Purpose |
|-------|---------|
| 300 ms after launch | Call `BringWTToFront` before bubbletea enters alt screen |
| +1000 ms | SSH connect + first frame render (~2 frames at 100 ms/tick) |
| 700 ms after kill | Let WT close the tab before next `RunAndShot` starts |

If the screenshot is black: the timing is wrong. The most common cause is
calling `BringWTToFront` too late (after alt-screen entry). Probe timing
by taking shots at 1 s, 2 s, 3 s:

```powershell
Start-Process 'wt.exe' -ArgumentList '--window', '0', 'new-tab', '--profile', 'testbed-autoclose', '--title', 'probe', 'cmd', '/c', "`"$testbedPath`" --port 22333 --frames 500"
Start-Sleep -Milliseconds 300; BringWTToFront
Start-Sleep -Milliseconds 1000; Shot 'C:\Temp\p1.png'; Write-Output "1s"
Start-Sleep -Milliseconds 1000; Shot 'C:\Temp\p2.png'; Write-Output "2s"
Start-Sleep -Milliseconds 1000; Shot 'C:\Temp\p3.png'; Write-Output "3s"
Get-Process -Name testbed -ErrorAction SilentlyContinue | Stop-Process -Force
```

## Cleanup

If stale tabs accumulate from a failed run:

```powershell
# Kill process, tab auto-closes
Get-Process -Name testbed -ErrorAction SilentlyContinue | Stop-Process -Force

# If tab is still open (e.g. testbed already exited), close it by
# focusing it by title then issuing closeTab:
# [Microsoft.VisualBasic.Interaction]::AppActivate('tb-ssh') | Out-Null
# Start-Sleep -Milliseconds 300
# Start-Process 'wt.exe' -ArgumentList '--window', '0', 'action', 'closeTab'
```

**Never** use `wt.exe --window new` thinking it creates a new process — WT
is single-instance. `$wtProc.Id` returned by `Start-Process wt.exe -PassThru`
is the launcher stub, not a killable window. The `testbed-autoclose` profile
is the only reliable cleanup mechanism.

## Interpreting screenshots

**What to look for:**

- **Staircase** — rows shift right each line. Cause: bare `\n` without `\r`
  (ONLCR missing). Fix: `--onlcr` flag or `applyONLCR` in `KittyStripWriter`.
- **Colors absent** — text is white/monochrome. Cause: bubbletea color profile
  is ASCII or ANSI; 256-color sequences are stripped. Fix: `tea.WithColorProfile(colorprofile.TrueColor)` in the server middleware.
- **Colors present, alignment correct** — no artifacts; pipeline is clean.
- **Black screen** — screenshot timing is wrong (see Timing rules above).

## Raw output analysis (no terminal needed)

To inspect ANSI sequences without launching a window:

```bash
dist/testbed.exe --port 22555 --frames 20 2>&1 | cat
```

Look for:
- `\x1b[38;5;Nm` — 256-color codes (colors active)
- `\x1b[ND` — cursor-left N (delta renderer moving within a line — correct)
- `\r\n` vs bare `\n` — bare `\n` causes staircase in SSH mode on Unix

## Testbed flags

| Flag | Default | Effect |
|------|---------|--------|
| `--port N` | 22222 | SSH listen port |
| `--frames N` | 0 (run forever) | Quit after N frames (~N×100 ms) |
| `--onlcr` | off | Wrap session output with ONLCRWriter |
| `--no-ssh` | off | Skip SSH; run model directly (baseline) |

## Key pitfalls learned

- `MainWindowHandle` is always 0 for `cmd.exe` — it's a console process.
  Use the `WindowsTerminal.exe` process handle instead.
- `AppActivate(title)` is unreliable when VS Code or another window has
  focus and uses focus-steal prevention. `SetForegroundWindow` on the WT
  `MainWindowHandle` is the reliable path.
- `sess.StdinPipe()` and `sess.StdoutPipe()` must be called **before**
  `sess.Shell()` in `golang.org/x/crypto/ssh`. Calling after returns nil,
  causing a nil-interface panic in `io.Copy`.
- `tea.WithEnvironment(envs)` after `wishbubbletea.MakeOptions(sess)` can
  silently break the program output. Use `tea.WithColorProfile` instead to
  override color detection.
