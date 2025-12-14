# Windows Verification Guide

This guide explains how to verify that the Windows installer registers the `BaudLink` service correctly and that the tray application starts and interacts with the service.

## Prerequisites
- Windows 10 or later
- Administrator privileges for installing and managing services
- NSIS (`makensis`) if you want to build the installer from source
- PowerShell 7+ recommended (or Windows PowerShell works too)

## Quick verification (recommended)
1. Build the binaries:

```powershell
make build
```

2. Option A — Run the installer (recommended):

- Build the NSIS installer with `make package-windows` (requires `makensis`).
- Run the generated installer (`build\packages\baudlink-installer.exe`) as Administrator.

3. Option B — Manual install (no installer required):

```powershell
# Run in Admin PowerShell
sc create "BaudLink" binPath= """C:\path\to\build\baudlink-service.exe""" start= auto
Start-Service BaudLink
```

4. Start the tray application (as the current user):

```powershell
Start-Process -FilePath "C:\path\to\build\baudlink-service.exe" -ArgumentList tray
```

5. Verify behaviour:
- Use `Get-Service BaudLink` to verify status
- Check Event Viewer / Application logs for service output
- Look for the tray icon in the notification area

## Automated verification script
We've added a verification script that automates the above steps:

- `packaging\windows\verify-windows-install.ps1`

Usage examples:

- Dry run (no changes):
```powershell
pwsh -NoProfile -File packaging\windows\verify-windows-install.ps1 -DryRun
```

- Auto-elevate and run (prompts for UAC):
```powershell
pwsh -NoProfile -File packaging\windows\verify-windows-install.ps1 -AutoElevate
```

- Run and keep installed (no cleanup):
```powershell
pwsh -NoProfile -File packaging\windows\verify-windows-install.ps1 -NoCleanup -AutoElevate
```

- Collect logs to a file:
```powershell
pwsh -NoProfile -File packaging\windows\verify-windows-install.ps1 -AutoElevate -LogFile C:\temp\baudlink-verify.log
```

The script exits with a non-zero status code when verification fails. When run with `-NoCleanup`, it leaves the service and tray running so you can inspect them interactively.
