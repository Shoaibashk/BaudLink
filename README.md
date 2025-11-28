# BaudLink

<div align="center">

![BaudLink Logo](https://img.shields.io/badge/BaudLink-Serial%20Agent-blue?style=for-the-badge&logo=go)

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/Shoaibashk/BaudLink/build.yml?style=flat-square)](https://github.com/Shoaibashk/BaudLink/actions)
[![Release](https://img.shields.io/github/v/release/Shoaibashk/BaudLink?style=flat-square)](https://github.com/Shoaibashk/BaudLink/releases)

**Cross-platform Serial Port Background Service**

[Features](#features) • [Installation](#installation) • [Quick Start](#quick-start) • [API Documentation](docs/API.md) • [Security](docs/SECURITY.md)

</div>

---

## Overview

BaudLink is a **cross-platform serial port background service** that runs on Windows, Linux, and Raspberry Pi. It manages all serial hardware and exposes a public gRPC API for any client - Python, C#, Node.js, Web, Mobile, or CLI.

```text
           Any Client (Python | C# | Web | Mobile | CLI)
                              |
                              | gRPC / WebSocket
                              |
                    +-------------------+
                    |   BaudLink Agent  |
                    | (Background Svc)  |
                    +-------------------+
                              |
                              | USB / COM / UART
                              |
                       Hardware Devices
```

**No UI. No frontend. Just a rock-solid hardware agent.** 💪

## Features

### 🔌 Serial Port Management

- **Auto-detect ports** - Discover all USB, native, Bluetooth, and virtual serial ports
- **Open/Close** - Manage port lifecycle with exclusive locking
- **Read/Write** - Send and receive data with timeout support
- **Streaming** - Real-time bidirectional data streaming
- **Hot-plug support** - Detect port changes on the fly

### 🌐 Network API

- **gRPC API** - High-performance, strongly-typed API
- **Streaming support** - Server, client, and bidirectional streaming
- **Cross-language** - Use from any language with gRPC support

### 🔐 Security

- **TLS encryption** - Secure transport layer
- **Port locking** - Exclusive access control
- **Network binding** - Control service exposure

### ⚙️ System Integration

- **Windows Service** - Run as Windows background service
- **systemd service** - Run as Linux/Raspberry Pi daemon
- **Auto-start** - Start on system boot
- **Logging** - Comprehensive audit logging

## Installation

### From Releases

Download the latest release for your platform:

```bash
# Linux/macOS
curl -LO https://github.com/Shoaibashk/BaudLink/releases/latest/download/baudlink_linux_amd64.tar.gz
tar xzf baudlink_linux_amd64.tar.gz
sudo mv baudlink /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/Shoaibashk/BaudLink/releases/latest/download/baudlink_windows_amd64.zip -OutFile baudlink.zip
Expand-Archive baudlink.zip -DestinationPath C:\Program Files\BaudLink
```

### From Source

```bash
# Clone the repository
git clone https://github.com/Shoaibashk/BaudLink.git
cd BaudLink

# Build (outputs to build/ directory)
make build

# Or build directly
go build -o build/baudlink .

# Install globally
go install .
```

### Using Go Install

```bash
go install github.com/Shoaibashk/BaudLink@latest
```

## Quick Start

### 1. Scan for Serial Ports

```bash
baudlink scan
```

Output:

```text
Found 2 serial port(s):

  COM3 - USB Serial Device
  COM4 - Arduino Uno
```

### 2. Start the Agent

```bash
# Run in foreground
baudlink serve

# With custom config
baudlink serve --config ./config/agent.yaml

# With custom address
baudlink serve --address 0.0.0.0:50051
```

### 3. Connect from a Client

**Python:**

```python
import grpc
from serial_pb2 import ListPortsRequest
from serial_pb2_grpc import SerialServiceStub

channel = grpc.insecure_channel('localhost:50051')
stub = SerialServiceStub(channel)

# List ports
ports = stub.ListPorts(ListPortsRequest())
for port in ports.ports:
    print(f"{port.name}: {port.description}")
```

**Go:**

```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := pb.NewSerialServiceClient(conn)

resp, _ := client.ListPorts(context.Background(), &pb.ListPortsRequest{})
for _, port := range resp.Ports {
    fmt.Printf("%s: %s\n", port.Name, port.Description)
}
```

## Running as a Service

### Windows

```powershell
# Install the service
baudlink service install

# Start the service
baudlink service start

# Check status
baudlink service status

# Stop the service
baudlink service stop

# Uninstall
baudlink service uninstall
```

### Linux / Raspberry Pi

```bash
# Install the service (requires sudo)
sudo baudlink service install

# Start the service
sudo systemctl start baudlink

# Enable on boot
sudo systemctl enable baudlink

# Check status
sudo systemctl status baudlink

# View logs
sudo journalctl -u baudlink -f
```

## Configuration

Configuration file location:

- **Windows:** `C:\ProgramData\BaudLink\agent.yaml`
- **Linux/macOS:** `/etc/baudlink/agent.yaml`

Generate a default config:

```bash
baudlink config init
```

### Example Configuration

```yaml
server:
  grpc_address: "0.0.0.0:50051"
  max_connections: 100

tls:
  enabled: false
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"

serial:
  defaults:
    baud_rate: 9600
    data_bits: 8
    stop_bits: 1
    parity: "none"
  scan_interval: 5

logging:
  level: "info"
  format: "json"
  file: "/var/log/baudlink/agent.log"
```

## Project Structure

```text
BaudLink/
├── api/
│   ├── grpc_server.go     # gRPC implementation
│   └── proto/
│       ├── README.md       # Proto documentation (for submodule use)
│       ├── buf.yaml        # Buf configuration (for submodule use)
│       ├── serial.proto    # gRPC definitions
│       ├── serial.pb.go    # Generated protobuf code
│       ├── serial_grpc.pb.go
│       └── examples/       # Example buf.gen.yaml files for clients
├── cmd/
│   ├── root.go            # Root command
│   ├── serve.go           # Serve command
│   ├── scan.go            # Scan command
│   ├── version.go         # Version command
│   └── service_*.go       # Service management
├── config/
│   ├── config.go          # Config loading
│   └── agent.yaml         # Example config
├── internal/
│   └── serial/
│       ├── scanner.go     # Port discovery
│       ├── manager.go     # Port management
│       └── reader.go      # Continuous reading
├── service/
│   ├── windows.go         # Windows service
│   └── systemd.go         # Linux service
├── tools/
│   └── grpcclient/        # Test client
├── docs/
│   ├── API.md             # API documentation
│   └── SECURITY.md        # Security guide
├── build/                  # Build output directory
├── main.go                # Entry point
├── Makefile               # Build automation
├── go.mod
└── README.md
```

## API Reference

See [API Documentation](docs/API.md) for complete gRPC API reference.

### Key Operations

| Operation | Description |
|-----------|-------------|
| `ListPorts` | Discover all serial ports |
| `OpenPort` | Open a port with configuration |
| `ClosePort` | Close an open port |
| `Write` | Write data to a port |
| `Read` | Read data from a port |
| `StreamRead` | Stream incoming data |
| `StreamWrite` | Stream outgoing data |
| `BiDirectionalStream` | Full-duplex streaming |

## Development

### Prerequisites

- Go 1.22 or later
- Protocol Buffers compiler (`protoc`)
- gRPC Go plugins

### Building

```bash
# Install dependencies
go mod download

# Generate proto files (if protoc is installed)
make proto

# Build (outputs to build/ directory)
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint
```

### Testing with a Virtual Port

On Linux, you can create a virtual serial port pair:

```bash
socat -d -d pty,raw,echo=0 pty,raw,echo=0
```

## Using Proto Definitions in Your Project

The proto definitions can be used as a **git submodule** to share contracts with UI projects, mobile apps, or other services.

### Option 1: Using the Proto Directory Directly

If the proto directory has been extracted to a separate repository (e.g., `BaudLink-protos`):

```bash
# Add as a git submodule
git submodule add https://github.com/Shoaibashk/BaudLink-protos.git protos/baudlink
git submodule update --init --recursive
```

### Option 2: Using Buf to Generate Client Code

With [Buf](https://buf.build/), you can easily generate client code for various languages:

```bash
# Install buf
brew install bufbuild/buf/buf  # macOS
# See https://docs.buf.build/installation for other platforms

# Generate TypeScript client code
buf generate protos/baudlink
```

Example `buf.gen.yaml` for a TypeScript project:

```yaml
version: v2
inputs:
  - directory: protos/baudlink
plugins:
  - remote: buf.build/bufbuild/es
    out: src/gen
    opt: target=ts
  - remote: buf.build/connectrpc/es
    out: src/gen
    opt: target=ts
```

See the [Proto README](api/proto/README.md) for complete documentation and examples for other languages (Python, C#, Dart, etc.).

## Contributing

Contributions are welcome! Please see our contributing guidelines.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [go-serial](https://github.com/bugst/go-serial) - Cross-platform serial library
- [gRPC-Go](https://github.com/grpc/grpc-go) - gRPC for Go
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management

## Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/Shoaibashk/BaudLink/issues)
- 💬 [Discussions](https://github.com/Shoaibashk/BaudLink/discussions)

---

<div align="center">

Made with ❤️ by [Shoaibashk](https://github.com/Shoaibashk)

</div>

### Project Structure

```
.
├── api/           # gRPC server and protobuf definitions
├── cmd/           # CLI commands
├── config/        # Configuration loading
├── internal/      # Internal packages (serial port handling)
├── service/       # System service wrappers
├── tools/         # Development tools (gRPC test client)
├── docs/          # Documentation
├── build/         # Build output (gitignored)
├── main.go        # Application entry point
├── Makefile       # Build automation
└── go.mod         # Go module definition
```

## Release

Releases are automated using GoReleaser. To create a new release:

1. Tag a new version: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`
3. GitHub Actions will automatically build and publish the release

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
