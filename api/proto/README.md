# BaudLink Proto Definitions

This directory contains the Protocol Buffer definitions for the BaudLink gRPC API. These protos can be used as a **git submodule** to share contracts across multiple projects (e.g., UI clients, mobile apps, or other services).

## Overview

The `serial.proto` file defines the `SerialService` gRPC service and all related message types for:

- **Port Discovery** - List and get info about serial ports
- **Port Management** - Open, close, and configure ports
- **Data Transfer** - Read and write data
- **Streaming** - Real-time bidirectional data streaming
- **Health & Diagnostics** - Ping and agent info

## Using as a Git Submodule

### Adding the Submodule to Your Project

If this proto directory has been extracted to a separate repository (e.g., `BaudLink-protos`), you can add it as a submodule:

```bash
# Add the submodule to your project
git submodule add https://github.com/Shoaibashk/BaudLink-protos.git protos/baudlink

# Initialize and fetch the submodule
git submodule update --init --recursive
```

### Updating the Submodule

```bash
# Navigate to the submodule directory
cd protos/baudlink

# Pull the latest changes
git pull origin main

# Go back to your project root and commit the update
cd ../..
git add protos/baudlink
git commit -m "Update BaudLink protos"
```

## Generating Client Code

### Prerequisites

Install the Protocol Buffers compiler and language-specific plugins:

```bash
# Install protoc (Protocol Buffers compiler)
# macOS
brew install protobuf

# Ubuntu/Debian
sudo apt install protobuf-compiler

# Windows (using Chocolatey)
choco install protoc
```

### Using Buf (Recommended)

[Buf](https://buf.build/) is the recommended tool for working with Protocol Buffers.

```bash
# Install buf
# macOS
brew install bufbuild/buf/buf

# Other platforms: https://docs.buf.build/installation
```

Create a `buf.gen.yaml` in your project:

```yaml
version: v2
plugins:
  # For TypeScript/JavaScript (Connect-Web)
  - remote: buf.build/connectrpc/es
    out: src/gen
    opt: target=ts

  # For TypeScript/JavaScript (gRPC-Web)
  - remote: buf.build/grpc/web
    out: src/gen
    opt: import_style=typescript

  # For Python
  - remote: buf.build/protocolbuffers/python
    out: src/gen
  - remote: buf.build/grpc/python
    out: src/gen

  # For Go
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
```

Generate the code:

```bash
buf generate protos/baudlink
```

### Manual Generation with protoc

#### TypeScript/JavaScript (gRPC-Web)

```bash
# Install gRPC-Web plugin
npm install -g grpc-web

# Generate
protoc -I=protos/baudlink \
  --js_out=import_style=commonjs:src/gen \
  --grpc-web_out=import_style=typescript,mode=grpcwebtext:src/gen \
  protos/baudlink/serial.proto
```

#### TypeScript/JavaScript (Connect-RPC)

```bash
# Install dependencies
npm install @bufbuild/buf @bufbuild/protoc-gen-es @connectrpc/protoc-gen-connect-es

# Generate using buf
npx buf generate protos/baudlink
```

#### Python

```bash
# Install gRPC tools
pip install grpcio-tools

# Generate
python -m grpc_tools.protoc -I=protos/baudlink \
  --python_out=src/gen \
  --grpc_python_out=src/gen \
  protos/baudlink/serial.proto
```

#### C# / .NET

```bash
# Install Grpc.Tools NuGet package
dotnet add package Grpc.Tools

# Generate (typically handled by .csproj)
protoc -I=protos/baudlink \
  --csharp_out=src/Gen \
  --grpc_csharp_out=src/Gen \
  protos/baudlink/serial.proto
```

#### Dart/Flutter

```bash
# Install protoc plugin
dart pub global activate protoc_plugin

# Generate
protoc -I=protos/baudlink \
  --dart_out=grpc:lib/src/gen \
  protos/baudlink/serial.proto
```

## Example Usage

### TypeScript (Connect-RPC)

```typescript
import { createClient } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { SerialService } from "./gen/serial_connect";

const transport = createGrpcWebTransport({
  baseUrl: "http://localhost:50051",
});

const client = createClient(SerialService, transport);

// List ports
const response = await client.listPorts({});
for (const port of response.ports) {
  console.log(`${port.name}: ${port.description}`);
}
```

### Python

```python
import grpc
from serial_pb2 import ListPortsRequest
from serial_pb2_grpc import SerialServiceStub

channel = grpc.insecure_channel('localhost:50051')
stub = SerialServiceStub(channel)

# List ports
response = stub.ListPorts(ListPortsRequest())
for port in response.ports:
    print(f"{port.name}: {port.description}")
```

## Proto File Structure

```
.
├── README.md              # This file
├── buf.yaml               # Buf configuration for this module
├── serial.proto           # Main proto definitions
├── serial.pb.go           # Generated Go code (for BaudLink server)
├── serial_grpc.pb.go      # Generated Go gRPC code
└── examples/              # Example buf.gen.yaml files
    ├── buf.gen.typescript.yaml  # TypeScript/Connect-RPC
    ├── buf.gen.grpc-web.yaml    # gRPC-Web
    ├── buf.gen.python.yaml      # Python
    └── buf.gen.dart.yaml        # Dart/Flutter
```

## API Reference

### Service Definition

```protobuf
service SerialService {
  // Port Discovery
  rpc ListPorts(ListPortsRequest) returns (ListPortsResponse);
  rpc GetPortInfo(GetPortInfoRequest) returns (PortInfo);
  
  // Port Management
  rpc OpenPort(OpenPortRequest) returns (OpenPortResponse);
  rpc ClosePort(ClosePortRequest) returns (ClosePortResponse);
  rpc GetPortStatus(GetPortStatusRequest) returns (PortStatus);
  
  // Data Transfer
  rpc Write(WriteRequest) returns (WriteResponse);
  rpc Read(ReadRequest) returns (ReadResponse);
  
  // Streaming
  rpc StreamRead(StreamReadRequest) returns (stream DataChunk);
  rpc StreamWrite(stream DataChunk) returns (StreamWriteResponse);
  rpc BiDirectionalStream(stream DataChunk) returns (stream DataChunk);
  
  // Port Configuration
  rpc ConfigurePort(ConfigurePortRequest) returns (ConfigurePortResponse);
  rpc GetPortConfig(GetPortConfigRequest) returns (PortConfig);
  
  // Health & Diagnostics
  rpc Ping(PingRequest) returns (PingResponse);
  rpc GetAgentInfo(GetAgentInfoRequest) returns (AgentInfo);
}
```

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
| `Ping` | Health check |
| `GetAgentInfo` | Get agent version and info |

For complete API documentation, see the main [BaudLink repository](https://github.com/Shoaibashk/BaudLink).

## License

This project is licensed under the Apache License 2.0.
