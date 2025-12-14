param(
    [string]$OutDir = "build\packages",
    [string]$NsisExe = "makensis"
)

$Root = Split-Path -Parent $MyInvocation.MyCommand.Definition
$NsisScript = Join-Path $Root "baudlink.nsi"
$OutFile = Join-Path $OutDir "baudlink-installer.exe"

if (!(Test-Path $OutDir)) { New-Item -ItemType Directory -Path $OutDir -Force | Out-Null }

# Ensure build artifacts exist
if (!(Test-Path "build\baudlink-service.exe") -or !(Test-Path "build\baudlink-cli.exe")) {
    Write-Host "Building binaries..."
    Invoke-Expression "make build"
}

# Run makensis to create installer
if (!(Get-Command $NsisExe -ErrorAction SilentlyContinue)) {
    Write-Error "makensis not found in PATH. Install NSIS and ensure makensis is available."
    exit 1
}

Push-Location $Root
& $NsisExe /INPUTCHARSET UTF8 "$NsisScript"
Pop-Location

# NSIS produces 'BaudLink-installer.exe' (as defined in the .nsi)
$nsisOut = Join-Path $Root "BaudLink-installer.exe"
if (Test-Path $nsisOut) {
    Move-Item -Path $nsisOut -Destination $OutFile -Force
    Write-Host "Created installer at $OutFile"
} else {
    Write-Host "NSIS did not produce expected installer file: $nsisOut"
}
