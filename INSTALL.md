# BaudLink Installation Guide

## Windows Installation

### Prerequisites
- Windows 10 or later
- Administrator privileges (for service installation)

### Installation Steps

1. **Download the release**
   - Download `BaudLink_Windows_x86_64.zip` from the releases page
   - Extract to a folder (e.g., `C:\Program Files\BaudLink`)

2. **Verify files**
   After extraction, you should have:
   ```
   baudlink.exe       (CLI application)
   baudlink-tray.exe  (System tray GUI)
   README.md
   LICENSE
   ```

3. **Install as Windows Service** (Run as Administrator)
   ```powershell
   # Open PowerShell as Administrator
   cd "C:\Program Files\BaudLink"
   
   # Install the service (will auto-start service and tray)
   .\baudlink.exe service install
   ```
   
   This single command will:
   - Install the BaudLink Windows service
   - Start the service automatically
   - Launch the system tray application
   
   Expected output:
   ```
   Service baudlink installed successfully
   Starting service...
   Service baudlink started successfully
   System tray application started
   ```

4. **Check status**
   ```powershell
   .\baudlink.exe service status
   ```
   
   Expected output:
   ```
   Service baudlink: Running
   System Tray: Running
   ```

5. **System Tray**
   - After service starts, look for the BaudLink icon in your system tray (bottom-right)
   - Click the icon to:
     - View available serial ports
     - Check service status
     - Stop/start the service
     - Access other features

### Management Commands

```powershell
# Install service (auto-starts service + tray)
.\baudlink.exe service install

# Manually start service + tray (if stopped)
.\baudlink.exe service start

# Stop tray + service
.\baudlink.exe service stop

# Check status
.\baudlink.exe service status

# Uninstall
.\baudlink.exe service uninstall
```

### Manual Tray Launch

If you want to run only the system tray (without installing the service):
```powershell
.\baudlink-tray.exe
```

### CLI Commands

```powershell
# Scan for available ports
.\baudlink.exe scan

# Run server directly (without service)
.\baudlink.exe serve

# Show version
.\baudlink.exe version

# Show help
.\baudlink.exe help
```

## Troubleshooting

### "Access is denied" error
- Service commands require Administrator privileges
- Right-click PowerShell and select "Run as Administrator"

### System tray not appearing
1. Check if it's hidden in the overflow area (click the `^` icon)
2. Verify the service is running: `.\baudlink.exe service status`
3. Manually start the tray: `.\baudlink-tray.exe`

### Tray executable not found
- Ensure both `baudlink.exe` and `baudlink-tray.exe` are in the same directory
- When using `service start`, the CLI will search for the tray executable in:
  - Same directory as baudlink.exe
  - Parent directory (for development builds)
  - System PATH

### Service won't start
1. Check if another instance is running
2. Check Windows Event Viewer for detailed error logs
3. Verify the configuration file exists

## Uninstallation

```powershell
# Run as Administrator
.\baudlink.exe service stop
.\baudlink.exe service uninstall

# Then delete the BaudLink folder
```

## Development Build

For developers building from source:

```powershell
# Build both executables
make build

# Both files will be in the build/ directory:
# - build/baudlink.exe
# - build/baudlink-tray.exe

# Test service commands (requires admin)
.\build\baudlink.exe service start
```

## Packaging / Installer

You can create platform installers for distribution using the provided packaging scripts and Makefile targets.

### Linux (systemd tarball)

```bash
# Build and create a tar.gz package containing the service binary, systemd unit, and installer
make package-linux

# Extract and install (as root)
tar xzf build/packages/baudlink_linux_$(uname -m).tar.gz -C /tmp
cd /tmp/baudlink_linux_$(uname -m)
sudo ./install.sh
```

This will install `/usr/local/bin/baudlink-service` and `/etc/systemd/system/baudlink.service`, then enable and start the unit.

### Windows (NSIS installer)

Requirements: NSIS (makensis) must be installed and available on PATH.

```powershell
# Build and create a Windows installer (requires makensis)
make package-windows

# Run the generated installer (build/packages/baudlink-installer.exe)
./build/packages/baudlink-installer.exe
```

The installer will place `baudlink-service.exe` and `baudlink-cli.exe` under `C:\Program Files\BaudLink` and register a Windows service named `BaudLink`.

## Smoke tests & CI ✅

- PowerShell smoke test (Windows): `scripts/smoke-test.ps1` — starts a local `baudlink-service`, runs a few CLI commands, and stops the service. Example:

```powershell
pwsh -NoProfile -File .\scripts\smoke-test.ps1 -Port 50055
```

- Cross-platform Bash smoke test (Linux/macOS): `scripts/smoke-test.sh` (requires `jq`):

```bash
bash ./scripts/smoke-test.sh 50055
```

- CI: GitHub Actions workflow is configured in `.github/workflows/ci.yml` to build both binaries, run the smoke tests on Ubuntu and Windows runners, and execute `go test ./...`.
