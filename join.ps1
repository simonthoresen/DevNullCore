$ErrorActionPreference = 'Stop'

# Connection endpoints are passed as a comma-separated list of host:port pairs
# via the NS_ENDPOINTS environment variable. Each is tried in order.
if (-not $env:NS_ENDPOINTS) {
    Write-Host "Error: NS_ENDPOINTS not set. Use the invite command from a null-space server." -ForegroundColor Red
    exit 1
}

$name = $env:USERNAME
$name = Read-Host "Enter your player name (default: $name)"
if ([string]::IsNullOrWhiteSpace($name)) {
    $name = $env:USERNAME
}

$sshOpts = @(
    "-t",
    "-o", "ConnectTimeout=5",
    "-o", "StrictHostKeyChecking=no",
    "-o", "UserKnownHostsFile=/dev/null"
)

$endpoints = $env:NS_ENDPOINTS -split ','
foreach ($ep in $endpoints) {
    $parts = $ep -split ':', 2
    $host  = $parts[0]
    $port  = $parts[1]

    Write-Host "Trying $host`:$port ..." -ForegroundColor DarkGray
    ssh @sshOpts -p $port "${name}@${host}"
    if ($LASTEXITCODE -eq 0) { exit 0 }
}

Write-Host "Could not connect to any endpoint." -ForegroundColor Red
exit 1
