param(
    [string]$Game = $(if ($args.Count -ge 1) { $args[0] } else { "towerdefense" }),
    [string]$Password = $(if ($args.Count -ge 2) { $args[1] } else { "changeme" })
)

$root = Split-Path -Parent $PSScriptRoot
$script:tunnelShell = $null
$script:tunnelWatcher = $null

function Stop-Tunnel {
    if ($script:tunnelShell) {
        $childProcesses = Get-CimInstance Win32_Process -Filter "ParentProcessId = $($script:tunnelShell.Id)" -ErrorAction SilentlyContinue
        foreach ($child in $childProcesses) {
            Stop-Process -Id $child.ProcessId -Force -ErrorAction SilentlyContinue
        }

        Stop-Process -Id $script:tunnelShell.Id -Force -ErrorAction SilentlyContinue
    }
}

function Stop-TunnelWatcher {
    if ($script:tunnelWatcher) {
        Stop-Job -Job $script:tunnelWatcher -ErrorAction SilentlyContinue
        Remove-Job -Job $script:tunnelWatcher -Force -ErrorAction SilentlyContinue
    }
}

function Start-TunnelWatcher {
    param(
        [int]$TunnelShellPid,
        [int]$ConsoleShellPid
    )

    $script:tunnelWatcher = Start-Job -ScriptBlock {
        param(
            [int]$ObservedTunnelPid,
            [int]$ObservedConsolePid
        )

        function Get-DescendantProcesses {
            param([int]$RootPid)

            $all = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue
            if (-not $all) {
                return @()
            }

            $childrenByParent = @{}
            foreach ($proc in $all) {
                if (-not $childrenByParent.ContainsKey($proc.ParentProcessId)) {
                    $childrenByParent[$proc.ParentProcessId] = @()
                }
                $childrenByParent[$proc.ParentProcessId] += $proc
            }

            $queue = [System.Collections.Generic.Queue[object]]::new()
            $results = [System.Collections.Generic.List[object]]::new()
            $queue.Enqueue($RootPid)

            while ($queue.Count -gt 0) {
                $parentPid = [int]$queue.Dequeue()
                foreach ($child in ($childrenByParent[$parentPid] | Select-Object -Unique)) {
                    $results.Add($child) | Out-Null
                    $queue.Enqueue([int]$child.ProcessId)
                }
            }

            return $results
        }

        while ($true) {
            Start-Sleep -Milliseconds 500

            if (-not (Get-Process -Id $ObservedTunnelPid -ErrorAction SilentlyContinue)) {
                $targets = Get-DescendantProcesses -RootPid $ObservedConsolePid | Where-Object {
                    $_.Name -in @('go.exe', 'null-space.exe')
                }

                foreach ($target in $targets) {
                    Stop-Process -Id $target.ProcessId -Force -ErrorAction SilentlyContinue
                }

                break
            }

            if (-not (Get-Process -Id $ObservedConsolePid -ErrorAction SilentlyContinue)) {
                break
            }
        }
    } -ArgumentList $TunnelShellPid, $ConsoleShellPid
}

$existingListener = Get-NetTCPConnection -LocalPort 23234 -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if ($existingListener) {
    Write-Error "Port 23234 is already in use by PID $($existingListener.OwningProcess). Stop that process or use a different listen port before starting null-space."
    exit 1
}

$tunnelCommand = @'
$Host.UI.RawUI.WindowTitle = "null-space Pinggy Tunnel"
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host "            NULL-SPACE PINGGY TUNNEL         " -ForegroundColor Black -BackgroundColor Green
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host "Keep this window open while the server is running." -ForegroundColor Cyan
Write-Host "Copy the tcp://... address shown below and give it to players." -ForegroundColor Cyan
Write-Host "If prompted for a password, press Enter." -ForegroundColor Cyan
Write-Host ""
ssh -p 443 -o ServerAliveInterval=30 -o StrictHostKeyChecking=accept-new -o ExitOnForwardFailure=yes -R0:127.0.0.1:23234 tcp@a.pinggy.io
'@

$encodedTunnelCommand = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($tunnelCommand))

Write-Host "Opening Pinggy tunnel window..." -ForegroundColor Cyan
$script:tunnelShell = Start-Process -FilePath "pwsh" `
    -ArgumentList @("-NoExit", "-EncodedCommand", $encodedTunnelCommand) `
    -WorkingDirectory $root `
    -PassThru

Start-TunnelWatcher -TunnelShellPid $script:tunnelShell.Id -ConsoleShellPid $PID

Clear-Host
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host "                 LOBBY OPEN                  " -ForegroundColor Black -BackgroundColor Green
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host "Game:      $Game"
Write-Host "Tunnel:    a separate PowerShell window is showing the Pinggy tcp:// address"
Write-Host "Tunnel PID: $($script:tunnelShell.Id)"
Write-Host ""
Write-Host "Local admin console is live in this terminal." -ForegroundColor Cyan
Write-Host "Type chat text to broadcast globally, or use /commands as admin." -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop both the server and the tunnel." -ForegroundColor Cyan
Write-Host ""

$serverExitCode = 0

Push-Location $root
try {
    & go run ./cmd/null-space --game $Game --password $Password
    if ($LASTEXITCODE) {
        $serverExitCode = $LASTEXITCODE
    }
}
finally {
    Pop-Location
    Stop-TunnelWatcher
    Stop-Tunnel
}

if ($serverExitCode -ne 0) {
    exit $serverExitCode
}