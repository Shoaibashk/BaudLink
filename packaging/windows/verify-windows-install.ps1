param(
    [string]$InstallerPath = "build\packages\baudlink-installer.exe",
    [string]$ServiceName = "BaudLink",
    [string]$ServiceExe = "build\baudlink-service.exe",
    [string]$TrayArg = "tray",
    [int]$TimeoutSeconds = 20,
    [switch]$NoCleanup,
    [switch]$DryRun,
    [switch]$AutoElevate,
    [string]$LogFile = "packaging\windows\verify-windows-install.log"
)

function Is-Admin {
    $current = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($current)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Write-Log {
    param([string]$msg)
    $timestamp = (Get-Date).ToString("o")
    $line = "$timestamp `t $msg"
    Write-Host $msg
    try { Add-Content -Path $LogFile -Value $line -ErrorAction SilentlyContinue } catch {}
}

function Relaunch-Elevated {
    param([array]$origArgs)
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = (Get-Command pwsh).Source
    $psi.Arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`" $($origArgs -join ' ')"
    $psi.Verb = "runas"
    try {
        [System.Diagnostics.Process]::Start($psi) | Out-Null
        exit 0
    } catch {
        Write-Log "User declined elevation or elevation failed: $_"
        exit 1
    }
}

if (-not (Is-Admin)) {
    if ($AutoElevate) {
        Write-Log "Not elevated; attempting to relaunch as admin..."
        Relaunch-Elevated -origArgs $MyInvocation.BoundParameters.Keys | Out-Null
    }
    Write-Log "This verification script should be run as Administrator. Re-run as Administrator or use -AutoElevate to prompt for elevation.";
    if (-not $DryRun) { exit 1 } else { Write-Log "DryRun mode: continuing without Admin rights" }
}

Write-Log "Starting Windows install & tray verification for service '$ServiceName'"

if ($DryRun) { Write-Log "DryRun enabled: no changes will be made" }

if (!(Test-Path $ServiceExe)) {
    Write-Log "Service binary not found at $ServiceExe; building..."
    if (-not $DryRun) {
        & make build
        if ($LASTEXITCODE -ne 0) { Write-Log "make build failed"; exit 2 }
    } else {
        Write-Log "DryRun: skipping build"
    }
}

$fullExe = (Resolve-Path $ServiceExe).Path

# If an installer exists, prefer it (silent install) otherwise install via sc
if (Test-Path $InstallerPath) {
    Write-Log "Running installer $InstallerPath (silent)..."
    if (-not $DryRun) { & $InstallerPath /S; Start-Sleep -Seconds 2 } else { Write-Log "DryRun: would run installer $InstallerPath /S" }
} else {
    Write-Host "No installer found; installing service manually using sc"
    # Remove existing service if present
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc) {
        Write-Log "Stopping existing service..."
        if (-not $DryRun) { Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue; sc.exe delete "$ServiceName" | Out-Null; Start-Sleep -Seconds 1 } else { Write-Log "DryRun: would stop and delete existing service" }
    }

    $binPathEscaped = $fullExe
    Write-Log "Creating service $ServiceName -> $binPathEscaped"
    if (-not $DryRun) { sc.exe create "$ServiceName" binPath= "\"$binPathEscaped\"" start= auto } else { Write-Log "DryRun: would run sc create for $binPathEscaped" }
}

Write-Log "Starting service $ServiceName"
if (-not $DryRun) { Start-Service -Name $ServiceName } else { Write-Log "DryRun: would start service" }

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
while ((Get-Date) -lt $deadline) {
    $s = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($s -and $s.Status -eq 'Running') { break }
    Start-Sleep -Milliseconds 200
}

$s = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if (-not $s -or $s.Status -ne 'Running') {
    Write-Log "Service $ServiceName did not start successfully"
    if (-not $DryRun) { sc.exe query "$ServiceName" }
    if (-not $DryRun) { exit 3 } else { Write-Log "DryRun: continuing despite service not running" }
}

Write-Log "Service is running. Starting tray application (as current user)"
if (-not $DryRun) { $trayProc = Start-Process -FilePath $fullExe -ArgumentList $TrayArg -PassThru; Start-Sleep -Seconds 2 } else { Write-Log "DryRun: would start tray process: $fullExe $TrayArg" }

# Find the tray process via commandline
$trayFound = Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -and $_.CommandLine -like "* $TrayArg*" }
if (-not $trayFound) {
    Write-Warning "Tray process not detected by CommandLine scan. Checking by process name."
    $trayFound = Get-Process -Name (Split-Path $fullExe -Leaf -Resolve) -ErrorAction SilentlyContinue | Where-Object { $_.Id -eq $trayProc.Id }
}

if (-not $trayFound) {
    Write-Log "Tray process did not start or could not be detected"
    # Cleanup: stop service
    if (-not $NoCleanup -and -not $DryRun) { Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue; sc.exe delete "$ServiceName" | Out-Null }
    if (-not $DryRun) { exit 4 } else { Write-Log "DryRun: continuing despite missing tray" }
}

Write-Log "Tray process started (PID: $($trayProc.Id))"

Write-Log "Testing service control via PowerShell"
if (-not $DryRun) {
    Stop-Service -Name $ServiceName -ErrorAction Stop
    Start-Sleep -Seconds 1
    if ((Get-Service -Name $ServiceName).Status -ne 'Stopped') { Write-Log "Service did not stop as expected"; exit 5 }

    Start-Service -Name $ServiceName
    Start-Sleep -Seconds 1
    if ((Get-Service -Name $ServiceName).Status -ne 'Running') { Write-Log "Service did not start as expected"; exit 6 }
} else {
    Write-Log "DryRun: skipping start/stop checks"
}

Write-Log "Service control verified. Cleaning up: stopping tray and uninstalling service"
if (-not $NoCleanup) {
    if (-not $DryRun) { try { Stop-Process -Id $trayProc.Id -Force -ErrorAction SilentlyContinue } catch {}; Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue; sc.exe delete "$ServiceName" | Out-Null }
    else { Write-Log "DryRun: skipping cleanup" }
} else {
    Write-Log "NoCleanup: leaving service and tray running for inspection"
}

Write-Log "Windows install & tray verification completed successfully"
exit 0
