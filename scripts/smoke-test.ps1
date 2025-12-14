param(
    [int]$Port = 50055,
    [int]$WaitSeconds = 10
)

$servicePath = Join-Path (Get-Location) "build\baudlink-service.exe"
$cliPath = Join-Path (Get-Location) "build\baudlink-cli.exe"

if (!(Test-Path $servicePath)) { Write-Error "Service binary not found: $servicePath"; exit 2 }
if (!(Test-Path $cliPath)) { Write-Error "CLI binary not found: $cliPath"; exit 2 }

$address = "localhost:$Port"

# Start service
$proc = Start-Process -FilePath $servicePath -ArgumentList "serve","--address",$address,"--debug" -PassThru -NoNewWindow
Write-Host "Started service PID $($proc.Id) on $address, waiting for ready..."

$start = Get-Date
while ((Get-Date) - $start -lt (New-TimeSpan -Seconds $WaitSeconds)) {
    try {
        # try info as readiness probe
        $out = & $cliPath --address $address info --json 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Service ready."
            break
        }
    } catch { }
    Start-Sleep -Milliseconds 200
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "Service didn't become ready within $WaitSeconds seconds"
    Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
    exit 3
}

# Run commands
Write-Host "Running scan..."
$scanOut = & $cliPath --address $address scan --json
if ($scanOut -eq 'null' -or [string]::IsNullOrEmpty($scanOut)) {
    Write-Host "No ports found"
    $ports = @()
} else {
    try {
        $ports = $scanOut | ConvertFrom-Json
    } catch {
        Write-Host "Scan output not JSON: $scanOut"
        $ports = @()
    }
}
Write-Host "Found $($ports.Length) ports."
if ($ports.Length -gt 0) {
    $first = $ports[0]
    Write-Host "Found port: $($first.name -or $first) - skipping open/status tests that require hardware access."
}
Write-Host "Running info..."
$infoOut = & $cliPath --address $address info --json
if ($LASTEXITCODE -ne 0) {
    Write-Host "info command failed with exit code $LASTEXITCODE"
    Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
    exit 4
}

try {
    $info = $infoOut | ConvertFrom-Json
} catch {
    Write-Host "info output not JSON: $infoOut"
    Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
    exit 5
}

if ($info.config.grpc_address -ne $address) {
    Write-Host "info.config.grpc_address mismatch: expected $address got $($info.config.grpc_address)"
    Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
    exit 6
}

if (-not ($info.supported_features -contains 'grpc')) {
    Write-Host "info.supported_features does not contain 'grpc'"
    Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
    exit 7
}

# Cleanup
Write-Host "Stopping service PID $($proc.Id)"
Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue

Write-Host "Smoke test completed."
exit 0
